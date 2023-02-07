package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/api"
)

func TestMessage_ReplaceBotNameInCommands(t *testing.T) {
	// given
	msg := api.Message{
		Type: api.DefaultMessage,
		PlaintextInputs: []api.LabelInput{
			{
				Command: "@OldBot command1",
				Text:    "Top command 1",
			},
			{
				Command: "@OldBot command2",
				Text:    "Top command 2",
			},
		},
		//Base: api.Base{
		//	Header: "Test",
		//},
		Sections: []api.Section{
			{
				Buttons: []api.Button{
					{
						Name:    "Foo",
						Command: "@OldBot foo bar",
					},
					{
						Name:    "Bar",
						Command: "@OldBot get po",
					},
				},
				MultiSelect: api.MultiSelect{
					Name:    "Sample",
					Command: "@OldBot edit SourceBindings",
					Options: []api.OptionItem{
						{Name: "BAR", Value: "bar"},
					},
					InitialOptions: []api.OptionItem{
						{Name: "BAR", Value: "bar"},
					},
				},
				Selects: api.Selects{
					ID: "foo",
					Items: []api.Select{
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
				PlaintextInputs: []api.LabelInput{
					{
						Command: "@OldBot command",
						Text:    "Foo",
					},
				},
			},
		},
	}
	expectedMsg := api.Message{
		Type: api.DefaultMessage,
		PlaintextInputs: []api.LabelInput{
			{
				Command: "@NewBot command1",
				Text:    "Top command 1",
			},
			{
				Command: "@NewBot command2",
				Text:    "Top command 2",
			},
		},
		//Base: api.Base{
		//	Header: "Test",
		//},
		Sections: []api.Section{
			{
				Buttons: []api.Button{
					{
						Name:    "Foo",
						Command: "@NewBot foo bar",
					},
					{
						Name:    "Bar",
						Command: "@NewBot get po",
					},
				},
				MultiSelect: api.MultiSelect{
					Name:    "Sample",
					Command: "@NewBot edit SourceBindings",
					Options: []api.OptionItem{
						{Name: "BAR", Value: "bar"},
					},
					InitialOptions: []api.OptionItem{
						{Name: "BAR", Value: "bar"},
					},
				},
				Selects: api.Selects{
					ID: "foo",
					Items: []api.Select{
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
				PlaintextInputs: []api.LabelInput{
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
