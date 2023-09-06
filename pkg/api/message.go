package api

import (
	"fmt"
	"time"
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
	// DefaultMessage defines a message that should be displayed in default mode supported by communicator.
	DefaultMessage MessageType = ""
	// BaseBodyWithFilterMessage defines a message that should be displayed in plaintext mode supported by communicator.
	// In this form the built-in filter is supported.
	// NOTE: only BaseBody is preserved. All other properties are ignored even if set.
	BaseBodyWithFilterMessage MessageType = "baseBodyWithFilter"
	// NonInteractiveSingleSection it is an indicator for non-interactive platforms, that they can render this event
	// even though they have limited capability. As a result, a given message has the following restriction:
	//  - the whole message should have exactly one section
	//  - section interactive elements such as buttons, select, multiselect, and inputs are ignored.
	//  - the base body of the message is ignored
	//  - Timestamp field is optional
	NonInteractiveSingleSection MessageType = "nonInteractiveEventSingleSection"
	// PopupMessage defines a message that should be displayed to the user as popup (if possible).
	PopupMessage MessageType = "form"
)

// Message represents a generic message with interactive buttons.
type Message struct {
	Type              MessageType `json:"type,omitempty"`
	BaseBody          Body        `json:"baseBody,omitempty"`
	Timestamp         time.Time   `json:"timestamp,omitempty"`
	Sections          []Section   `json:"sections,omitempty"`
	PlaintextInputs   LabelInputs `json:"plaintextInputs,omitempty"`
	OnlyVisibleForYou bool        `json:"onlyVisibleForYou,omitempty"`
	ReplaceOriginal   bool        `json:"replaceOriginal,omitempty"`
}

func (msg *Message) IsEmpty() bool {
	var emptyBase Body
	if msg.BaseBody != emptyBase {
		return false
	}
	if msg.HasInputs() {
		return false
	}
	if msg.HasSections() {
		return false
	}
	if !msg.Timestamp.IsZero() {
		return false
	}

	return true
}

// HasSections returns true if message has interactive sections.
func (msg *Message) HasSections() bool {
	return len(msg.Sections) != 0
}

// HasInputs returns true if message has interactive inputs.
func (msg *Message) HasInputs() bool {
	return len(msg.PlaintextInputs) != 0
}

// Select holds data related to the select drop-down.
type Select struct {
	Type    SelectType `json:"type,omitempty"`
	Name    string     `json:"name,omitempty"`
	Command string     `json:"command,omitempty"`
	// OptionGroups provides a way to group options in a select menu.
	OptionGroups []OptionGroup `json:"optionGroups,omitempty"`
	// InitialOption holds already pre-selected options. MUST be a sub-set of OptionGroups.
	InitialOption *OptionItem `json:"initialOption,omitempty"`
}

// Base holds generic message fields.
type Base struct {
	Header      string `json:"header,omitempty"`
	Description string `json:"description,omitempty"`
	Body        Body   `json:"body,omitempty"`
}

// Body holds message body fields.
type Body struct {
	CodeBlock string `json:"codeBlock,omitempty"`
	Plaintext string `json:"plaintext,omitempty"`
}

// Section holds section related fields.
type Section struct {
	Base            `json:",inline"`
	Buttons         Buttons      `json:"buttons,omitempty"`
	MultiSelect     MultiSelect  `json:"multiSelect,omitempty"`
	Selects         Selects      `json:"selects,omitempty"`
	PlaintextInputs LabelInputs  `json:"plaintextInputs,omitempty"`
	TextFields      TextFields   `json:"textFields,omitempty"`
	BulletLists     BulletLists  `json:"bulletLists,omitempty"`
	Context         ContextItems `json:"context,omitempty"`
}

// BulletLists holds the bullet lists.
type BulletLists []BulletList

// AreItemsDefined returns true if at least one list has items defined.
func (l BulletLists) AreItemsDefined() bool {
	for _, list := range l {
		if len(list.Items) > 0 {
			return true
		}
	}
	return false
}

// LabelInputs holds the plain text input items.
type LabelInputs []LabelInput

// ContextItems holds context items.
type ContextItems []ContextItem

// TextFields holds text field items.
type TextFields []TextField

// TextField holds a text field data.
type TextField struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// IsEmpty returns true if all fields have zero-value.
func (t *TextField) IsEmpty() bool {
	return t.Value == "" && t.Key == ""
}

// BulletList defines a bullet list primitive.
type BulletList struct {
	Title string   `json:"title,omitempty"`
	Items []string `json:"items,omitempty"`
}

// IsDefined returns true if there are any context items defined.
func (c ContextItems) IsDefined() bool {
	return len(c) > 0
}

// ContextItem holds context item.
type ContextItem struct {
	Text string `json:"text,omitempty"`
}

// Selects holds multiple Select objects.
type Selects struct {
	// ID allows to identify a given block when we do the updated.
	ID    string   `json:"id,omitempty"`
	Items []Select `json:"items,omitempty"`
}

// DispatchedInputAction defines when the action should be sent to our backend.
type DispatchedInputAction string

// Defines the possible options to dispatch the input action.
const (
	NoDispatchInputAction          DispatchedInputAction = ""
	DispatchInputActionOnEnter     DispatchedInputAction = "on_enter_pressed"
	DispatchInputActionOnCharacter DispatchedInputAction = "on_character_entered"
)

// LabelInput is used to create input elements to use in messages.
type LabelInput struct {
	Command          string                `json:"command,omitempty"`
	Text             string                `json:"text,omitempty"`
	Placeholder      string                `json:"placeholder,omitempty"`
	DispatchedAction DispatchedInputAction `json:"dispatchedAction,omitempty"`
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
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// MultiSelect holds multi select related fields.
type MultiSelect struct {
	Name        string `json:"name,omitempty"`
	Description Body   `json:"description,omitempty"`
	Command     string `json:"command,omitempty"`

	// Options holds all available options
	Options []OptionItem `json:"options,omitempty"`

	// InitialOptions hold already pre-selected options. MUST be a sub-set of Options.
	InitialOptions []OptionItem `json:"initialOptions,omitempty"`
}

// OptionGroup holds information about options in the same group.
type OptionGroup struct {
	Name    string       `json:"name,omitempty"`
	Options []OptionItem `json:"options,omitempty"`
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

// ButtonDescriptionStyle defines the style of the button description.
type ButtonDescriptionStyle string

const (
	// ButtonDescriptionStyleBold defines the bold style for the button description.
	ButtonDescriptionStyleBold ButtonDescriptionStyle = "bold"

	// ButtonDescriptionStyleCode defines the code style for the button description.
	ButtonDescriptionStyleCode ButtonDescriptionStyle = "code"
)

// Button holds definition of action button.
type Button struct {
	Description string `json:"description,omitempty"`

	// DescriptionStyle defines the style of the button description. If not provided, the default style (ButtonDescriptionStyleCode) is used.
	DescriptionStyle ButtonDescriptionStyle `json:"descriptionStyle"`

	Name    string      `json:"name,omitempty"`
	Command string      `json:"command,omitempty"`
	URL     string      `json:"url,omitempty"`
	Style   ButtonStyle `json:"style,omitempty"`
}

// ButtonBuilder provides a simplified way to construct a Button model.
type ButtonBuilder struct{}

func NewMessageButtonBuilder() *ButtonBuilder {
	return &ButtonBuilder{}
}

// ForCommandWithDescCmd returns button command where description and command are the same.
func (b *ButtonBuilder) ForCommandWithDescCmd(name, cmd string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}
	return b.commandWithCmdDesc(name, cmd, cmd, bt)
}

// ForCommandWithBoldDesc returns button command where description and command are different.
func (b *ButtonBuilder) ForCommandWithBoldDesc(name, desc, cmd string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}
	return b.commandWithDesc(name, cmd, desc, bt, ButtonDescriptionStyleBold)
}

// DescriptionURL returns link button with description.
func (b *ButtonBuilder) DescriptionURL(name, cmd string, url string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}

	return Button{
		Name:        name,
		Description: fmt.Sprintf("%s %s", MessageBotNamePlaceholder, cmd),
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
	cmd = fmt.Sprintf("%s %s", MessageBotNamePlaceholder, cmd)
	return Button{
		Name:    name,
		Command: cmd,
		Style:   bt,
	}
}

// ForCommand returns button command with description in adaptive code block.
//
// For displaying description in bold, use ForCommandWithBoldDesc.
func (b *ButtonBuilder) ForCommand(name, cmd, desc string, style ...ButtonStyle) Button {
	bt := ButtonStyleDefault
	if len(style) > 0 {
		bt = style[0]
	}
	return b.commandWithCmdDesc(name, cmd, desc, bt)
}

// ForURLWithBoldDesc returns link button with description.
func (b *ButtonBuilder) ForURLWithBoldDesc(name, desc, url string, style ...ButtonStyle) Button {
	urlBtn := b.ForURL(name, url, style...)
	urlBtn.Description = desc
	urlBtn.DescriptionStyle = ButtonDescriptionStyleBold

	return urlBtn
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

func (b *ButtonBuilder) commandWithCmdDesc(name, cmd, desc string, style ButtonStyle) Button {
	desc = fmt.Sprintf("%s %s", MessageBotNamePlaceholder, desc)
	return b.commandWithDesc(name, cmd, desc, style, ButtonDescriptionStyleCode)
}

func (b *ButtonBuilder) commandWithDesc(name, cmd, desc string, style ButtonStyle, descStyle ButtonDescriptionStyle) Button {
	cmd = fmt.Sprintf("%s %s", MessageBotNamePlaceholder, cmd)
	return Button{
		Name:             name,
		Command:          cmd,
		Description:      desc,
		DescriptionStyle: descStyle,
		Style:            style,
	}
}
