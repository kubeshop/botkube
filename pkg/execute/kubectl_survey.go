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

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

const (
	verbsDropdownCommand             = "kcc --verbs"
	resourceTypesDropdownCommand     = "kcc --resource-type"
	resourceNamesDropdownCommand     = "kcc --resource-name"
	resourceNamespaceDropdownCommand = "kcc --namespace"
	kubectlCommandName               = "kubectl"
	dropdownItemsLimit               = 100
)

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

// KubectlSurvey provides functionality to handle interactive kubectl command selection.
type KubectlSurvey struct {
	log             logrus.FieldLogger
	kcExecutor      kcExecutor
	merger          kcMerger
	namespaceLister NamespaceLister
	commandGuard    FakeCommandGuard
}

// NewKubectlSurvey returns a new KubectlSurvey instance.
func NewKubectlSurvey(log logrus.FieldLogger, merger kcMerger, executor kcExecutor, namespaceLister NamespaceLister) *KubectlSurvey {
	return &KubectlSurvey{
		log:             log,
		kcExecutor:      executor,
		merger:          merger,
		namespaceLister: namespaceLister,
		commandGuard:    FakeCommandGuard{},
	}
}

// Do executes a given kcc command based on args.
// TODO: once we will have a real use-case, we should abstract the Slack state and introduce our own model.
func (e *KubectlSurvey) Do(ctx context.Context, args []string, platform config.CommPlatformIntegration, bindings []string, state *slack.BlockActionStates, botName string) (interactive.Message, error) {
	var empty interactive.Message

	if platform != config.SocketSlackCommPlatformIntegration {
		e.log.Debug("Interactive survey is not supported on %s platform", platform)
		return empty, nil
	}

	allVerbs, allTypes, defaultNs := e.getEnableKubectlDetails(bindings)
	allVerbsSelect := *VerbSelect(botName, allVerbs)

	// if only command name was specified, return initial survey message
	if len(args) == 1 {
		// We start a new interactive block, so we generate unique ID.
		// Later when we update this message with a new "body" e.g. update command preview
		// the block state remains the same as Slack always see it under the same id.
		// If we use different ID each time we update the message, Slack will clean up the state
		// meaning we will lose information about verb/resourceType/resourceName that were previously selected.
		id, err := uuid.NewRandom()
		if err != nil {
			return empty, err
		}
		return Survey(id.String(), allVerbsSelect), nil
	}

	stateDetails := e.extractStateDetails(botName, state)
	if stateDetails.namespace == "" {
		stateDetails.namespace = defaultNs
	}

	cmdVerb := args[1]

	cmds := executorsRunner{
		"--verbs": func() (interactive.Message, error) {
			return e.renderSurvey(ctx, botName, stateDetails, true, bindings, allVerbs, allTypes)
		},
		"--resource-type": func() (interactive.Message, error) {
			// the resource type was selected, so clear resource name from command preview.
			return e.renderSurvey(ctx, botName, stateDetails, false, bindings, allVerbs, allTypes)
		},
		"--resource-name": func() (interactive.Message, error) {
			// this is called only when the resource name is directly selected from dropdown, so we need to include
			// it in command preview.
			return e.renderSurvey(ctx, botName, stateDetails, true, bindings, allVerbs, allTypes)
		},
		"--namespace": func() (interactive.Message, error) {
			// when the namespace was changed, there is a small chance that resource name will be still matching,
			// we will need to do the external call to check that. For now, we clear resource name from command preview.
			return e.renderSurvey(ctx, botName, stateDetails, false, bindings, allVerbs, allTypes)
		},
	}

	msg, err := cmds.SelectAndRun(cmdVerb)
	if err != nil {
		return empty, err
	}
	return msg, nil
}

func (e *KubectlSurvey) renderSurvey(ctx context.Context, botName string, stateDetails stateDetails, includeResourceName bool, bindings, allVerbs, allTypes []string) (interactive.Message, error) {
	var empty interactive.Message

	allVerbsSelect := VerbSelect(botName, allVerbs)
	if allVerbsSelect == nil {
		return empty, errors.New("verbs dropdown select cannot be empty")
	}

	// 1. Refresh resource type list
	matchingTypes, err := e.getAllowedResourcesSelectList(botName, stateDetails.verb, allTypes)
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
		return Survey(
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
		return Survey(
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
	return Survey(
		stateDetails.dropdownsBlockID, *allVerbsSelect,
		WithAdditionalSelects(matchingTypes, resNames, nsNames),
		WithAdditionalSections(preview),
	), nil
}

func (e *KubectlSurvey) tryToGetResourceNamesSelect(botName string, bindings []string, state stateDetails) *interactive.Select {
	if state.resourceType == "" {
		return nil
	}
	cmd := fmt.Sprintf(`%s get %s --ignore-not-found=true -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'`, kubectlCommandName, state.resourceType)
	if state.namespace != "" {
		cmd = fmt.Sprintf("%s -n %s", cmd, state.namespace)
	}

	out, err := e.kcExecutor.Execute(bindings, cmd, true)
	if err != nil {
		return nil
	}

	lines := strings.FieldsFunc(out, splitByNewLines)
	return ResourceNamesSelect(botName, overflowSentence(lines))
}

func (e *KubectlSurvey) tryToGetNamespaceSelect(ctx context.Context, botName string, bindings []string, details stateDetails) *interactive.Select {
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
		ResourceNamespaceSelect(botName, finalNS, nil)
	}

	return ResourceNamespaceSelect(botName, finalNS, &details.namespace)
}

func (e *KubectlSurvey) getEnableKubectlDetails(bindings []string) (verbs []string, resources []string, namespace string) {
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
func (e *KubectlSurvey) getAllowedResourcesSelectList(botName, verb string, resources []string) (*interactive.Select, error) {
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

	return ResourceTypeSelect(botName, allowedResourcesList), nil
}

type stateDetails struct {
	dropdownsBlockID string

	verb         string
	namespace    string
	resourceType string
	resourceName string
}

func (e *KubectlSurvey) extractStateDetails(botName string, state *slack.BlockActionStates) stateDetails {
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

func (e *KubectlSurvey) contains(matchingTypes *interactive.Select, resourceType string) bool {
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

func (e *KubectlSurvey) buildCommandPreview(name string, state stateDetails, includeResourceName bool) *interactive.Section {
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
