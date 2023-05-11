package action_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/action"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

func TestProvider_RenderedActionsForEvent(t *testing.T) {
	// given
	testCases := []struct {
		Name               string
		Config             config.Actions
		Event              event
		SourceBindings     []string
		ExpectedResult     []action.Action
		ExpectedErrMessage string
	}{
		{
			Name:           "Success - filter disabled actions and the ones with different bindings",
			Config:         fixActionsConfig(),
			SourceBindings: []string{"success", "disabled"},
			Event:          fixEvent("name"),
			ExpectedResult: []action.Action{
				{
					Command:          "{{BotName}} kubectl get po name",
					ExecutorBindings: []string{"executor-binding1", "executor-binding2"},
					DisplayName:      "Success",
				},
			},
		},
		{
			Name:           "No matching actions",
			Config:         fixActionsConfig(),
			SourceBindings: []string{"totally-different"},
			Event:          fixEvent("name"),
			ExpectedResult: nil,
		},
		{
			Name:           "Both valid and invalid actions",
			Config:         fixActionsConfig(),
			SourceBindings: []string{"success", "invalid-command"},
			Event:          fixEvent("name"),
			ExpectedResult: []action.Action{
				{
					Command:          "{{BotName}} kubectl get po name",
					ExecutorBindings: []string{"executor-binding1", "executor-binding2"},
					DisplayName:      "Success",
				},
			},
			ExpectedErrMessage: heredoc.Doc(`
				1 error occurred:
					* while rendering command "kubectl get po {{ .SomethingElse }}" for Action "Invalid Command": template: action-cmd:1:18: executing "action-cmd" at <.SomethingElse>: can't evaluate field SomethingElse in type action.renderingData`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			provider := action.NewProvider(loggerx.NewNoop(), tc.Config, nil)

			// when
			result, err := provider.RenderedActions(tc.Event, tc.SourceBindings)

			// then
			if tc.ExpectedErrMessage != "" {
				assert.Equal(t, tc.ExpectedErrMessage, err.Error())
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}

func TestProvider_ExecuteEventAction(t *testing.T) {
	// given
	botName := "my-bot"
	userName := `Automation "Test"`
	executorBindings := []string{"executor-binding1", "executor-binding2"}
	eventAction := action.Action{
		Command:          "kubectl get po foo",
		ExecutorBindings: executorBindings,
		DisplayName:      "Test",
	}
	expectedExecutorInput := execute.NewDefaultInput{
		CommGroupName:   "n/a",
		Platform:        "n/a",
		NotifierHandler: nil, // won't check it
		Conversation: execute.Conversation{
			Alias:            "n/a",
			ID:               "n/a",
			ExecutorBindings: executorBindings,
			IsKnown:          true,
			CommandOrigin:    command.AutomationOrigin,
		},
		Message: "kubectl get po foo",
		User: execute.UserInput{
			Mention:     userName,
			DisplayName: userName,
		},
	}

	execFactory := &fakeFactory{t: t, expectedInput: expectedExecutorInput}
	provider := action.NewProvider(loggerx.NewNoop(), config.Actions{}, execFactory)

	// when
	msg := provider.ExecuteAction(context.Background(), eventAction)
	msg.ReplaceBotNamePlaceholder(botName)

	// then
	assert.Equal(t, fixInteractiveMessage(botName), msg)
}

func fixActionsConfig() config.Actions {
	executorBindings := []string{"executor-binding1", "executor-binding2"}
	sampleCommand := "kubectl get po {{ .Event.Name }}"
	return config.Actions{
		"success": {
			Enabled:     true,
			DisplayName: "Success",
			Command:     sampleCommand,
			Bindings: config.ActionBindings{
				Sources:   []string{"success", "success2"},
				Executors: executorBindings,
			},
		},
		"disabled": {
			Enabled:     false,
			DisplayName: "Disabled",
			Command:     sampleCommand,
			Bindings: config.ActionBindings{
				Sources:   []string{"disabled"},
				Executors: executorBindings,
			},
		},
		"different": {
			Enabled:     true,
			DisplayName: "Different",
			Command:     sampleCommand,
			Bindings: config.ActionBindings{
				Sources:   []string{"different"},
				Executors: executorBindings,
			},
		},
		"invalid-command": {
			Enabled:     true,
			DisplayName: "Invalid Command",
			Command:     "kubectl get po {{ .SomethingElse }}",
			Bindings: config.ActionBindings{
				Sources:   []string{"invalid-command"},
				Executors: executorBindings,
			},
		},
	}
}

type event struct {
	Name string
}

func fixEvent(name string) event {
	return event{
		Name: name,
	}
}

type fakeFactory struct {
	t             *testing.T
	expectedInput execute.NewDefaultInput
}

func (f *fakeFactory) NewDefault(input execute.NewDefaultInput) execute.Executor {
	input.NotifierHandler = nil
	require.Equal(f.t, f.expectedInput, input)

	return &fakeExecutor{}
}

type fakeExecutor struct{}

func (fakeExecutor) Execute(_ context.Context) interactive.CoreMessage {
	return fixInteractiveMessage("{{BotName}}")
}

func fixInteractiveMessage(botName string) interactive.CoreMessage {
	return interactive.CoreMessage{
		Header: "Sample",
		Message: api.Message{
			PlaintextInputs: []api.LabelInput{
				{
					Command:          fmt.Sprintf("%s kubectl get po foo", botName),
					Text:             "",
					Placeholder:      "",
					DispatchedAction: "",
				},
			},
		},
	}
}
