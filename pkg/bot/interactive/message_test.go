package interactive_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

func TestMessage_ReplaceBotNameInCommands(t *testing.T) {
	// given
	msg := interactive.Message{
		Type: interactive.Default,
		PlaintextInputs: []interactive.LabelInput{
			{
				Command: "@OldBot command1",
				Text:    "Top command 1",
			},
			{
				Command: "@OldBot command2",
				Text:    "Top command 2",
			},
		},
		Base: interactive.Base{
			Header: "Test",
		},
		Sections: []interactive.Section{
			{
				Buttons: []interactive.Button{
					{
						Name:    "Foo",
						Command: "@OldBot foo bar",
					},
					{
						Name:    "Bar",
						Command: "@OldBot get po",
					},
				},
				MultiSelect: interactive.MultiSelect{
					Name:    "Sample",
					Command: "@OldBot edit SourceBindings",
					Options: []interactive.OptionItem{
						{Name: "BAR", Value: "bar"},
					},
					InitialOptions: []interactive.OptionItem{
						{Name: "BAR", Value: "bar"},
					},
				},
				Selects: interactive.Selects{
					ID: "foo",
					Items: []interactive.Select{
						{
							Name:    "one",
							Command: "@OldBot get po",
						},
						{
							Name:    "two",
							Command: "@OldBot get po",
						},
					},
				},
				PlaintextInputs: []interactive.LabelInput{
					{
						Command: "@OldBot command",
						Text:    "Foo",
					},
				},
			},
		},
	}
	expectedMsg := interactive.Message{
		Type: interactive.Default,
		PlaintextInputs: []interactive.LabelInput{
			{
				Command: "@NewBot command1",
				Text:    "Top command 1",
			},
			{
				Command: "@NewBot command2",
				Text:    "Top command 2",
			},
		},
		Base: interactive.Base{
			Header: "Test",
		},
		Sections: []interactive.Section{
			{
				Buttons: []interactive.Button{
					{
						Name:    "Foo",
						Command: "@NewBot foo bar",
					},
					{
						Name:    "Bar",
						Command: "@NewBot get po",
					},
				},
				MultiSelect: interactive.MultiSelect{
					Name:    "Sample",
					Command: "@NewBot edit SourceBindings",
					Options: []interactive.OptionItem{
						{Name: "BAR", Value: "bar"},
					},
					InitialOptions: []interactive.OptionItem{
						{Name: "BAR", Value: "bar"},
					},
				},
				Selects: interactive.Selects{
					ID: "foo",
					Items: []interactive.Select{
						{
							Name:    "one",
							Command: "@NewBot get po",
						},
						{
							Name:    "two",
							Command: "@NewBot get po",
						},
					},
				},
				PlaintextInputs: []interactive.LabelInput{
					{
						Command: "@NewBot command",
						Text:    "Foo",
					},
				},
			},
		},
	}

	// when
	msg.ReplaceBotNameInCommands("@OldBot", "@NewBot")

	// then
	assert.Equal(t, expectedMsg, msg)
}
