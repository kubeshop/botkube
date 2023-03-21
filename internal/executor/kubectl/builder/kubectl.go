package builder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/internal/command"
	"github.com/kubeshop/botkube/pkg/api"
)

var (
	errUnsupportedCommand   = errors.New("unsupported command")
	errRequiredVerbDropdown = errors.New("verbs dropdown select cannot be empty")
)

const (
	interactiveBuilderIndicator      = "@builder"
	verbsDropdownCommand             = "@builder --verbs"
	resourceTypesDropdownCommand     = "@builder --resource-type"
	resourceNamesDropdownCommand     = "@builder --resource-name"
	resourceNamespaceDropdownCommand = "@builder --namespace"
	filterPlaintextInputCommand      = "@builder --filter-query"
	kubectlCommandName               = "kubectl"
	dropdownItemsLimit               = 100
	kubectlMissingCommandMsg         = "Please specify the kubectl command"
)

// Kubectl provides functionality to handle interactive kubectl command selection.
type Kubectl struct {
	commandGuard     CommandGuard
	namespaceLister  NamespaceLister
	log              logrus.FieldLogger
	kcRunner         KubectlRunner
	cfg              Config
	defaultNamespace string
	authCheck        AuthChecker
}

// NewKubectl returns a new Kubectl instance.
func NewKubectl(kcRunner KubectlRunner, cfg Config, logger logrus.FieldLogger, guard CommandGuard, defaultNamespace string, lister NamespaceLister, authCheck AuthChecker) *Kubectl {
	return &Kubectl{
		kcRunner:         kcRunner,
		log:              logger,
		namespaceLister:  lister,
		authCheck:        authCheck,
		commandGuard:     guard,
		cfg:              cfg,
		defaultNamespace: defaultNamespace,
	}
}

// ShouldHandle returns true if it's a valid command for interactive builder.
func ShouldHandle(cmd string) bool {
	if cmd == "" || strings.HasPrefix(cmd, interactiveBuilderIndicator) {
		return true
	}
	return false
}

// Handle constructs the interactive command builder messages.
func (e *Kubectl) Handle(ctx context.Context, cmd string, isInteractivitySupported bool, state *slack.BlockActionStates) (api.Message, error) {
	var empty api.Message

	if !isInteractivitySupported {
		e.log.Debug("Interactive kubectl command builder is not supported. Requesting a full kubectl command.")
		return e.message(kubectlMissingCommandMsg)
	}

	allVerbs, allTypes := e.cfg.Allowed.Verbs, e.cfg.Allowed.Resources
	allVerbs = e.commandGuard.FilterSupportedVerbs(allVerbs)

	if len(allVerbs) == 0 {
		msg := fmt.Sprintf("Unfortunately non of configured %q verbs are supported by interactive command builder.", strings.Join(e.cfg.Allowed.Verbs, ","))
		return e.message(msg)
	}
	args := strings.Fields(cmd)
	if len(args) < 2 { // return initial command builder message as there is no builder params
		return e.initialMessage(allVerbs)
	}
	cmd = fmt.Sprintf("%s %s", args[0], args[1])

	stateDetails := e.extractStateDetails(state)
	if stateDetails.namespace == "" {
		stateDetails.namespace = e.defaultNamespace
	}

	e.log.WithFields(logrus.Fields{
		"namespace":    stateDetails.namespace,
		"resourceName": stateDetails.resourceName,
		"resourceType": stateDetails.resourceType,
		"verb":         stateDetails.verb,
	}).Debug("Extracted Slack state")

	cmds := executorsRunner{
		verbsDropdownCommand: func() (api.Message, error) {
			return e.renderMessage(ctx, stateDetails, allVerbs, allTypes)
		},
		resourceTypesDropdownCommand: func() (api.Message, error) {
			// the resource type was selected, so clear resource name from command preview.
			stateDetails.resourceName = ""
			e.log.Info("Selecting resource type")
			return e.renderMessage(ctx, stateDetails, allVerbs, allTypes)
		},
		resourceNamesDropdownCommand: func() (api.Message, error) {
			// this is called only when the resource name is directly selected from dropdown, so we need to include
			// it in command preview.
			return e.renderMessage(ctx, stateDetails, allVerbs, allTypes)
		},
		resourceNamespaceDropdownCommand: func() (api.Message, error) {
			// when the namespace was changed, there is a small chance that resource name will be still matching,
			// we will need to do the external call to check that. For now, we clear resource name from command preview.
			stateDetails.resourceName = ""
			return e.renderMessage(ctx, stateDetails, allVerbs, allTypes)
		},
		filterPlaintextInputCommand: func() (api.Message, error) {
			return e.renderMessage(ctx, stateDetails, allVerbs, allTypes)
		},
	}

	msg, err := cmds.SelectAndRun(cmd)
	switch err {
	case nil:
	case command.ErrVerbNotSupported:
		return errMessage(allVerbs, ":exclamation: Unfortunately, interactive command builder doesn't support %q verb yet.", stateDetails.verb)
	default:
		e.log.WithField("error", err.Error()).Error("Cannot render the kubectl command builder.")
		return empty, err
	}
	return msg, nil
}

func (e *Kubectl) initialMessage(allVerbs []string) (api.Message, error) {
	var empty api.Message

	// We start a new interactive block, so we generate unique ID.
	// Later when we update this message with a new "body" e.g. update command preview
	// the block state remains the same as Slack always see it under the same id.
	// If we use different ID each time we update the message, Slack will clean up the state
	// meaning we will lose information about verb/resourceType/resourceName that were previously selected.
	id, err := uuid.NewRandom()
	if err != nil {
		return empty, err
	}
	allVerbsSelect := VerbSelect(allVerbs, "")
	if allVerbsSelect == nil {
		return empty, errRequiredVerbDropdown
	}

	msg := KubectlCmdBuilderMessage(id.String(), *allVerbsSelect)
	// we are the initial message, don't replace the original one as we need to send a brand-new message visible only to the user
	// otherwise we can replace a message that is publicly visible.
	msg.ReplaceOriginal = false

	return msg, nil
}

func errMessage(allVerbs []string, errMsgFormat string, args ...any) (api.Message, error) {
	dropdownsBlockID, err := uuid.NewRandom()
	if err != nil {
		return api.Message{}, err
	}

	selects := api.Section{
		Selects: api.Selects{
			ID: dropdownsBlockID.String(),
		},
	}

	allVerbsSelect := VerbSelect(allVerbs, "")
	if allVerbsSelect != nil {
		selects.Selects.Items = []api.Select{
			*allVerbsSelect,
		}
	}

	errBody := api.Section{
		Base: api.Base{
			Body: api.Body{
				Plaintext: fmt.Sprintf(errMsgFormat, args...),
			},
		},
	}

	return api.Message{
		ReplaceOriginal:   true,
		OnlyVisibleForYou: true,
		Sections: []api.Section{
			selects,
			errBody,
		},
	}, nil
}

func (e *Kubectl) renderMessage(ctx context.Context, stateDetails stateDetails, allVerbs, allTypes []string) (api.Message, error) {
	var empty api.Message

	allVerbsSelect := VerbSelect(allVerbs, stateDetails.verb)
	if allVerbsSelect == nil {
		return empty, errRequiredVerbDropdown
	}

	// 1. Refresh resource type list
	matchingTypes, err := e.getAllowedResourcesSelectList(stateDetails.verb, allTypes, stateDetails.resourceType)
	if err != nil {
		return empty, err
	}

	// 2. If a given verb doesn't have assigned resource types,
	//    render:
	//      1. Dropdown with all verbs
	//      2. Filter input
	//      3. Command preview. For example:
	//           kubectl api-resources
	if matchingTypes == nil {
		// we must zero those fields as they are known only if we know the resource type and this verb doesn't have one :)
		stateDetails.resourceType = ""
		stateDetails.resourceName = ""
		stateDetails.namespace = ""
		preview := e.buildCommandPreview(stateDetails)

		return KubectlCmdBuilderMessage(
			stateDetails.dropdownsBlockID, *allVerbsSelect,
			WithAdditionalSections(preview...),
		), nil
	}

	// 3. If resource type is not on the list anymore,
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
		resNames = e.tryToGetResourceNamesSelect(ctx, stateDetails)
		nsNames  = e.tryToGetNamespaceSelect(ctx, stateDetails)
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
	preview := e.buildCommandPreview(stateDetails)
	return KubectlCmdBuilderMessage(
		stateDetails.dropdownsBlockID, *allVerbsSelect,
		WithAdditionalSelects(matchingTypes, resNames, nsNames),
		WithAdditionalSections(preview...),
	), nil
}

func (e *Kubectl) tryToGetResourceNamesSelect(ctx context.Context, state stateDetails) *api.Select {
	e.log.Info("Get resource names")
	if state.resourceType == "" {
		e.log.Info("Return empty resource name")
		return EmptyResourceNameDropdown()
	}
	cmd := fmt.Sprintf(`get %s --ignore-not-found=true -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'`, state.resourceType)
	if state.namespace != "" {
		cmd = fmt.Sprintf("%s -n %s", cmd, state.namespace)
	}
	e.log.Infof("Run cmd %q", cmd)

	out, err := e.kcRunner.RunKubectlCommand(ctx, os.Getenv("KUBECONFIG"), e.defaultNamespace, cmd)
	if err != nil {
		e.log.WithField("error", err.Error()).Error("Cannot fetch resource names. Returning empty resource name dropdown.")
		return EmptyResourceNameDropdown()
	}

	lines := getNonEmptyLines(out)
	if len(lines) == 0 {
		return EmptyResourceNameDropdown()
	}

	return ResourceNamesSelect(overflowSentence(lines), state.resourceName)
}

func (e *Kubectl) tryToGetNamespaceSelect(ctx context.Context, details stateDetails) *api.Select {
	log := e.log.WithFields(logrus.Fields{
		"state": details,
	})

	resourceDetails, err := e.commandGuard.GetResourceDetails(details.verb, details.resourceType)
	if err != nil {
		log.WithField("error", err.Error()).Error("Cannot fetch resource details, ignoring namespace dropdown...")
		return nil
	}

	if !resourceDetails.Namespaced {
		log.Debug("Resource is not namespace-scoped, ignore namespace dropdown...")
		return nil
	}

	initialNamespace := newDropdownItem(details.namespace, details.namespace)
	initialNamespace = e.appendNamespaceSuffixIfDefault(initialNamespace)

	allNs := []dropdownItem{
		initialNamespace,
	}
	for _, name := range e.collectAdditionalNamespaces(ctx) {
		kv := newDropdownItem(name, name)
		if name == details.namespace {
			// already added, skip it
			continue
		}
		allNs = append(allNs, kv)
	}

	return ResourceNamespaceSelect(allNs, initialNamespace)
}

func (e *Kubectl) collectAdditionalNamespaces(ctx context.Context) []string {
	// if preconfigured, use specified those Namespaces
	if len(e.cfg.Allowed.Namespaces) > 0 {
		return e.cfg.Allowed.Namespaces
	}

	// user didn't narrow down the namespace dropdown, so let's try to get all namespaces.
	allClusterNamespaces, err := e.namespaceLister.List(ctx, metav1.ListOptions{
		Limit: dropdownItemsLimit,
	})
	if err != nil {
		e.log.WithField("error", err.Error()).Error("Cannot fetch all available Kubernetes namespaces, ignoring namespace dropdown...")
		// we cannot fetch other namespaces, so let's render only the default one.
		return nil
	}

	var out []string
	for _, item := range allClusterNamespaces.Items {
		out = append(out, item.Name)
	}

	return out
}

// UX requirement to append the (namespace) suffix if the namespace is called `default`.
func (e *Kubectl) appendNamespaceSuffixIfDefault(in dropdownItem) dropdownItem {
	if in.Name == "default" {
		in.Name += " (namespace)"
	}
	return in
}

// getAllowedResourcesSelectList returns dropdown select with allowed resources for a given verb.
func (e *Kubectl) getAllowedResourcesSelectList(verb string, resources []string, resourceType string) (*api.Select, error) {
	allowedResources, err := e.commandGuard.GetAllowedResourcesForVerb(verb, resources)
	if err != nil {
		return nil, err
	}

	allowedResourcesList := make([]string, 0, len(allowedResources))
	for _, item := range allowedResources {
		allowedResourcesList = append(allowedResourcesList, item.Name)
	}

	return ResourceTypeSelect(allowedResourcesList, resourceType), nil
}

type stateDetails struct {
	dropdownsBlockID string

	verb         string
	namespace    string
	resourceType string
	resourceName string
	filter       string
}

func (e *Kubectl) extractStateDetails(state *slack.BlockActionStates) stateDetails {
	if state == nil {
		return stateDetails{}
	}

	details := stateDetails{}
	for blockID, blocks := range state.Values {
		if !strings.Contains(blockID, filterPlaintextInputCommand) {
			details.dropdownsBlockID = blockID
		}
		for id, act := range blocks {
			id = strings.TrimPrefix(id, kubectlCommandName)
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
			case filterPlaintextInputCommand:
				details.filter = act.Value
			}
		}
	}
	return details
}

func (e *Kubectl) contains(matchingTypes *api.Select, resourceType string) bool {
	if matchingTypes == nil {
		return false
	}

	if matchingTypes.InitialOption != nil && matchingTypes.InitialOption.Value == resourceType {
		return true
	}

	return false
}

func (e *Kubectl) buildCommandPreview(state stateDetails) []api.Section {
	resourceDetails, err := e.commandGuard.GetResourceDetails(state.verb, state.resourceType)
	if err != nil {
		e.log.WithFields(logrus.Fields{
			"state": state,
			"error": err.Error(),
		}).Error("Cannot get resource details")
		return []api.Section{InternalErrorSection()}
	}

	err = e.authCheck.CheckUserAccess(state.namespace, state.verb, state.resourceType, state.resourceName)
	if err != nil {
		return []api.Section{
			{
				Base: api.Base{
					Header: "Missing permissions",
					Body: api.Body{
						Plaintext: err.Error(),
					},
				},
				Context: []api.ContextItem{
					{
						Text: "To learn more about `kubectl` RBAC visit https://docs.botkube.io/configuration/executor/kubectl.",
					},
				},
			},
		}
	}

	if resourceDetails.SlashSeparatedInCommand && state.resourceName == "" {
		// we should not render the command as it will be invalid anyway without the resource name
		return nil
	}

	cmd := fmt.Sprintf("%s %s %s", kubectlCommandName, state.verb, state.resourceType)

	resourceNameSeparator := " "
	if resourceDetails.SlashSeparatedInCommand {
		// sometimes kubectl commands requires slash separator, without it, it will not work. For example:
		//   kubectl logs deploy/<deploy_name>
		resourceNameSeparator = "/"
	}

	if state.resourceName != "" {
		cmd = fmt.Sprintf("%s%s%s", cmd, resourceNameSeparator, state.resourceName)
	}

	if resourceDetails.Namespaced && state.namespace != "" {
		cmd = fmt.Sprintf("%s -n %s", cmd, state.namespace)
	}

	if state.filter != "" {
		cmd = fmt.Sprintf("%s --filter=%q", cmd, state.filter)
	}

	return PreviewSection(cmd, FilterSection())
}

func (e *Kubectl) message(msg string) (api.Message, error) {
	return api.NewPlaintextMessage(msg, true), nil
}

type (
	executorFunc    func() (api.Message, error)
	executorsRunner map[string]executorFunc
)

func (cmds executorsRunner) SelectAndRun(cmdVerb string) (api.Message, error) {
	cmdVerb = strings.ToLower(cmdVerb)
	fn, found := cmds[cmdVerb]
	if !found {
		return api.Message{}, errUnsupportedCommand
	}
	return fn()
}
