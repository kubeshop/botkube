package kubernetes

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
)

func TestGetExtraButtonsAssignedToEvent(t *testing.T) {
	// given
	builder := MessageBuilder{}
	givenButtons := []config.ExtraButtons{
		{
			// This is fully valid
			Enabled: true,
			Trigger: config.Trigger{
				Type: []config.EventType{"error"},
			},
			Button: config.Button{
				DisplayName: "Ask AI",
				CommandTpl:  "ai --resource={{ .Namespace }}/{{ .Kind | lower }}/{{ .Name }} --error={{ .Reason }} --bk-cmd-header='AI assistance'",
			},
		},
		{
			// This is valid, as the 'ERROR' type should be normalized to "error"
			Enabled: true,
			Trigger: config.Trigger{
				Type: []config.EventType{"ERROR"},
			},
			Button: config.Button{
				DisplayName: "Get",
				CommandTpl:  "kubectl get {{ .Kind | lower }}",
			},
		},
		{
			// This is invalid, as we can't render `.Event`
			Enabled: true,
			Trigger: config.Trigger{
				Type: []config.EventType{"error"},
			},
			Button: config.Button{
				DisplayName: "Ask AI v2",
				CommandTpl:  "ai {{.Event.Namespace}} this one is wrong",
			},
		},
		{
			// This is invalid, as the DisplayName and Trigger is not set
			Enabled: true,
			Trigger: config.Trigger{},
			Button: config.Button{
				CommandTpl: "ai {{.Event.Namespace}} this one is wrong",
			},
		},
		{
			// This is invalid, but should be ignored as it's disabled
			Enabled: false,
			Trigger: config.Trigger{},
			Button: config.Button{
				CommandTpl: "ai {{.Event.Namespace}} this one is wrong",
			},
		},
	}

	// when
	gotBtns, err := builder.getExtraButtonsAssignedToEvent(givenButtons, event.Event{Type: "error"})
	assert.EqualError(t, err, heredoc.Doc(`
        2 errors occurred:
        	* invalid extraButtons[2].commandTpl: template: Ask AI v2:1:11: executing "Ask AI v2" at <.Event.Namespace>: can't evaluate field Event in type event.Event
        	* invalid extraButtons[3]: displayName cannot be empty, trigger.type cannot be empty`))

	// then
	require.Len(t, gotBtns, 2)
	for idx, btn := range gotBtns {
		assert.Equal(t, givenButtons[idx].Button.DisplayName, btn.Name)
	}
}
