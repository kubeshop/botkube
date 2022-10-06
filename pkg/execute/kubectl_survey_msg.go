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
func VerbSelect(botName string, verbs []string) *interactive.Select {
	return selectDropdown("Commands", verbsDropdownCommand, botName, verbs, nil)
}

// ResourceTypeSelect return drop-down select for kubectl resources types.
func ResourceTypeSelect(botName string, resources []string) *interactive.Select {
	return selectDropdown("Resources", resourceTypesDropdownCommand, botName, resources, nil)
}

// ResourceNamesSelect return drop-down select for kubectl resources names.
func ResourceNamesSelect(botName string, names []string) *interactive.Select {
	return selectDropdown("Resource name", resourceNamesDropdownCommand, botName, names, nil)
}

// ResourceNamespaceSelect return drop-down select for kubectl allowed namespaces.
func ResourceNamespaceSelect(botName string, names []string, initialNamespace *string) *interactive.Select {
	return selectDropdown("Namespaces", resourceNamespaceDropdownCommand, botName, names, initialNamespace)
}

func selectDropdown(name, cmd, botName string, items []string, initialItem *string) *interactive.Select {
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
	if initialItem != nil {
		initialOption = &interactive.OptionItem{
			Name:  *initialItem,
			Value: *initialItem,
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
