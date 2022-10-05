package interactive

// EventCommandsSection defines a structure of commands for a given event.
func EventCommandsSection(cmdPrefix string, optionItems []OptionItem) Section {
	section := Section{
		Selects: Selects{
			ID: "",
			Items: []Select{
				{
					Name:    "Select command...",
					Command: cmdPrefix,
					OptionGroups: []OptionGroup{
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
