package execute_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

const testingBotName = "@BKTesting"

func TestCommandPreview(t *testing.T) {
	var (
		logger, _   = logtest.NewNullLogger()
		fixBindings = []string{"kc-read-only", "kc-delete-pod"}
	)

	tests := []struct {
		name string
		args []string

		expMsg interactive.Message
	}{
		{
			name: "Print the verb and resource type based on a given state when triggered by verbs dropdown",
			args: strings.Fields("kcc --verbs"),

			expMsg: fixStateSurveyMessage("kubectl get pods ", "@BKTesting kubectl get pods ", fixAllDropdown()...),
		},
		{
			name: "Print the verb and resource type based on a given state when triggered by resource type dropdown",
			args: strings.Fields("kcc --resource-type"),

			expMsg: fixStateSurveyMessage("kubectl get pods ", "@BKTesting kubectl get pods ", fixAllDropdown()...),
		},
		{
			name: "Print the verb, resource type, and resource name based on a given state when triggered by resource name dropdown",
			args: strings.Fields("kcc --resource-name"),

			expMsg: fixStateSurveyMessage("kubectl get pods nginx2", "@BKTesting kubectl get pods nginx2", fixAllDropdown()...),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var (
				expKubectlCmd = `kubectl get pods --ignore-not-found=true -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'`
				state         = fixStateForAllDropdowns()
				kcExecutor    = &fakeKcExecutor{}
				kcMerger      = newFakeKcMerger([]string{"get", "describe"}, []string{"deploy", "pods"})
			)

			kcSurveyExecutor := execute.NewKubectlSurvey(logger, kcMerger, kcExecutor)

			// when
			gotMsg, err := kcSurveyExecutor.Do(tc.args, config.SocketSlackCommPlatformIntegration, fixBindings, state, testingBotName)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expMsg, gotMsg)
			assert.Equal(t, expKubectlCmd, kcExecutor.command)
			assert.True(t, kcExecutor.isAuthed)
			assert.Equal(t, fixBindings, kcExecutor.bindings)
		})
	}
}

func TestIgnorePlatformsOtherThanSocketSlack(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	platforms := []config.CommPlatformIntegration{
		config.SlackCommPlatformIntegration,
		config.MattermostCommPlatformIntegration,
		config.TeamsCommPlatformIntegration,
		config.DiscordCommPlatformIntegration,
		config.ElasticsearchCommPlatformIntegration,
		config.WebhookCommPlatformIntegration,
	}
	for _, platform := range platforms {
		t.Run(fmt.Sprintf("Should ignore %s", platform), func(t *testing.T) {
			// given
			kcSurveyExecutor := execute.NewKubectlSurvey(logger, nil, nil)

			// when
			gotMsg, err := kcSurveyExecutor.Do(nil, platform, nil, nil, "")

			// then
			require.NoError(t, err)
			assert.Empty(t, gotMsg)
		})
	}
}

func TestShouldReturnInitialMessage(t *testing.T) {
	// given
	var (
		logger, _        = logtest.NewNullLogger()
		kcMerger         = newFakeKcMerger([]string{"get", "describe"}, []string{"deploy", "pods"})
		kcSurveyExecutor = execute.NewKubectlSurvey(logger, kcMerger, nil)
		expMsg           = fixInitialSurveyMessage()
	)

	// when command args are not specified
	cmd := []string{"kcc"}
	gotMsg, err := kcSurveyExecutor.Do(cmd, config.SocketSlackCommPlatformIntegration, nil, nil, testingBotName)

	// then
	require.NoError(t, err)

	require.Len(t, gotMsg.Sections, 1)
	assert.NotEmpty(t, gotMsg.Sections[0].Selects.ID) // assert that we fill that property
	gotMsg.Sections[0].Selects.ID = ""                // zero that before comparison, as this is UUID that it's different in each test execution.

	assert.Equal(t, expMsg, gotMsg)
}

func TestShouldNotPrintTheResourceNameIfKubectlExecutorFails(t *testing.T) {
	// given
	var (
		logger, _  = logtest.NewNullLogger()
		state      = fixStateForAllDropdowns()
		kcExecutor = &fakeErrorKcExecutor{}
		kcMerger   = newFakeKcMerger([]string{"get", "describe"}, []string{"deploy", "pods"})
		args       = []string{"kcc", "--verbs"}
		expMsg     = fixStateSurveyMessage("kubectl get pods ", "@BKTesting kubectl get pods ", fixVerbsDropdown(), fixResourceTypeDropdown())
	)

	kcSurveyExecutor := execute.NewKubectlSurvey(logger, kcMerger, kcExecutor)

	// when
	gotMsg, err := kcSurveyExecutor.Do(args, config.SocketSlackCommPlatformIntegration, []string{"kc-read-only"}, state, testingBotName)

	// then
	require.NoError(t, err)
	assert.Equal(t, expMsg, gotMsg)
}

func fixStateForAllDropdowns() *slack.BlockActionStates {
	return &slack.BlockActionStates{
		Values: map[string]map[string]slack.BlockAction{
			"dropdown-block-id-403aca17d958": {
				"@BKTesting kcc --resource-name": {
					SelectedOption: slack.OptionBlockObject{
						Value: "nginx2",
					},
				},
				"@BKTesting kcc --resource-type": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "pods",
					},
				},
				"@BKTesting kcc --verbs": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "get",
					},
				},
			},
		},
	}
}

func fixInitialSurveyMessage() interactive.Message {
	return interactive.Message{
		Sections: []interactive.Section{
			{
				Selects: interactive.Selects{
					Items: []interactive.Select{
						fixVerbsDropdown(),
						fixResourceTypeDropdown(),
					},
				},
			},
		},
		OnlyVisibleForYou: true,
		ReplaceOriginal:   true,
	}
}

func fixVerbsDropdown() interactive.Select {
	return interactive.Select{
		Name:    "Commands",
		Command: "@BKTesting kcc --verbs",
		OptionGroups: []interactive.OptionGroup{
			{
				Name: "Commands",
				Options: []interactive.OptionItem{
					{
						Name:  "describe",
						Value: "describe",
					},
					{
						Name:  "get",
						Value: "get",
					},
				},
			},
		},
	}
}

func fixResourceTypeDropdown() interactive.Select {
	return interactive.Select{
		Name:    "Resources",
		Command: "@BKTesting kcc --resource-type",
		OptionGroups: []interactive.OptionGroup{
			{
				Name: "Resources",
				Options: []interactive.OptionItem{
					{
						Name:  "deploy",
						Value: "deploy",
					},
					{
						Name:  "pods",
						Value: "pods",
					},
				},
			},
		},
	}
}

func fixResourceNamesDropdown() interactive.Select {
	return interactive.Select{
		Name:    "Resource name",
		Command: "@BKTesting kcc --resource-name",
		OptionGroups: []interactive.OptionGroup{
			{
				Name: "Resource name",
				Options: []interactive.OptionItem{
					{
						Name:  "kc line1",
						Value: "kc line1",
					},
					{
						Name:  "line2",
						Value: "line2",
					},
				},
			},
		},
	}
}

func fixAllDropdown() []interactive.Select {
	return []interactive.Select{
		fixVerbsDropdown(),
		fixResourceTypeDropdown(),
		fixResourceNamesDropdown(),
	}
}

func fixStateSurveyMessage(kcCommandPreview, kcCommand string, dropdowns ...interactive.Select) interactive.Message {
	return interactive.Message{
		Sections: []interactive.Section{
			{
				Selects: interactive.Selects{
					ID:    "dropdown-block-id-403aca17d958", // It's important to have the same ID as we have in fixture state object.
					Items: dropdowns,
				},
			},
			{
				Base: interactive.Base{
					Body: interactive.Body{
						CodeBlock: kcCommandPreview,
					},
				},
				Buttons: interactive.Buttons{
					interactive.Button{
						Name:    "Run command",
						Command: kcCommand,
						Style:   "primary",
					},
				},
			},
		},
		OnlyVisibleForYou: true,
		ReplaceOriginal:   true,
	}
}

type fakeKcExecutor struct {
	isAuthed bool
	command  string
	bindings []string
}

func (r *fakeKcExecutor) Execute(bindings []string, command string, isAuthChannel bool) (string, error) {
	r.bindings = bindings
	r.command = command
	r.isAuthed = isAuthChannel

	return "kc line1\nline2", nil
}

type fakeErrorKcExecutor struct{}

func (r *fakeErrorKcExecutor) Execute(_ []string, _ string, _ bool) (string, error) {
	return "", errors.New("fake error")
}

type fakeKcMerger struct {
	allowedVerbs     []string
	allowedResources []string
}

func newFakeKcMerger(allowedVerbs []string, allowedResources []string) *fakeKcMerger {
	return &fakeKcMerger{allowedVerbs: allowedVerbs, allowedResources: allowedResources}
}

func (r *fakeKcMerger) MergeAllEnabled(_ []string) kubectl.EnabledKubectl {
	verbs := map[string]struct{}{}
	for _, name := range r.allowedVerbs {
		verbs[name] = struct{}{}
	}
	resources := map[string]struct{}{}
	for _, name := range r.allowedResources {
		resources[name] = struct{}{}
	}
	return kubectl.EnabledKubectl{
		AllowedKubectlVerb:     verbs,
		AllowedKubectlResource: resources,
	}
}
