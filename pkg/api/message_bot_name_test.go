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
				Command:          "{{BotName}} command1",
				Text:             "{{BotName}} Top command 1",
				Placeholder:      "{{BotName}} Placeholder",
				DispatchedAction: api.DispatchInputActionOnEnter,
			},
			{
				Command:          "{{BotName}} command2",
				Text:             "Top command 2",
				Placeholder:      "Placeholder 2",
				DispatchedAction: api.DispatchInputActionOnEnter,
			},
		},
		BaseBody: api.Body{
			CodeBlock: "Test",
			Plaintext: "{{BotName}} text",
		},
		Sections: []api.Section{
			{
				Buttons: []api.Button{
					{
						Description: "{{BotName}} desc",
						Name:        "{{BotName}} Foo",
						Command:     "{{BotName}} foo bar",
						URL:         "url",
						Style:       api.ButtonStylePrimary,
					},
					{
						Description: "desc 2",
						Name:        "Bar",
						Command:     "{{BotName}} get po",
						URL:         "url 2",
						Style:       api.ButtonStyleDanger,
					},
				},
				MultiSelect: api.MultiSelect{
					Name:    "Sample",
					Command: "{{BotName}} edit SourceBindings",
					Options: []api.OptionItem{
						{Name: "BAR", Value: "{{BotName}} bar"},
						{Name: "{{BotName}} BAZ", Value: "{{BotName}} baz"},
						{Name: "XYZ", Value: "xyz"},
					},
					InitialOptions: []api.OptionItem{
						{Name: "BAR", Value: "bar"},
						{Name: "{{BotName}} BAZ", Value: "{{BotName}} baz"},
					},
				},
				Selects: api.Selects{
					ID: "foo",
					Items: []api.Select{
						{
							Name:    "one",
							Command: "{{BotName}} get po",
						},
						{
							Name:    "two",
							Command: "{{BotName}} get po",
						},
						{
							Name:    "three",
							Command: "{{BotName}} get po",
							OptionGroups: []api.OptionGroup{
								{
									Name: "{{BotName}} get po",
									Options: []api.OptionItem{
										{Name: "BAR", Value: "{{BotName}} bar"},
										{Name: "{{BotName}} BAZ", Value: "{{BotName}} baz"},
										{Name: "XYZ", Value: "xyz"},
									},
								},
							},
							InitialOption: &api.OptionItem{
								Name: "BAR", Value: "bar",
							},
						},
					},
				},
				PlaintextInputs: []api.LabelInput{
					{
						Command: "{{BotName}} command",
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
				Command:          "@NewBot command1",
				Text:             "@NewBot Top command 1",
				Placeholder:      "@NewBot Placeholder",
				DispatchedAction: api.DispatchInputActionOnEnter,
			},
			{
				Command:          "@NewBot command2",
				Text:             "Top command 2",
				Placeholder:      "Placeholder 2",
				DispatchedAction: api.DispatchInputActionOnEnter,
			},
		},
		BaseBody: api.Body{
			CodeBlock: "Test",
			Plaintext: "@NewBot text",
		},
		Sections: []api.Section{
			{
				Buttons: []api.Button{
					{
						Description: "@NewBot desc",
						Name:        "@NewBot Foo",
						Command:     "@NewBot foo bar",
						URL:         "url",
						Style:       api.ButtonStylePrimary,
					},
					{
						Description: "desc 2",
						Name:        "Bar",
						Command:     "@NewBot get po",
						URL:         "url 2",
						Style:       api.ButtonStyleDanger,
					},
				},
				MultiSelect: api.MultiSelect{
					Name:    "Sample",
					Command: "@NewBot edit SourceBindings",
					Options: []api.OptionItem{
						{Name: "BAR", Value: "@NewBot bar"},
						{Name: "@NewBot BAZ", Value: "@NewBot baz"},
						{Name: "XYZ", Value: "xyz"},
					},
					InitialOptions: []api.OptionItem{
						{Name: "BAR", Value: "bar"},
						{Name: "@NewBot BAZ", Value: "@NewBot baz"},
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
						{
							Name:    "three",
							Command: "@NewBot get po",
							OptionGroups: []api.OptionGroup{
								{
									Name: "@NewBot get po",
									Options: []api.OptionItem{
										{Name: "BAR", Value: "@NewBot bar"},
										{Name: "@NewBot BAZ", Value: "@NewBot baz"},
										{Name: "XYZ", Value: "xyz"},
									},
								},
							},
							InitialOption: &api.OptionItem{
								Name: "BAR", Value: "bar",
							},
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
	msg.ReplaceBotNamePlaceholder("@NewBot")

	// then
	assert.Equal(t, expectedMsg, msg)
}
