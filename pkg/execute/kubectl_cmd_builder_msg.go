package execute

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

type (
	// KubectlCmdBuilderOptions holds builder message options.
	KubectlCmdBuilderOptions struct {
		selects  []interactive.Select
		sections []interactive.Section
	}
	// KubectlCmdBuilderOption defines option mutator signature.
	KubectlCmdBuilderOption func(options *KubectlCmdBuilderOptions)

	// dropdownItem describes the data for the dropdown item.
	dropdownItem struct {
		Name  string
		Value string
	}
)

// newDropdownItem returns the dropdownItem instance.
func newDropdownItem(key, value string) dropdownItem {
	return dropdownItem{
		Name:  key,
		Value: value,
	}
}

// dropdownItemsFromSlice is a helper function to create the dropdown items from string slice.
// Name and Value will represent the same data.
func dropdownItemsFromSlice(in []string) []dropdownItem {
	var out []dropdownItem
	for _, item := range in {
		out = append(out, newDropdownItem(item, item))
	}
	return out
}

// WithAdditionalSelects adds additional selects to a given kubectl KubectlCmdBuilderMessage message.
func WithAdditionalSelects(in ...*interactive.Select) KubectlCmdBuilderOption {
	return func(options *KubectlCmdBuilderOptions) {
		for _, s := range in {
			if s == nil {
				continue
			}
			options.selects = append(options.selects, *s)
		}
	}
}

// WithAdditionalSections adds additional sections to a given kubectl KubectlCmdBuilderMessage message.
func WithAdditionalSections(in ...interactive.Section) KubectlCmdBuilderOption {
	return func(options *KubectlCmdBuilderOptions) {
		options.sections = append(options.sections, in...)
	}
}

// KubectlCmdBuilderMessage returns message for constructing kubectl command.
func KubectlCmdBuilderMessage(dropdownsBlockID string, verbs interactive.Select, opts ...KubectlCmdBuilderOption) interactive.Message {
	defaultOpt := KubectlCmdBuilderOptions{
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
func PreviewSection(botName, cmd string, input interactive.LabelInput) []interactive.Section {
	btn := interactive.ButtonBuilder{BotName: botName}
	return []interactive.Section{
		{
			Base: interactive.Base{
				Body: interactive.Body{
					CodeBlock: cmd,
				},
			},
			PlaintextInputs: interactive.LabelInputs{
				input,
			},
		},
		{
			Buttons: interactive.Buttons{
				btn.ForCommandWithoutDesc(interactive.RunCommandName, cmd, interactive.ButtonStylePrimary),
			},
		},
	}
}

// InternalErrorSection returns preview command section with Run button.
func InternalErrorSection() interactive.Section {
	return interactive.Section{
		Base: interactive.Base{
			Body: interactive.Body{
				CodeBlock: "Sorry, an internal error occurred while rendering command preview. See the logs for more details.",
			},
		},
	}
}

// FilterSection returns filter input block.
func FilterSection(botName string) interactive.LabelInput {
	return interactive.LabelInput{
		Text:             "Filter output",
		DispatchedAction: interactive.DispatchInputActionOnCharacter,
		Placeholder:      "Filter output by string (optional)",
		// the whitespace at the end is required, otherwise we will not recognize the command
		// as we will receive:
		//   kc-cmd-builder --filterinput string
		// instead of:
		//   kc-cmd-builder --filter input string
		// TODO: this can be fixed by smarter command parser.
		ID: fmt.Sprintf("%s %s ", botName, filterPlaintextInputCommand),
	}
}

// VerbSelect return drop-down select for kubectl verbs.
func VerbSelect(botName string, verbs []string, initialItem string) *interactive.Select {
	return selectDropdown("Select command", verbsDropdownCommand, botName, dropdownItemsFromSlice(verbs), newDropdownItem(initialItem, initialItem))
}

// ResourceTypeSelect return drop-down select for kubectl resources types.
func ResourceTypeSelect(botName string, resources []string, initialItem string) *interactive.Select {
	return selectDropdown("Select resource", resourceTypesDropdownCommand, botName, dropdownItemsFromSlice(resources), newDropdownItem(initialItem, initialItem))
}

// ResourceNamesSelect return drop-down select for kubectl resources names.
func ResourceNamesSelect(botName string, names []string, initialItem string) *interactive.Select {
	return selectDropdown("Select resource name", resourceNamesDropdownCommand, botName, dropdownItemsFromSlice(names), newDropdownItem(initialItem, initialItem))
}

// ResourceNamespaceSelect return drop-down select for kubectl allowed namespaces.
func ResourceNamespaceSelect(botName string, names []dropdownItem, initialNamespace dropdownItem) *interactive.Select {
	return selectDropdown("Select namespace", resourceNamespaceDropdownCommand, botName, names, initialNamespace)
}

func selectDropdown(name, cmd, botName string, items []dropdownItem, initialItem dropdownItem) *interactive.Select {
	if len(items) == 0 {
		return nil
	}

	var opts []interactive.OptionItem
	foundInitialOptOnList := false
	for _, item := range items {
		if item.Value == "" || item.Name == "" {
			continue
		}

		if initialItem.Value == item.Value && initialItem.Name == item.Name {
			foundInitialOptOnList = true
		}

		opts = append(opts, interactive.OptionItem{
			Name:  item.Name,
			Value: item.Value,
		})
	}

	var initialOption *interactive.OptionItem
	if foundInitialOptOnList {
		initialOption = &interactive.OptionItem{
			Name:  initialItem.Name,
			Value: initialItem.Value,
		}
	}

	if len(opts) == 0 {
		return nil
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
//  4. We don't set the command, so the ID of this select is always randomized by Slack server.
//     As a result, the dropdown value is not cached, and we avoid problem with showing the outdated value.
func EmptyResourceNameDropdown() *interactive.Select {
	return &interactive.Select{
		Type: interactive.ExternalSelect,
		Name: "No resources found",
		InitialOption: &interactive.OptionItem{
			Name:  "No resources found",
			Value: "no-resources",
		},
	}
}
