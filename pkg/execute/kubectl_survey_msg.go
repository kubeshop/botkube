package execute

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

// Survey returns the survey message for selecting kubectl command.
func Survey(verbs, resources, resourceNames *interactive.Select, commandPreview *interactive.Section, dropdownsBlockID string) interactive.Message {
	var selects []interactive.Select
	if verbs != nil {
		selects = append(selects, *verbs)
	}
	if resources != nil {
		selects = append(selects, *resources)
	}
	if resourceNames != nil {
		selects = append(selects, *resourceNames)
	}

	var sections []interactive.Section

	if len(selects) > 0 {
		sections = append(sections, interactive.Section{
			Selects: interactive.Selects{
				ID:    dropdownsBlockID,
				Items: selects,
			},
		})
	}

	if commandPreview != nil {
		sections = append(sections, *commandPreview)
	}

	return interactive.Message{
		ReplaceOriginal:   true,
		OnlyVisibleForYou: true,
		Sections:          sections,
	}
}

// PreviewSection returns preview command section with Run button.
func PreviewSection(botName, cmd string) *interactive.Section {
	btn := interactive.ButtonBuilder{BotName: botName}
	return &interactive.Section{
		Base: interactive.Base{
			Body: interactive.Body{
				CodeBlock: cmd,
			},
		},
		Buttons: interactive.Buttons{
			btn.ForCommand("Run command", cmd, interactive.ButtonStylePrimary),
		},
	}
}

// VerbSelect return drop-down select for kubectl verbs.
func VerbSelect(botName string, verbs []string) *interactive.Select {
	return selectDropdown("Commands", verbsDropdownCommand, botName, verbs)
}

// ResourceTypeSelect return drop-down select for kubectl resources types.
func ResourceTypeSelect(botName string, resources []string) *interactive.Select {
	return selectDropdown("Resources", resourceTypesDropdownCommand, botName, resources)
}

// ResourceNamesSelect return drop-down select for kubectl resources names.
func ResourceNamesSelect(botName string, names []string) *interactive.Select {
	return selectDropdown("Resource name", resourceNamesDropdownCommand, botName, names)
}

func selectDropdown(name, cmd, botName string, items []string) *interactive.Select {
	if len(items) == 0 {
		return nil
	}

	var opts []interactive.OptionItem
	for _, itemName := range items {
		opts = append(opts, interactive.OptionItem{
			Name:  itemName,
			Value: itemName,
		})
	}

	return &interactive.Select{
		Name:    name,
		Command: fmt.Sprintf("%s %s", botName, cmd),
		OptionGroups: []interactive.OptionGroup{
			{
				Name:    name,
				Options: opts,
			},
		},
	}
}
