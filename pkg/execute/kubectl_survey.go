package execute

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
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

	if len(args) <= 1 {
		return Survey(verbs, resTypes, nil, nil), nil
	}

	if len(args) < 2 {
		return empty, errInvalidCommand
	}

	var (
		//cmdName = args[0]
		cmdVerb = args[1]
		cmdArgs = args[2:]
	)

	cmds := executorsRunner{
		"verbs": func() (interactive.Message, error) {
			preview, cmd := getCommandPreview(botName, conv.State)
			resNames := e.tryToGetResourceNamesForCommand(botName, bindings, cmd)
			return Survey(verbs, resTypes, resNames, preview), nil
		},
		"resource": func() (interactive.Message, error) {
			switch cmdArgs[0] {
			case "type":
				preview, cmd := getCommandPreview(botName, conv.State)
				resNames := e.tryToGetResourceNamesForCommand(botName, bindings, cmd)
				return Survey(verbs, resTypes, resNames, preview), nil
			case "name":
				preview, cmd := getCommandPreview(botName, conv.State)
				resNames := e.tryToGetResourceNamesForCommand(botName, bindings, cmd)
				out := Survey(verbs, resTypes, resNames, preview)
				return out, nil
			}
			return empty, nil
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
	for idx := range lines {
		lines[idx] = strings.Split(lines[idx], " ")[0]
	}

	return ResourceNames(botName, overflowSentence(lines))
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

func getCommandPreview(name string, state *slack.BlockActionStates) (*interactive.Section, string) {
	var (
		verb         string
		resourceType string
		resourceName string
	)
	for _, blocks := range state.Values {
		for id, act := range blocks {
			id = strings.TrimPrefix(id, name)
			id = strings.TrimSpace(id)

			switch id {
			case "kcc verbs":
				verb = act.SelectedOption.Value
			case "kcc resource type":
				resourceType = act.SelectedOption.Value
			case "kcc resource name":
				resourceName = act.SelectedOption.Value
			}
		}
	}

	if verb == "" || resourceType == "" {
		return nil, ""
	}

	// fetchingCommand is command without the resource name
	fetchingCommand := ""
	if verb != "" && resourceType != "" {
		fetchingCommand = fmt.Sprintf("kubectl %s %s", verb, resourceType)
	}

	fullCmd := fmt.Sprintf("kubectl %s %s %s", verb, resourceType, resourceName)

	return PreviewSection(name, fullCmd), fetchingCommand
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
