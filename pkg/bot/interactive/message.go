package interactive

import (
	"fmt"
)

// ButtonStyle is a style of Button element.
type ButtonStyle string

// Represents a general button styles.
const (
	ButtonStyleDefault ButtonStyle = ""
	ButtonStylePrimary ButtonStyle = "primary"
	ButtonStyleDanger  ButtonStyle = "danger"
)

// SelectType is a type of Button element.
type SelectType string

// Represents a select dropdown types.
const (
	StaticSelect   SelectType = "static"
	ExternalSelect SelectType = "external"
)

// MessageType defines the message type.
type MessageType string

const (
	// Default defines a message that should be displayed in default mode supported by communicator.
	Default MessageType = ""
	// Popup defines a message that should be displayed to the user as popup (if possible).
	Popup MessageType = "form"
)

// PlainTextType defines the label as plain_text
type PlainTextType string

// PlainTextInputType defines the element as plain_text_input
type PlainTextInputType string

const (
	// PlainText refers to plain_text
	PlainText PlainTextType = "plain_text"
	// PlainTextInput refers to plain_text for elements
	PlainTextInput PlainTextInputType = "plain_text_input"
)

// Message represents a generic message with interactive buttons.
type Message struct {
	Type MessageType
	Base
	Sections          []Section
	Inputs            []Input
	OnlyVisibleForYou bool
	ReplaceOriginal   bool
}

// HasSections returns true if message has interactive sections.
func (msg *Message) HasSections() bool {
	return len(msg.Sections) != 0
}

// HasInputs returns true if message has interactive inputs.
func (msg *Message) HasInputs() bool {
	return len(msg.Inputs) != 0
}

// Select holds data related to the select drop-down.
type Select struct {
	Type    SelectType
	Name    string
	Command string
	// OptionGroups provides a way to group options in a select menu.
	OptionGroups []OptionGroup
	// InitialOption holds already pre-selected options. MUST be a sub-set of OptionGroups.
	InitialOption *OptionItem
}

// Base holds generic message fields.
type Base struct {
	Header      string
	Description string
	Body        Body
}

// Body holds message body fields.
type Body struct {
	CodeBlock string
	Plaintext string
}

// Section holds section related fields.
type Section struct {
	Base
	Buttons     Buttons
	MultiSelect MultiSelect
	Selects     Selects
	TextFields  TextFields
	Context     ContextItems
}

// ContextItems holds context items.
type ContextItems []ContextItem

// TextFields holds text field items.
type TextFields []TextField

// TextField holds a text field data.
type TextField struct {
	Text string
}

// IsDefined returns true if there are any context items defined.
func (c ContextItems) IsDefined() bool {
	return len(c) > 0
}

// ContextItem holds context item.
type ContextItem struct {
	Text string
}

// Selects holds multiple Select objects.
type Selects struct {
	// ID allows to identify a given block when we do the updated.
	ID    string
	Items []Select
}

// Input is used to create input elements to use in slack messages.
type Input struct {
	ID               string
	DispatchedAction bool
	Element          InputElement
	Label            InputLabel
}

// InputElement is one of the components of Input. This component is mostly used to hold elements like input text, etc...
type InputElement struct {
	Type PlainTextInputType
}

// InputLabel refers to label of input element
type InputLabel struct {
	Type PlainTextType
	Text string
}

// AreOptionsDefined returns true if some options are available.
func (s *Selects) AreOptionsDefined() bool {
	if s == nil {
		return false
	}
	return len(s.Items) > 0
}

// OptionItem defines an option model.
type OptionItem struct {
	Name  string
	Value string
}

// MultiSelect holds multi select related fields.
type MultiSelect struct {
	Name        string
	Description Body
	Command     string

	// Options holds all available options
	Options []OptionItem

	// InitialOptions hold already pre-selected options. MUST be a sub-set of Options.
	InitialOptions []OptionItem
}

// OptionGroup holds information about options in the same group.
type OptionGroup struct {
	Name    string
	Options []OptionItem
}

// AreOptionsDefined returns true if some options are available.
func (m *MultiSelect) AreOptionsDefined() bool {
	if m == nil {
		return false
	}
	if len(m.Options) == 0 {
		return false
	}
	return true
}

// Buttons holds definition of interactive buttons.
type Buttons []Button

// AtLeastOneButtonHasDescription returns true if there is at least one button with description associated with it.
func (s *Buttons) AtLeastOneButtonHasDescription() bool {
	if s == nil {
		return false
	}
	for _, item := range *s {
		if item.Description != "" {
			return true
		}
	}

	return false
}

// Button holds definition of action button.
type Button struct {
	Description string
	Name        string
	Command     string
	URL         string
	Style       ButtonStyle
}

// ButtonBuilder provides a simplified way to construct a Button model.
type ButtonBuilder struct {
	BotName string
}

// ForCommandWithDescCmd returns button command where description and command are the same.
func (b *ButtonBuilder) ForCommandWithDescCmd(name, cmd string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}
	return b.commandWithDesc(name, cmd, cmd, bt)
}

// DescriptionURL returns link button with description.
func (b *ButtonBuilder) DescriptionURL(name, cmd string, url string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}

	return Button{
		Name:        name,
		Description: fmt.Sprintf("%s %s", b.BotName, cmd),
		URL:         url,
		Style:       bt,
	}
}

// ForCommandWithoutDesc returns button command without description.
func (b *ButtonBuilder) ForCommandWithoutDesc(name, cmd string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}
	cmd = fmt.Sprintf("%s %s", b.BotName, cmd)
	return Button{
		Name:    name,
		Command: cmd,
		Style:   bt,
	}
}

// ForCommand returns button command.
func (b *ButtonBuilder) ForCommand(name, cmd, desc string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}
	cmd = fmt.Sprintf("%s %s", b.BotName, cmd)
	desc = fmt.Sprintf("%s %s", b.BotName, desc)
	return Button{
		Name:        name,
		Command:     cmd,
		Description: desc,
		Style:       bt,
	}
}

// ForURL returns link button.
func (b *ButtonBuilder) ForURL(name, url string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}

	return Button{
		Name:  name,
		URL:   url,
		Style: bt,
	}
}

func (b *ButtonBuilder) commandWithDesc(name, cmd, desc string, style ButtonStyle) Button {
	cmd = fmt.Sprintf("%s %s", b.BotName, cmd)
	desc = fmt.Sprintf("%s %s", b.BotName, desc)
	return Button{
		Name:        name,
		Command:     cmd,
		Description: desc,
		Style:       style,
	}
}
