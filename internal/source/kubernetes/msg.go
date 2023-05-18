package kubernetes

import (
	"bytes"
	"fmt"
	"text/template"

	sprig "github.com/go-task/slim-sprig"
	"github.com/sirupsen/logrus"
	"k8s.io/kubectl/pkg/util/slice"

	"github.com/kubeshop/botkube/internal/source/kubernetes/commander"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	event "github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/pkg/api"
)

var emojiForLevel = map[config.Level]string{
	config.Success: "🟢",
	config.Info:    "💡",
	config.Error:   "❗",
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

func (m *MessageBuilder) FromEvent(event event.Event, actions []config.Action) (api.Message, error) {
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

	cmdSection, err := m.getCommandSelectIfShould(event)
	if err != nil {
		return api.Message{}, err
	}

	btns, err := m.getExternalActions(actions, event)
	if err != nil {
		return api.Message{}, err
	}
	if cmdSection != nil || len(btns) > 0 {
		msg.Sections = append(msg.Sections, api.Section{
			Buttons: btns,
			Selects: ptrSection(cmdSection),
		})
	}
	return msg, nil
}

func (m *MessageBuilder) getExternalActions(actions []config.Action, e event.Event) (api.Buttons, error) {
	var actBtns api.Buttons
	for _, act := range actions {
		if !slice.ContainsString(act.Trigger.Type, e.Type.String(), nil) {
			continue
		}

		btn, err := m.renderActionButton(act, e)
		if err != nil {
			return nil, err
		}
		actBtns = append(actBtns, btn)
	}

	return actBtns, nil
}

func ptrSection(s *api.Selects) api.Selects {
	if s == nil {
		return api.Selects{}
	}
	return *s
}

func (m *MessageBuilder) getCommandSelectIfShould(event event.Event) (*api.Selects, error) {
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
	return &api.Selects{
		ID: "",
		Items: []api.Select{
			{
				Name:    "Run command...",
				Command: cmdPrefix,
				OptionGroups: []api.OptionGroup{
					{
						Name:    "Supported commands",
						Options: optionItems,
					},
				},
			},
		},
	}, nil
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
	section.TextFields = m.appendTextFieldIfNotEmpty(section.TextFields, "Type", event.Reason)
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

func (m *MessageBuilder) renderActionButton(act config.Action, e event.Event) (api.Button, error) {
	tmpl, err := template.New("example").Funcs(sprig.FuncMap()).Parse(act.Button.CommandTpl)
	if err != nil {
		return api.Button{}, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, e)
	if err != nil {
		return api.Button{}, err
	}

	btns := api.NewMessageButtonBuilder()
	return btns.ForCommandWithoutDesc(act.Button.DisplayName, buf.String(), api.ButtonStylePrimary), nil
}
