package api

import (
	"fmt"
	"strings"
)

const (
	// MessageBotNamePlaceholder is a cross-platform placeholder for bot name.
	MessageBotNamePlaceholder = "{{BotName}}"
	maxReplaceNo              = 100
)

// BotNameOption allows modifying ReplaceBotNamePlaceholder related options.
type BotNameOption func(opts *BotNameOptions)

// BotNameOptions holds options used in ReplaceBotNamePlaceholder func
type BotNameOptions struct {
	ClusterName string
}

// BotNameWithClusterName sets the cluster name for places where MessageBotNamePlaceholder was also specified.
func BotNameWithClusterName(clusterName string) BotNameOption {
	return func(opts *BotNameOptions) {
		opts.ClusterName = clusterName
	}
}

// ReplaceBotNamePlaceholder replaces bot name placeholder with a given name.
func (msg *Message) ReplaceBotNamePlaceholder(new string, opts ...BotNameOption) {
	var options BotNameOptions
	for _, mutate := range opts {
		mutate(&options)
	}

	for idx, item := range msg.Sections {
		msg.Sections[idx].Buttons = ReplaceBotNameInButtons(item.Buttons, new, options)
		msg.Sections[idx].PlaintextInputs = ReplaceBotNameInLabels(item.PlaintextInputs, new, options)
		msg.Sections[idx].Selects = ReplaceBotNameInSelects(item.Selects, new, options)
		msg.Sections[idx].MultiSelect = ReplaceBotNameInMultiSelect(item.MultiSelect, new, options)

		msg.Sections[idx].Base = ReplaceBotNameInBase(item.Base, new)
		msg.Sections[idx].TextFields = ReplaceBotNameInTextFields(item.TextFields, new)
		msg.Sections[idx].Context = ReplaceBotNameInContextItems(item.Context, new)
	}

	msg.PlaintextInputs = ReplaceBotNameInLabels(msg.PlaintextInputs, new, options)
	msg.BaseBody = ReplaceBotNameInBody(msg.BaseBody, new)
}

// ReplaceBotNameInButtons replaces bot name placeholder with a given name.
func ReplaceBotNameInButtons(btns Buttons, name string, opts BotNameOptions) Buttons {
	for i, item := range btns {
		btns[i].Command = commandReplaceWithAppend(item.Command, name, opts)
		btns[i].Description = replace(btns[i].Description, name)
		btns[i].Name = replace(btns[i].Name, name)
	}
	return btns
}

// ReplaceBotNameInLabels replaces bot name placeholder with a given name.
func ReplaceBotNameInLabels(labels LabelInputs, name string, opts BotNameOptions) LabelInputs {
	for i, item := range labels {
		labels[i].Command = commandReplacePrepend(item.Command, name, opts)
		labels[i].Text = replace(item.Text, name)
		labels[i].Placeholder = replace(item.Placeholder, name)
	}
	return labels
}

// ReplaceBotNameInSelects replaces bot name placeholder with a given name.
func ReplaceBotNameInSelects(selects Selects, name string, opts BotNameOptions) Selects {
	for i, item := range selects.Items {
		selects.Items[i].Command = commandReplacePrepend(item.Command, name, opts)
		selects.Items[i].Name = replace(item.Name, name)
		selects.Items[i].OptionGroups = ReplaceBotNameInOptionGroups(item.OptionGroups, name)
		selects.Items[i].InitialOption = ReplaceBotNameInOptionItem(item.InitialOption, name)
	}
	return selects
}

// ReplaceBotNameInMultiSelect replaces bot name placeholder with a given name.
func ReplaceBotNameInMultiSelect(ms MultiSelect, name string, opts BotNameOptions) MultiSelect {
	ms.Command = commandReplacePrepend(ms.Command, name, opts)
	ms.Name = replace(ms.Name, name)
	ms.Description = ReplaceBotNameInBody(ms.Description, name)
	ms.InitialOptions = ReplaceBotNameInOptions(ms.InitialOptions, name)
	ms.Options = ReplaceBotNameInOptions(ms.Options, name)
	return ms
}

// ReplaceBotNameInBase replaces bot name placeholder with a given name.
func ReplaceBotNameInBase(base Base, name string) Base {
	base.Description = replace(base.Description, name)
	base.Header = replace(base.Header, name)
	base.Body = ReplaceBotNameInBody(base.Body, name)
	return base
}

// ReplaceBotNameInBody replaces bot name placeholder with a given name.
func ReplaceBotNameInBody(body Body, name string) Body {
	body.Plaintext = replace(body.Plaintext, name)
	body.CodeBlock = replace(body.CodeBlock, name)
	return body
}

// ReplaceBotNameInTextFields replaces bot name placeholder with a given name.
func ReplaceBotNameInTextFields(fields TextFields, name string) TextFields {
	for i, item := range fields {
		fields[i].Value = replace(item.Value, name)
		fields[i].Key = replace(item.Key, name)
	}
	return fields
}

// ReplaceBotNameInContextItems replaces bot name placeholder with a given name.
func ReplaceBotNameInContextItems(items ContextItems, name string) ContextItems {
	for i, item := range items {
		items[i].Text = replace(item.Text, name)
	}
	return items
}

// ReplaceBotNameInOptionItem replaces bot name placeholder with a given name.
func ReplaceBotNameInOptionItem(item *OptionItem, name string) *OptionItem {
	if item == nil {
		return nil
	}
	item.Name = replace(item.Name, name)
	item.Value = replace(item.Value, name)
	return item
}

// ReplaceBotNameInOptions replaces bot name placeholder with a given name.
func ReplaceBotNameInOptions(items []OptionItem, name string) []OptionItem {
	for i, item := range items {
		items[i].Name = replace(item.Name, name)
		items[i].Value = replace(item.Value, name)
	}
	return items
}

// ReplaceBotNameInOptionGroups replaces bot name placeholder with a given name.
func ReplaceBotNameInOptionGroups(groups []OptionGroup, name string) []OptionGroup {
	for i, item := range groups {
		groups[i].Name = replace(item.Name, name)
		groups[i].Options = ReplaceBotNameInOptions(item.Options, name)
	}
	return groups
}

func replace(text, new string) string {
	return strings.Replace(text, MessageBotNamePlaceholder, new, maxReplaceNo)
}

func commandReplaceWithAppend(cmd, botName string, opts BotNameOptions) string {
	if cmd == "" {
		return cmd
	}
	if !strings.Contains(cmd, MessageBotNamePlaceholder) {
		return cmd
	}

	cmd = replace(cmd, botName)
	if opts.ClusterName == "" {
		return cmd
	}

	return fmt.Sprintf("%s --cluster-name=%q ", cmd, opts.ClusterName)
}

func commandReplacePrepend(cmd, botName string, opts BotNameOptions) string {
	if cmd == "" {
		return cmd
	}
	cmd = replace(cmd, botName)

	if !strings.Contains(cmd, MessageBotNamePlaceholder) {
		return cmd
	}
	if opts.ClusterName == "" {
		return cmd
	}

	parts := strings.SplitAfterN(cmd, botName, 2)
	switch len(parts) {
	case 0, 1: // if there is no bot name we don't need to add cluster name as this command won't be never executed against our instance
		return cmd
	default:
		// we need to append the --cluster-name flag right after the `@Botkube {plugin_name}`
		// As a result, we won't break the order of other flags.

		tokenized := strings.Fields(parts[1])
		if len(tokenized) < 2 {
			return cmd
		}

		pluginName := tokenized[0]
		// we cannot do `strings.Join` on tokenized slice, as we need to preserve all whitespaces that where declared by plugin
		// e.g. `--filter=` is different from `--filter ` and we don't know which one was used, so we can break it if we won't preserve the space.
		restMessage := strings.TrimPrefix(tokenized[0], parts[1])

		cmd = fmt.Sprintf("%s %s --cluster-name=%q %s", botName, pluginName, opts.ClusterName, restMessage)
	}

	return cmd
}
