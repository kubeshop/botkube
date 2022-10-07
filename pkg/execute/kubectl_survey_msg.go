package execute

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

type (
	// SurveyOptions holds survey message options.
	SurveyOptions struct {
		selects  []interactive.Select
		sections []interactive.Section
	}
	// SurveyOption defines option mutator signature.
	SurveyOption func(options *SurveyOptions)
)

// WithAdditionalSelects adds additional selects to a given kubectl Survey message.
func WithAdditionalSelects(in ...*interactive.Select) SurveyOption {
	return func(options *SurveyOptions) {
		for _, s := range in {
			if s == nil {
				continue
			}
			options.selects = append(options.selects, *s)
		}
	}
}

// WithAdditionalSections adds additional sections to a given kubectl Survey message.
func WithAdditionalSections(in ...*interactive.Section) SurveyOption {
	return func(options *SurveyOptions) {
		for _, s := range in {
			if s == nil {
				continue
			}
			options.sections = append(options.sections, *s)
		}
	}
}

// Survey returns the survey message for selecting kubectl command.
func Survey(dropdownsBlockID string, verbs interactive.Select, opts ...SurveyOption) interactive.Message {
	defaultOpt := SurveyOptions{
		selects: []interactive.Select{
			verbs,
		},
	}
	for _, opt := range opts {
		opt(&defaultOpt)
	}

	var sections []interactive.Section
	sections = append(sections, interactive.Section{
		Selects: interactive.Selects{
			ID:    dropdownsBlockID,
			Items: defaultOpt.selects,
		},
	})

	sections = append(sections, defaultOpt.sections...)
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
			btn.ForCommand(interactive.RunCommandName, cmd, interactive.ButtonStylePrimary),
		},
	}
}

// VerbSelect return drop-down select for kubectl verbs.
func VerbSelect(botName string, verbs []string, initialItem string) *interactive.Select {
	return selectDropdown("Commands", verbsDropdownCommand, botName, verbs, initialItem)
}

// ResourceTypeSelect return drop-down select for kubectl resources types.
func ResourceTypeSelect(botName string, resources []string, initialItem string) *interactive.Select {
	return selectDropdown("Resources", resourceTypesDropdownCommand, botName, resources, initialItem)
}

// ResourceNamesSelect return drop-down select for kubectl resources names.
func ResourceNamesSelect(botName string, names []string, initialItem string) *interactive.Select {
	return selectDropdown("Resource names", resourceNamesDropdownCommand, botName, names, initialItem)
}

// ResourceNamespaceSelect return drop-down select for kubectl allowed namespaces.
func ResourceNamespaceSelect(botName string, names []string, initialNamespace string) *interactive.Select {
	return selectDropdown("Namespaces", resourceNamespaceDropdownCommand, botName, names, initialNamespace)
}

func selectDropdown(name, cmd, botName string, items []string, initialItem string) *interactive.Select {
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

	var initialOption *interactive.OptionItem
	if initialItem != "" {
		initialOption = &interactive.OptionItem{
			Name:  initialItem,
			Value: initialItem,
		}
	}

	return &interactive.Select{
		Name:          name,
		Command:       fmt.Sprintf("%s %s", botName, cmd),
		InitialOption: initialOption,
		OptionGroups: []interactive.OptionGroup{
			{
				Name:    name,
				Options: opts,
			},
		},
	}
}

// EmptyResourceNameDropdown returns a select that simulates an empty one.
// Normally, Slack doesn't allow to return a static select with no options.
// This is a workaround to send a dropdown that it's rendered even if empty.
// We use that to preserve a proper order in displayed dropdowns.
//
// How it works under the hood:
//  1. This select is converted to external data source (https://api.slack.com/reference/block-kit/block-elements#external_select)
//  2. We change the `min_query_length` to 0 to remove th "Type minimum of 3 characters to see options" message.
//  3. Our backend doesn't return any options, so you see "No result".
func EmptyResourceNameDropdown(botName string) *interactive.Select {
	return &interactive.Select{
		Type:    "external",
		Name:    "Resource names",
		Command: fmt.Sprintf("%s %s", botName, resourceNamesDropdownCommand),
	}
}
