package builder

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

type (
	// MessageOptions holds builder message options.
	MessageOptions struct {
		selects  []api.Select
		sections []api.Section
	}
	// MesageOption defines option mutator signature.
	MesageOption func(options *MessageOptions)

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
func WithAdditionalSelects(in ...*api.Select) MesageOption {
	return func(options *MessageOptions) {
		for _, s := range in {
			if s == nil {
				continue
			}
			options.selects = append(options.selects, *s)
		}
	}
}

// WithAdditionalSections adds additional sections to a given kubectl KubectlCmdBuilderMessage message.
func WithAdditionalSections(in ...api.Section) MesageOption {
	return func(options *MessageOptions) {
		options.sections = append(options.sections, in...)
	}
}

// KubectlCmdBuilderMessage returns message for constructing kubectl command.
func KubectlCmdBuilderMessage(dropdownsBlockID string, verbs api.Select, opts ...MesageOption) api.Message {
	defaultOpt := MessageOptions{
		selects: []api.Select{
			verbs,
		},
	}
	for _, opt := range opts {
		opt(&defaultOpt)
	}

	var sections []api.Section
	sections = append(sections, api.Section{
		Selects: api.Selects{
			ID:    dropdownsBlockID,
			Items: defaultOpt.selects,
		},
	})

	sections = append(sections, defaultOpt.sections...)
	return api.Message{
		ReplaceOriginal:   true,
		OnlyVisibleForYou: true,
		Sections:          sections,
	}
}

// PreviewSection returns preview command section with Run button.
func PreviewSection(cmd string, input api.LabelInput) []api.Section {
	btn := api.ButtonBuilder{}
	return []api.Section{
		{
			Base: api.Base{
				Body: api.Body{
					CodeBlock: cmd,
				},
			},
			PlaintextInputs: api.LabelInputs{
				input,
			},
		},
		{
			Buttons: api.Buttons{
				btn.ForCommand(interactive.RunCommandName, cmd, api.ButtonStylePrimary),
			},
		},
	}
}

// InternalErrorSection returns preview command section with Run button.
func InternalErrorSection() api.Section {
	return api.Section{
		Base: api.Base{
			Body: api.Body{
				CodeBlock: "Sorry, an internal error occurred while rendering command preview. See the logs for more details.",
			},
		},
	}
}

// FilterSection returns filter input block.
func FilterSection() api.LabelInput {
	return api.LabelInput{
		Text:             "Filter output",
		DispatchedAction: api.DispatchInputActionOnCharacter,
		Placeholder:      "Filter output by string (optional)",
		// the whitespace at the end is required, otherwise we will not recognize the command
		// as we will receive:
		//   kubectl @builder --filterinput string
		// instead of:
		//   kubectl @builder --filter input string
		// TODO: this can be fixed by smarter command parser on the Botkube core side.
		Command: fmt.Sprintf("%s %s %s ", api.MessageBotNamePlaceholder, kubectlCommandName, filterPlaintextInputCommand),
	}
}

// VerbSelect return drop-down select for kubectl verbs.
func VerbSelect(verbs []string, initialItem string) *api.Select {
	return selectDropdown("Select command", verbsDropdownCommand, dropdownItemsFromSlice(verbs), newDropdownItem(initialItem, initialItem))
}

// ResourceTypeSelect return drop-down select for kubectl resources types.
func ResourceTypeSelect(resources []string, initialItem string) *api.Select {
	return selectDropdown("Select resource", resourceTypesDropdownCommand, dropdownItemsFromSlice(resources), newDropdownItem(initialItem, initialItem))
}

// ResourceNamesSelect return drop-down select for kubectl resources names.
func ResourceNamesSelect(names []string, initialItem string) *api.Select {
	return selectDropdown("Select resource name", resourceNamesDropdownCommand, dropdownItemsFromSlice(names), newDropdownItem(initialItem, initialItem))
}

// ResourceNamespaceSelect return drop-down select for kubectl allowed namespaces.
func ResourceNamespaceSelect(names []dropdownItem, initialNamespace dropdownItem) *api.Select {
	return selectDropdown("Select namespace", resourceNamespaceDropdownCommand, names, initialNamespace)
}

func selectDropdown(name, cmd string, items []dropdownItem, initialItem dropdownItem) *api.Select {
	if len(items) == 0 {
		return nil
	}

	var opts []api.OptionItem
	foundInitialOptOnList := false
	for _, item := range items {
		if item.Value == "" || item.Name == "" {
			continue
		}

		if initialItem.Value == item.Value && initialItem.Name == item.Name {
			foundInitialOptOnList = true
		}

		opts = append(opts, api.OptionItem{
			Name:  item.Name,
			Value: item.Value,
		})
	}

	var initialOption *api.OptionItem
	if foundInitialOptOnList {
		initialOption = &api.OptionItem{
			Name:  initialItem.Name,
			Value: initialItem.Value,
		}
	}

	if len(opts) == 0 {
		return nil
	}

	return &api.Select{
		Name:          name,
		Command:       fmt.Sprintf("%s %s %s", api.MessageBotNamePlaceholder, kubectlCommandName, cmd),
		InitialOption: initialOption,
		OptionGroups: []api.OptionGroup{
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
func EmptyResourceNameDropdown() *api.Select {
	return &api.Select{
		Type: api.ExternalSelect,
		Name: "No resources found",
		InitialOption: &api.OptionItem{
			Name:  "No resources found",
			Value: "no-resources",
		},
	}
}
