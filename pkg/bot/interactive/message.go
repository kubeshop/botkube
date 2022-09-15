package interactive

import (
	"fmt"
)

// Message represents a generic message with interactive buttons.
// NOTE: for now it's unknown how this will be consumed by other communication platforms and API may change in the near future.
type Message struct {
	Base
	Sections []Section
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

// Section holds section related fields.
type Section struct {
	Base
	Buttons Buttons
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
}

// buttonBuilder provides a simplified way to construct a Button model.
type buttonBuilder struct {
	botName string
}

// ForCommandWithDescCmd returns button command where description and command are the same.
func (b *buttonBuilder) ForCommandWithDescCmd(name, cmd string) Button {
	return b.commandWithDesc(name, cmd, cmd)
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
func (b *buttonBuilder) ForURL(name, url string) Button {
	return Button{
		Name: name,
		URL:  url,
	}
}

func (b *buttonBuilder) commandWithDesc(name, cmd, desc string) Button {
	cmd = fmt.Sprintf("%s %s", b.botName, cmd)
	desc = fmt.Sprintf("%s %s", b.botName, desc)
	return Button{
		Name:        name,
		Command:     cmd,
		Description: desc,
	}
}
