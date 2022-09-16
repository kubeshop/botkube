package bot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	formatx "github.com/kubeshop/botkube/pkg/format"
)

// SlackRenderer provides functionality to render Slack specific messages from a generic models.
type SlackRenderer struct {
	notification config.Notification
}

// NewSlackRenderer returns new SlackRenderer instance.
func NewSlackRenderer(notificationType config.Notification) *SlackRenderer {
	return &SlackRenderer{notification: notificationType}
}

// RenderEventMessage returns Slack message based on a given event.
func (b *SlackRenderer) RenderEventMessage(event events.Event) slack.Attachment {
	var attachment slack.Attachment

	switch b.notification.Type {
	case config.LongNotification:
		attachment = b.longNotification(event)
	case config.ShortNotification:
		fallthrough
	default:
		attachment = b.shortNotification(event)
	}

	// Add timestamp
	ts := json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10))
	if ts > "0" {
		attachment.Ts = ts
	}
	attachment.Color = attachmentColor[event.Level]
	return attachment
}

// RenderInteractiveMessage returns Slack message based on the input msg.
func (b *SlackRenderer) RenderInteractiveMessage(msg interactive.Message) slack.MsgOption {
	if msg.HasSections() {
		return slack.MsgOptionBlocks(b.RenderAsSlackBlocks(msg)...)
	}
	return b.renderAsSimpleTextSection(msg)
}

// RenderAsSlackBlocks returns the Slack message blocks for a given input message.
func (b *SlackRenderer) RenderAsSlackBlocks(msg interactive.Message) []slack.Block {
	var blocks []slack.Block
	if msg.Header != "" {
		blocks = append(blocks, b.mdTextSection("*%s*", msg.Header))
	}

	if msg.Description != "" {
		blocks = append(blocks, b.mdTextSection(msg.Description))
	}

	if msg.Body.Plaintext != "" {
		blocks = append(blocks, b.mdTextSection(msg.Body.Plaintext))
	}

	if msg.Body.CodeBlock != "" {
		blocks = append(blocks, b.mdTextSection(formatx.AdaptiveCodeBlock(msg.Body.CodeBlock)))
	}

	all := len(msg.Sections)
	for idx, s := range msg.Sections {
		blocks = append(blocks, b.renderSection(s)...)
		if !(idx == all-1) { // if not the last one, append divider
			blocks = append(blocks, slack.NewDividerBlock())
		}
	}

	return blocks
}

func (b *SlackRenderer) renderAsSimpleTextSection(msg interactive.Message) slack.MsgOption {
	var out strings.Builder
	if msg.Header != "" {
		out.WriteString(msg.Header + "\n")
	}
	if msg.Description != "" {
		out.WriteString(msg.Description + "\n")
	}

	if msg.Body.Plaintext != "" {
		out.WriteString(msg.Body.Plaintext)
	}

	if msg.Body.CodeBlock != "" {
		// we don't use the AdaptiveCodeBlock as we want to have a code block even for single lines
		// to make it more readable in the wide view.
		out.WriteString(formatx.CodeBlock(msg.Body.CodeBlock))
	}

	return slack.MsgOptionText(out.String(), false)
}

func (b *SlackRenderer) renderSection(in interactive.Section) []slack.Block {
	var out []slack.Block
	if in.Header != "" {
		out = append(out, b.mdTextSection("*%s*", in.Header))
	}

	if in.Description != "" {
		out = append(out, b.mdTextSection(in.Description))
	}

	if in.Body.Plaintext != "" {
		out = append(out, b.mdTextSection(in.Body.Plaintext))
	}

	if in.Body.CodeBlock != "" {
		out = append(out, b.mdTextSection(formatx.AdaptiveCodeBlock(in.Body.CodeBlock)))
	}

	out = append(out, b.renderButtons(in.Buttons)...)

	return out
}

// renderButtons renders button section.
//
//  1. With description, renders one per row. For example:
//     `@BotKube get pods` [Button "Get Pods"]
//     `@BotKube get deploys` [Button "Get Deployments"]
//
//  2. Without description: all in the same row. For example:
//     [Button "Get Pods"] [Button "Get Deployments"]
func (b *SlackRenderer) renderButtons(in interactive.Buttons) []slack.Block {
	if len(in) == 0 {
		return nil
	}

	if in.AtLeastOneButtonHasDescription() {
		return b.renderButtonsWithDescription(in)
	}

	var btns []slack.BlockElement
	for _, btn := range in {
		btns = append(btns, b.renderButton(btn))
	}

	return []slack.Block{
		slack.NewActionBlock(
			"",
			btns...,
		),
	}
}

func (b *SlackRenderer) renderButtonsWithDescription(in interactive.Buttons) []slack.Block {
	var out []slack.Block
	for _, btn := range in {
		out = append(out, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, formatx.AdaptiveCodeBlock(btn.Description), false, false),
			nil,
			slack.NewAccessory(b.renderButton(btn)),
		))
	}
	return out
}

func (b *SlackRenderer) renderButton(btn interactive.Button) slack.BlockElement {
	return &slack.ButtonBlockElement{
		Type:     slack.METButton,
		Text:     slack.NewTextBlockObject(slack.PlainTextType, btn.Name, true, false),
		Value:    btn.Command,
		ActionID: b.genBtnActionID(btn),
		Style:    btn.Style,

		// NOTE: We'll still receive an interaction payload and will need to send an acknowledgement response.
		URL: btn.URL,
	}
}

func (b *SlackRenderer) genBtnActionID(btn interactive.Button) string {
	if btn.Command != "" {
		return "cmd:" + btn.Command
	}
	return "url:" + btn.URL
}

func (b *SlackRenderer) mdTextSection(in string, args ...any) *slack.SectionBlock {
	msg := fmt.Sprintf(in, args...)
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, msg, false, false),
		nil, nil,
	)
}

func (b *SlackRenderer) longNotification(event events.Event) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*%s*", event.Title),
		Fields: []slack.AttachmentField{
			{
				Title: "Kind",
				Value: event.Kind,
				Short: true,
			},
			{

				Title: "Name",
				Value: event.Name,
				Short: true,
			},
		},
		Footer: "BotKube",
	}

	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, event.Namespace, "Namespace", true)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, event.Reason, "Reason", true)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, formatx.JoinMessages(event.Messages), "Message", false)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, event.Action, "Action", true)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, formatx.JoinMessages(event.Recommendations), "Recommendations", false)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, formatx.JoinMessages(event.Warnings), "Warnings", false)
	attachment.Fields = b.appendIfNotEmpty(attachment.Fields, event.Cluster, "Cluster", false)

	return attachment
}

func (b *SlackRenderer) appendIfNotEmpty(fields []slack.AttachmentField, in string, title string, short bool) []slack.AttachmentField {
	if in == "" {
		return fields
	}
	return append(fields, slack.AttachmentField{
		Title: title,
		Value: in,
		Short: short,
	})
}

func (b *SlackRenderer) shortNotification(event events.Event) slack.Attachment {
	return slack.Attachment{
		Title: event.Title,
		Fields: []slack.AttachmentField{
			{
				Value: formatx.ShortMessage(event),
			},
		},
		Footer: "BotKube",
	}
}
