package kubernetes

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/source/kubernetes/commander"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

var emojiForLevel = map[config.Level]string{
	config.Info:     "🟢",
	config.Warn:     "⚠️",
	config.Debug:    "ℹ️",
	config.Error:    "❗",
	config.Critical: "❗",
}

type EventCommandsGetter interface {
	GetCommandsForEvent(event event.Event) ([]commander.Command, error)
}
type MessageBuilder struct {
	commandsGetter           EventCommandsGetter
	log                      logrus.FieldLogger
	isInteractivitySupported bool
}

func NewMessageBuilder(isInteractivitySupported bool, log logrus.FieldLogger, commandsGetter EventCommandsGetter) *MessageBuilder {
	return &MessageBuilder{
		commandsGetter:           commandsGetter,
		log:                      log,
		isInteractivitySupported: isInteractivitySupported,
	}
}

func (m *MessageBuilder) FromEvent(event event.Event) (api.Message, error) {
	msg := api.Message{
		Timestamp: event.TimeStamp,
		Sections: []api.Section{
			m.baseNotificationSection(event),
		},
	}

	if !m.isInteractivitySupported {
		msg.Type = api.NonInteractiveSingleSection
		return msg, nil
	}

	cmdSection, err := m.getInteractiveEventSectionIfShould(event)
	if err != nil {
		return api.Message{}, err
	}
	if cmdSection != nil {
		msg.Sections = append(msg.Sections, *cmdSection)
	}

	return msg, nil
}

func (m *MessageBuilder) getInteractiveEventSectionIfShould(event event.Event) (*api.Section, error) {
	commands, err := m.commandsGetter.GetCommandsForEvent(event)
	if err != nil {
		return nil, fmt.Errorf("while getting commands for event: %w", err)
	}

	if len(commands) == 0 {
		return nil, nil
	}

	cmdPrefix := fmt.Sprintf("%s kubectl", api.MessageBotNamePlaceholder)
	var optionItems []api.OptionItem
	for _, cmd := range commands {
		optionItems = append(optionItems, api.OptionItem{
			Name:  cmd.Name,
			Value: cmd.Cmd,
		})
	}
	section := interactive.EventCommandsSection(cmdPrefix, optionItems)
	return &section, nil
}

func (m *MessageBuilder) baseNotificationSection(event event.Event) api.Section {
	section := api.Section{
		Base: api.Base{
			Header: fmt.Sprintf("%s %s", emojiForLevel[event.Level], event.Title),
		},
	}

	section.TextFields = m.appendTextFieldIfNotEmpty(section.TextFields, "Kind", event.Kind)
	section.TextFields = m.appendTextFieldIfNotEmpty(section.TextFields, "Name", event.Name)
	section.TextFields = m.appendTextFieldIfNotEmpty(section.TextFields, "Namespace", event.Namespace)
	section.TextFields = m.appendTextFieldIfNotEmpty(section.TextFields, "Reason", event.Reason)
	section.TextFields = m.appendTextFieldIfNotEmpty(section.TextFields, "Action", event.Action)
	section.TextFields = m.appendTextFieldIfNotEmpty(section.TextFields, "Cluster", event.Cluster)

	// Messages, Recommendations and Warnings formatted as bullet point lists.
	section.BulletLists = m.appendBulletListIfNotEmpty(section.BulletLists, "Messages", event.Messages)
	section.BulletLists = m.appendBulletListIfNotEmpty(section.BulletLists, "Recommendations", event.Recommendations)
	section.BulletLists = m.appendBulletListIfNotEmpty(section.BulletLists, "Warnings", event.Warnings)

	return section
}

func (m *MessageBuilder) appendTextFieldIfNotEmpty(fields api.TextFields, title, value string) []api.TextField {
	if value == "" {
		return fields
	}
	return append(fields, api.TextField{
		Key:   title,
		Value: value,
	})
}

func (m *MessageBuilder) appendBulletListIfNotEmpty(bulletLists api.BulletLists, title string, items []string) api.BulletLists {
	if len(items) == 0 {
		return bulletLists
	}
	return append(bulletLists, api.BulletList{
		Title: title,
		Items: items,
	})
}
