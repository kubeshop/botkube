package execute

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

const (
	verbsDropdownCommand         = "kcc --verbs"
	resourceTypesDropdownCommand = "kcc --resource-type"
	resourceNamesDropdownCommand = "kcc --resource-name"
	kubectlCommandName           = "kubectl"
)

// KubectlSurvey provides functionality to handle interactive kubectl command selection.
type KubectlSurvey struct {
	log        logrus.FieldLogger
	cfg        config.Config
	kcExecutor *Kubectl
	merger     *kubectl.Merger
}

// NewKubectlSurvey returns a new KubectlSurvey instance.
func NewKubectlSurvey(log logrus.FieldLogger, cfg config.Config, merger *kubectl.Merger, executor *Kubectl) *KubectlSurvey {
	return &KubectlSurvey{
		log:        log,
		cfg:        cfg,
		kcExecutor: executor,
		merger:     merger,
	}
}

// Do executes a given kcc command based on args.
func (e *KubectlSurvey) Do(args []string, platform config.CommPlatformIntegration, bindings []string, conv Conversation, botName string) (interactive.Message, error) {
	var empty interactive.Message

	if platform != config.SocketSlackCommPlatformIntegration {
		e.log.Debug("Interactive survey is not supported on %s platform", platform)
		return empty, nil
	}

	verbs, resTypes := e.getVerbsAndResourceTypeSelectLists(botName, bindings)

	if len(args) == 1 { // no args specified, only command name, so we sent generic message
		// We start a new interactive block, so we generate unique ID.
		// Later when we update this message with a new "body" e.g. update command preview
		// the block state remains the same as Slack always see it under the same id.
		// If we use different ID each time we update the message, Slack will clean up the state
		// meaning we will lose information about verb/resourceType/resourceName that were previously selected.
		id, err := uuid.NewRandom()
		if err != nil {
			return empty, err
		}
		return Survey(verbs, resTypes, nil, nil, id.String()), nil
	}

	var (
		cmdVerb = args[1]
	)

	cmds := executorsRunner{
		"--verbs": func() (interactive.Message, error) {
			preview, cmd, dropdownsBlockID := getCommandPreview(botName, conv.State, false)
			resNames := e.tryToGetResourceNamesForCommand(botName, bindings, cmd)
			return Survey(verbs, resTypes, resNames, preview, dropdownsBlockID), nil
		},
		"--resource-type": func() (interactive.Message, error) {
			preview, cmd, dropdownsBlockID := getCommandPreview(botName, conv.State, false)
			resNames := e.tryToGetResourceNamesForCommand(botName, bindings, cmd)
			return Survey(verbs, resTypes, resNames, preview, dropdownsBlockID), nil
		},
		"--resource-name": func() (interactive.Message, error) {
			preview, cmd, dropdownsBlockID := getCommandPreview(botName, conv.State, true)
			resNames := e.tryToGetResourceNamesForCommand(botName, bindings, cmd)
			return Survey(verbs, resTypes, resNames, preview, dropdownsBlockID), nil
		},
	}

	msg, err := cmds.SelectAndRun(cmdVerb)
	if err != nil {
		return empty, err
	}
	return msg, nil
}

func (e *KubectlSurvey) tryToGetResourceNamesForCommand(botName string, bindings []string, cmd string) *interactive.Select {
	if cmd == "" {
		return nil
	}

	getResNamesCmd := cmd + ` --ignore-not-found=true -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'`
	out, err := e.kcExecutor.Execute(bindings, getResNamesCmd, true)
	if err != nil {
		return nil
	}

	lines := strings.FieldsFunc(out, splitByNewLines)
	return ResourceNamesSelect(botName, overflowSentence(lines))
}

func (e *KubectlSurvey) getVerbsAndResourceTypeSelectLists(botName string, bindings []string) (*interactive.Select, *interactive.Select) {
	enabledKubectls := e.merger.MergeAllEnabled(bindings)
	var resources []string
	for key := range enabledKubectls.AllowedKubectlResource {
		resources = append(resources, key)
	}
	sort.Strings(resources)

	var verbs []string
	for key := range enabledKubectls.AllowedKubectlVerb {
		verbs = append(verbs, key)
	}
	sort.Strings(verbs)

	return VerbSelect(botName, verbs), ResourceTypeSelect(botName, resources)
}

func getCommandPreview(name string, state *slack.BlockActionStates, includeResourceName bool) (*interactive.Section, string, string) {
	var (
		verb         string
		resourceType string
		resourceName string
	)
	var dropdownsBlockID string
	for blockID, blocks := range state.Values {
		dropdownsBlockID = blockID
		for id, act := range blocks {
			id = strings.TrimPrefix(id, name)
			id = strings.TrimSpace(id)

			switch id {
			case verbsDropdownCommand:
				verb = act.SelectedOption.Value
			case resourceTypesDropdownCommand:
				resourceType = act.SelectedOption.Value
			case resourceNamesDropdownCommand:
				if includeResourceName {
					resourceName = act.SelectedOption.Value
				}
			}
		}
	}

	if verb == "" || resourceType == "" {
		return nil, "", dropdownsBlockID
	}

	// fetchingCommand is command always without the resource name
	fetchingCommand := ""
	if verb != "" && resourceType != "" {
		fetchingCommand = fmt.Sprintf("%s %s %s", kubectlCommandName, verb, resourceType)
	}

	fullCmd := fmt.Sprintf("%s %s %s %s", kubectlCommandName, verb, resourceType, resourceName)

	return PreviewSection(name, fullCmd), fetchingCommand, dropdownsBlockID
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
