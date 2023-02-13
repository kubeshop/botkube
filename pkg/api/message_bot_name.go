package api

import (
	"strings"
)

const (
	// MessageBotNamePlaceholder is a cross-platform placeholder for bot name.
	MessageBotNamePlaceholder = "{{BotName}}"
	maxReplaceNo              = 100
)

// ReplaceBotNamePlaceholder replaces bot name placeholder with a given name.
func (msg *Message) ReplaceBotNamePlaceholder(new string) {
	for idx, item := range msg.Sections {
		msg.Sections[idx].Buttons = ReplaceBotNameInButtons(item.Buttons, new)
		msg.Sections[idx].PlaintextInputs = ReplaceBotNameInLabels(item.PlaintextInputs, new)
		msg.Sections[idx].Selects = ReplaceBotNameInSelects(item.Selects, new)
		msg.Sections[idx].MultiSelect = ReplaceBotNameInMultiSelect(item.MultiSelect, new)

		msg.Sections[idx].Base = ReplaceBotNameInBase(item.Base, new)
		msg.Sections[idx].TextFields = ReplaceBotNameInTextFields(item.TextFields, new)
		msg.Sections[idx].Context = ReplaceBotNameInContextItems(item.Context, new)
	}
	msg.PlaintextInputs = ReplaceBotNameInLabels(msg.PlaintextInputs, new)
	msg.BaseBody = ReplaceBotNameInBody(msg.BaseBody, new)
}

// ReplaceBotNameInButtons replaces bot name placeholder with a given name.
func ReplaceBotNameInButtons(btns Buttons, name string) Buttons {
	for i, item := range btns {
		btns[i].Command = replace(item.Command, name)
		btns[i].Description = replace(btns[i].Description, name)
		btns[i].Name = replace(btns[i].Name, name)
	}
	return btns
}

// ReplaceBotNameInLabels replaces bot name placeholder with a given name.
func ReplaceBotNameInLabels(labels LabelInputs, name string) LabelInputs {
	for i, item := range labels {
		labels[i].Command = replace(item.Command, name)
		labels[i].Text = replace(item.Text, name)
		labels[i].Placeholder = replace(item.Placeholder, name)
	}
	return labels
}

// ReplaceBotNameInSelects replaces bot name placeholder with a given name.
func ReplaceBotNameInSelects(selects Selects, name string) Selects {
	for i, item := range selects.Items {
		selects.Items[i].Command = replace(item.Command, name)
		selects.Items[i].Name = replace(item.Name, name)
		selects.Items[i].OptionGroups = ReplaceBotNameInOptionGroups(item.OptionGroups, name)
		selects.Items[i].InitialOption = ReplaceBotNameInOptionItem(item.InitialOption, name)
	}
	return selects
}

// ReplaceBotNameInMultiSelect replaces bot name placeholder with a given name.
func ReplaceBotNameInMultiSelect(ms MultiSelect, name string) MultiSelect {
	ms.Command = replace(ms.Command, name)
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
		fields[i].Text = replace(item.Text, name)
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
