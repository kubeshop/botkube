package interactive

import "github.com/kubeshop/botkube/pkg/api"

// EventCommandsSection defines a structure of commands for a given event.
func EventCommandsSection(cmdPrefix string, optionItems []api.OptionItem) api.Section {
	section := api.Section{
		Selects: api.Selects{
			ID: "",
			Items: []api.Select{
				{
					Name:    "Run command...",
					Command: cmdPrefix,
					OptionGroups: []api.OptionGroup{
						{
							Name:    "Supported commands",
							Options: optionItems,
						},
					},
				},
			},
		},
	}

	return section
}
