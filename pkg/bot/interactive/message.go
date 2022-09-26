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

// MessageType defines the message type.
type MessageType string

const (
	// Default defines a message that should be displayed in default mode supported by communicator.
	Default MessageType = ""
	// Popup defines a message that should be displayed to the user as popup (if possible).
	Popup MessageType = "form"
)

// Message represents a generic message with interactive buttons.
type Message struct {
	Type MessageType
	Base
	Sections          []Section
	OnlyVisibleForYou bool
}

// HasSections returns true if message has interactive sections.
func (msg *Message) HasSections() bool {
	return len(msg.Sections) != 0
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

// buttonBuilder provides a simplified way to construct a Button model.
type buttonBuilder struct {
	botName string
}

// ForCommandWithDescCmd returns button command where description and command are the same.
func (b *buttonBuilder) ForCommandWithDescCmd(name, cmd string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}
	return b.commandWithDesc(name, cmd, cmd, bt)
}

func (b *buttonBuilder) DescriptionURL(name, cmd string, url string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}

	return Button{
		Name:        name,
		Description: fmt.Sprintf("%s %s", b.botName, cmd),
		URL:         url,
		Style:       bt,
	}
}

// ForCommand returns button command without description.
func (b *buttonBuilder) ForCommand(name, cmd string) Button {
	cmd = fmt.Sprintf("%s %s", b.botName, cmd)
	return Button{
		Name:    name,
		Command: cmd,
	}
}

// ForURL returns link button.
func (b *buttonBuilder) ForURL(name, url string, style ...ButtonStyle) Button {
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

func (b *buttonBuilder) commandWithDesc(name, cmd, desc string, style ButtonStyle) Button {
	cmd = fmt.Sprintf("%s %s", b.botName, cmd)
	desc = fmt.Sprintf("%s %s", b.botName, desc)
	return Button{
		Name:        name,
		Command:     cmd,
		Description: desc,
		Style:       style,
	}
}
