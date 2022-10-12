package execute

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

const (
	verbsDropdownCommand             = "kc-cmd-builder --verbs"
	resourceTypesDropdownCommand     = "kc-cmd-builder --resource-type"
	resourceNamesDropdownCommand     = "kc-cmd-builder --resource-name"
	resourceNamespaceDropdownCommand = "kc-cmd-builder --namespace"
	kubectlCommandName               = "kubectl"
	dropdownItemsLimit               = 100
	noKubectlCommandsInChannel       = "No `kubectl` commands are enabled in this channel. To learn how to enable them, visit https://botkube.io/docs/configuration/executor"
	kubectlMissingCommandMsg         = "Please specify the kubectl command"
)

var knownCmdPrefix = map[string]struct{}{
	verbsDropdownCommand:             {},
	resourceTypesDropdownCommand:     {},
	resourceNamesDropdownCommand:     {},
	resourceNamespaceDropdownCommand: {},
}

var errRequiredVerbDropdown = errors.New("verbs dropdown select cannot be empty")

type (
	kcMerger interface {
		MergeAllEnabled(includeBindings []string) kubectl.EnabledKubectl
	}
	kcExecutor interface {
		Execute(bindings []string, command string, isAuthChannel bool) (string, error)
	}
	// NamespaceLister provides an option to list all namespaces in a given cluster.
	NamespaceLister interface {
		List(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error)
	}
)

// KubectlCmdBuilder provides functionality to handle interactive kubectl command selection.
type KubectlCmdBuilder struct {
	log             logrus.FieldLogger
	kcExecutor      kcExecutor
	merger          kcMerger
	namespaceLister NamespaceLister
	commandGuard    FakeCommandGuard
}

// NewKubectlCmdBuilder returns a new KubectlCmdBuilder instance.
func NewKubectlCmdBuilder(log logrus.FieldLogger, merger kcMerger, executor kcExecutor, namespaceLister NamespaceLister) *KubectlCmdBuilder {
	return &KubectlCmdBuilder{
		log:             log,
		kcExecutor:      executor,
		merger:          merger,
		namespaceLister: namespaceLister,
		commandGuard:    FakeCommandGuard{},
	}
}

// CanHandle returns true if it's allowed kubectl command that can be handled by this executor.
func (e *KubectlCmdBuilder) CanHandle(args []string) bool {
	// if we know the command prefix, we can handle that command :)
	return e.GetCommandPrefix(args) != ""
}

// GetCommandPrefix returns the command prefix only if it's known.
func (e *KubectlCmdBuilder) GetCommandPrefix(args []string) string {
	switch len(args) {
	case 0:
		return ""

	case 1:
		//  check if it's only a kubectl command without arguments
		if slices.Contains(kubectlAlias, args[0]) {
			return args[0]
		}
		// it is a single arg which cannot start the kubectl command builder
		return ""

	default:
		// check if request comes from command builder message
		gotCmd := fmt.Sprintf("%s %s", args[0], args[1])
		if _, found := knownCmdPrefix[gotCmd]; found {
			return gotCmd
		}
		return ""
	}
}

// Do executes a given kc-cmd-builder command based on args.
//
// TODO: once we will have a real use-case, we should abstract the Slack state and introduce our own model.
func (e *KubectlCmdBuilder) Do(ctx context.Context, args []string, platform config.CommPlatformIntegration, bindings []string, state *slack.BlockActionStates, botName string) (interactive.Message, error) {
	var empty interactive.Message

	if platform != config.SocketSlackCommPlatformIntegration {
		e.log.Debug("Interactive kubectl command builder is not supported on %s platform", platform)
		return interactive.Message{
			Base: interactive.Base{
				Body: interactive.Body{
					Plaintext: kubectlMissingCommandMsg,
				},
			},
		}, nil
	}

	allVerbs, allTypes, defaultNs := e.getEnableKubectlDetails(bindings)
	if len(allVerbs) == 0 {
		return e.noVerbsAvailableInChannelMessage()
	}

	allVerbs = e.commandGuard.FilterSupportedVerbs(allVerbs)

	// if only command name was specified, return initial command builder message
	if len(args) == 1 {
		return e.initialMessage(botName, allVerbs)
	}

	stateDetails := e.extractStateDetails(botName, state)
	if stateDetails.namespace == "" {
		stateDetails.namespace = defaultNs
	}

	var (
		cmdName = args[0]
		cmdVerb = args[1]
		cmd     = fmt.Sprintf("%s %s", cmdName, cmdVerb)
	)

	cmds := executorsRunner{
		verbsDropdownCommand: func() (interactive.Message, error) {
			return e.renderMessage(ctx, botName, stateDetails, true, bindings, allVerbs, allTypes)
		},
		resourceTypesDropdownCommand: func() (interactive.Message, error) {
			// the resource type was selected, so clear resource name from command preview.
			return e.renderMessage(ctx, botName, stateDetails, false, bindings, allVerbs, allTypes)
		},
		resourceNamesDropdownCommand: func() (interactive.Message, error) {
			// this is called only when the resource name is directly selected from dropdown, so we need to include
			// it in command preview.
			return e.renderMessage(ctx, botName, stateDetails, true, bindings, allVerbs, allTypes)
		},
		resourceNamespaceDropdownCommand: func() (interactive.Message, error) {
			// when the namespace was changed, there is a small chance that resource name will be still matching,
			// we will need to do the external call to check that. For now, we clear resource name from command preview.
			return e.renderMessage(ctx, botName, stateDetails, false, bindings, allVerbs, allTypes)
		},
	}

	msg, err := cmds.SelectAndRun(cmd)
	if err != nil {
		return empty, err
	}
	return msg, nil
}

func (e *KubectlCmdBuilder) initialMessage(botName string, allVerbs []string) (interactive.Message, error) {
	var empty interactive.Message

	// We start a new interactive block, so we generate unique ID.
	// Later when we update this message with a new "body" e.g. update command preview
	// the block state remains the same as Slack always see it under the same id.
	// If we use different ID each time we update the message, Slack will clean up the state
	// meaning we will lose information about verb/resourceType/resourceName that were previously selected.
	id, err := uuid.NewRandom()
	if err != nil {
		return empty, err
	}
	allVerbsSelect := VerbSelect(botName, allVerbs, "")
	if allVerbsSelect == nil {
		return empty, errRequiredVerbDropdown
	}
	return KubectlCmdBuilderMessage(id.String(), *allVerbsSelect), nil
}

func (e *KubectlCmdBuilder) renderMessage(ctx context.Context, botName string, stateDetails stateDetails, includeResourceName bool, bindings, allVerbs, allTypes []string) (interactive.Message, error) {
	var empty interactive.Message

	allVerbsSelect := VerbSelect(botName, allVerbs, stateDetails.verb)
	if allVerbsSelect == nil {
		return empty, errRequiredVerbDropdown
	}

	// 1. Refresh resource type list
	matchingTypes, err := e.getAllowedResourcesSelectList(botName, stateDetails.verb, allTypes, stateDetails.resourceType)
	if err != nil {
		return empty, err
	}

	// 2. If a given verb doesn't have assigned resource types,
	//    render:
	//      1. Dropdown with all verbs
	//      2. Command preview. For example:
	//           kubectl api-resources
	if matchingTypes == nil {
		cmd := fmt.Sprintf("%s %s", kubectlCommandName, stateDetails.verb)
		return KubectlCmdBuilderMessage(
			stateDetails.dropdownsBlockID, *allVerbsSelect,
			WithAdditionalSections(PreviewSection(botName, cmd)),
		), nil
	}

	// 3. If resource type is not on the listy anymore,
	//    render:
	//      1. Dropdown with all verbs
	//      2. Dropdown with all related resource types
	//    because we don't know the resource type we cannot render:
	//      1. Resource names - obvious :).
	//      2. Namespaces as we don't know if it's cluster or namespace scoped resource.
	if !e.contains(matchingTypes, stateDetails.resourceType) {
		return KubectlCmdBuilderMessage(
			stateDetails.dropdownsBlockID, *allVerbsSelect,
			WithAdditionalSelects(matchingTypes),
		), nil
	}

	// At this stage we know that:
	//   1. Verb requires resource types
	//   2. Selected resource type is still valid for the selected verb
	var (
		resNames = e.tryToGetResourceNamesSelect(botName, bindings, stateDetails)
		nsNames  = e.tryToGetNamespaceSelect(ctx, botName, bindings, stateDetails)
	)

	// 4. If a given resource name is not on the list anymore, clear it.
	if !e.contains(resNames, stateDetails.resourceName) {
		stateDetails.resourceName = ""
	}

	// 5. If a given namespace is not on the list anymore, clear it.
	if !e.contains(nsNames, stateDetails.namespace) {
		stateDetails.namespace = ""
	}

	// 6. Render all dropdowns and full command preview.
	preview := e.buildCommandPreview(botName, stateDetails, includeResourceName)
	return KubectlCmdBuilderMessage(
		stateDetails.dropdownsBlockID, *allVerbsSelect,
		WithAdditionalSelects(matchingTypes, resNames, nsNames),
		WithAdditionalSections(preview),
	), nil
}

func (e *KubectlCmdBuilder) tryToGetResourceNamesSelect(botName string, bindings []string, state stateDetails) *interactive.Select {
	if state.resourceType == "" {
		return EmptyResourceNameDropdown(botName)
	}
	cmd := fmt.Sprintf(`%s get %s --ignore-not-found=true -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'`, kubectlCommandName, state.resourceType)
	if state.namespace != "" {
		cmd = fmt.Sprintf("%s -n %s", cmd, state.namespace)
	}

	out, err := e.kcExecutor.Execute(bindings, cmd, true)
	if err != nil {
		return EmptyResourceNameDropdown(botName)
	}

	lines := getNonEmptyLines(out)
	if len(lines) == 0 {
		return EmptyResourceNameDropdown(botName)
	}

	return ResourceNamesSelect(botName, overflowSentence(lines), state.resourceName)
}

func (e *KubectlCmdBuilder) tryToGetNamespaceSelect(ctx context.Context, botName string, bindings []string, details stateDetails) *interactive.Select {
	resourceDetails := e.commandGuard.GetResourceDetails(details.verb, details.resourceType)
	if !resourceDetails.Namespaced {
		return nil
	}

	allClusterNamespaces, err := e.namespaceLister.List(ctx, metav1.ListOptions{
		Limit: dropdownItemsLimit,
	})
	if err != nil {
		return nil
	}

	var (
		kc        = e.merger.MergeAllEnabled(bindings)
		allowedNS = kc.AllowedNamespacesPerResource[details.resourceType]
		finalNS   []string
	)

	defaultNSExists := false
	for _, item := range allClusterNamespaces.Items {
		if !allowedNS.IsAllowed(item.Name) {
			continue
		}
		if details.namespace == item.Name {
			defaultNSExists = true
		}
		finalNS = append(finalNS, item.Name)
	}

	if defaultNSExists {
		// The initial option MUST be a subset of all available dropdown options
		// if the default namespace was not found on that list, don't include it.
		ResourceNamespaceSelect(botName, finalNS, "")
	}

	return ResourceNamespaceSelect(botName, finalNS, details.namespace)
}

func (e *KubectlCmdBuilder) getEnableKubectlDetails(bindings []string) (verbs []string, resources []string, namespace string) {
	enabledKubectls := e.merger.MergeAllEnabled(bindings)
	for key := range enabledKubectls.AllowedKubectlResource {
		resources = append(resources, key)
	}
	sort.Strings(resources)

	for key := range enabledKubectls.AllowedKubectlVerb {
		verbs = append(verbs, key)
	}
	sort.Strings(verbs)

	if enabledKubectls.DefaultNamespace == "" {
		enabledKubectls.DefaultNamespace = kubectlDefaultNamespace
	}

	return verbs, resources, enabledKubectls.DefaultNamespace
}

// getAllowedResourcesSelectList returns dropdown select with allowed resources for a given verb.
func (e *KubectlCmdBuilder) getAllowedResourcesSelectList(botName, verb string, resources []string, resourceType string) (*interactive.Select, error) {
	allowedResources, err := e.commandGuard.GetAllowedResourcesForVerb(verb, resources)
	if err != nil {
		return nil, err
	}
	if len(allowedResources) == 0 {
		return nil, nil
	}

	allowedResourcesList := make([]string, 0, len(allowedResources))
	for _, item := range allowedResources {
		allowedResourcesList = append(allowedResourcesList, item.Name)
	}

	return ResourceTypeSelect(botName, allowedResourcesList, resourceType), nil
}

type stateDetails struct {
	dropdownsBlockID string

	verb         string
	namespace    string
	resourceType string
	resourceName string
}

func (e *KubectlCmdBuilder) extractStateDetails(botName string, state *slack.BlockActionStates) stateDetails {
	if state == nil {
		return stateDetails{}
	}

	details := stateDetails{}
	for blockID, blocks := range state.Values {
		details.dropdownsBlockID = blockID
		for id, act := range blocks {
			id = strings.TrimPrefix(id, botName)
			id = strings.TrimSpace(id)

			switch id {
			case verbsDropdownCommand:
				details.verb = act.SelectedOption.Value
			case resourceTypesDropdownCommand:
				details.resourceType = act.SelectedOption.Value
			case resourceNamesDropdownCommand:
				details.resourceName = act.SelectedOption.Value
			case resourceNamespaceDropdownCommand:
				details.namespace = act.SelectedOption.Value
			}
		}
	}
	return details
}

func (e *KubectlCmdBuilder) contains(matchingTypes *interactive.Select, resourceType string) bool {
	if matchingTypes == nil {
		return false
	}
	for _, item := range matchingTypes.OptionGroups {
		for _, matchingType := range item.Options {
			if resourceType == matchingType.Value {
				return true
			}
		}
	}
	return false
}

func (e *KubectlCmdBuilder) buildCommandPreview(name string, state stateDetails, includeResourceName bool) *interactive.Section {
	resourceDetails := e.commandGuard.GetResourceDetails(state.verb, state.resourceType)

	cmd := fmt.Sprintf("%s %s %s", kubectlCommandName, state.verb, state.resourceType)

	resourceNameSeparator := " "
	if resourceDetails.SlashSeparatedInCommand {
		// sometimes kubectl commands requires slash separator, without it, it will not work. For example:
		//   kubectl logs deploy/<deploy_name>
		resourceNameSeparator = "/"
	}

	if includeResourceName && state.resourceName != "" {
		cmd = fmt.Sprintf("%s%s%s", cmd, resourceNameSeparator, state.resourceName)
	}

	if resourceDetails.Namespaced && state.namespace != "" {
		cmd = fmt.Sprintf("%s -n %s", cmd, state.namespace)
	}

	return PreviewSection(name, cmd)
}

func (e *KubectlCmdBuilder) noVerbsAvailableInChannelMessage() (interactive.Message, error) {
	return interactive.Message{
		Base: interactive.Base{
			Body: interactive.Body{
				Plaintext: noKubectlCommandsInChannel,
			},
		},
	}, nil
}

func splitByNewLines(c rune) bool {
	return c == '\n' || c == '\r'
}

func overflowSentence(in []string) []string {
	for idx := range in {
		if len(in[idx]) < 76 { // Maximum length for text field in dropdown is 75 characters. (https://api.slack.com/reference/block-kit/composition-objects#option)
			continue
		}

		in[idx] = in[idx][:72] + "..."
	}
	return in
}

func getNonEmptyLines(in string) []string {
	lines := strings.FieldsFunc(in, splitByNewLines)
	var out []string
	for _, x := range lines {
		if x == "" {
			continue
		}
		out = append(out, x)
	}
	return out
}
