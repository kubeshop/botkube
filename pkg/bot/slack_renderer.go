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

const (
	urlButtonActionIDPrefix = "url:"
	cmdButtonActionIDPrefix = "cmd:"
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

// RenderModal returns a modal request view based on a given message.
func (b *SlackRenderer) RenderModal(msg interactive.Message) slack.ModalViewRequest {
	title := msg.Header
	msg.Header = ""
	return slack.ModalViewRequest{
		Type:          "modal",
		Title:         b.plainTextBlock(title),
		Submit:        b.plainTextBlock("Apply"),
		Close:         b.plainTextBlock("Cancel"),
		NotifyOnClose: false,
		Blocks: slack.Blocks{
			BlockSet: b.RenderAsSlackBlocks(msg),
		},
	}
}

// RenderInteractiveMessage returns Slack message based on the input msg.
func (b *SlackRenderer) RenderInteractiveMessage(msg interactive.Message) slack.MsgOption {
	if msg.HasSections() {
		blocks := b.RenderAsSlackBlocks(msg)
		return slack.MsgOptionBlocks(blocks...)
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

	//for _, s := range msg.Actions {
	//	blocks = append(blocks, b.renderSelects(s))
	//}

	all := len(msg.Sections)
	for idx, s := range msg.Sections {
		blocks = append(blocks, b.renderSection(s)...)
		if !(idx == all-1) { // if not the last one, append divider
			blocks = append(blocks, slack.NewDividerBlock())
		}
	}

	return blocks
}

func (b *SlackRenderer) renderSelects(s interactive.Selects) slack.Block {
	var elems []slack.BlockElement
	for _, s := range s.Items {
		placeholder := slack.NewTextBlockObject(slack.PlainTextType, s.Name, false, false)
		singleSelect := slack.NewOptionsSelectBlockElement("static_select", placeholder, s.Command)

		for _, group := range s.OptionGroups {
			var slackOptions []*slack.OptionBlockObject
			for _, opt := range group.Options {
				slackOptions = append(slackOptions, slack.NewOptionBlockObject(opt.Value, b.plainTextBlock(opt.Name), nil))
			}
			singleSelect.OptionGroups = append(singleSelect.OptionGroups, slack.NewOptionGroupBlockElement(b.plainTextBlock(group.Name), slackOptions...))
		}

		if opt := s.InitialOption; opt != nil {
			singleSelect.InitialOption = slack.NewOptionBlockObject(opt.Value, b.plainTextBlock(opt.Name), nil)
		}

		elems = append(elems, singleSelect)
	}

	// We use actions as we have only select items that we want to display in a single line.
	// https://api.slack.com/reference/block-kit/blocks#actions
	return slack.NewActionBlock(
		s.ID,
		elems...,
	)
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
	if in.MultiSelect.AreOptionsDefined() {
		sec := b.renderMultiselectWithDescription(in.MultiSelect)
		out = append(out, sec)
	}

	if len(in.Selects.Items) > 0 {
		out = append(out, b.renderSelects(in.Selects))
	}

	return out
}

// renderButtons renders button section.
//
//  1. With description, renders one per row. For example:
//     `@Botkube get pods` [Button "Get Pods"]
//     `@Botkube get deploys` [Button "Get Deployments"]
//
//  2. Without description: all in the same row. For example:
//     [Button "Get Pods"] [Button "Get Deployments"]
func (b *SlackRenderer) renderButtons(in interactive.Buttons) []slack.Block {
	if len(in) == 0 {
		return nil
	}

	if in.AtLeastOneButtonHasDescription() {
		// We use section layout as we also want to add text description
		// https://api.slack.com/reference/block-kit/blocks#section
		return b.renderButtonsWithDescription(in)
	}

	var btns []slack.BlockElement
	for _, btn := range in {
		btns = append(btns, b.renderButton(btn))
	}

	return []slack.Block{
		// We use actions layout as we have only buttons that we want to display in a single line.
		// https://api.slack.com/reference/block-kit/blocks#actions
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

func (b *SlackRenderer) renderMultiselectWithDescription(in interactive.MultiSelect) slack.Block {
	placeholder := slack.NewTextBlockObject(slack.PlainTextType, in.Name, false, false)
	multiSelect := slack.NewOptionsMultiSelectBlockElement("multi_static_select", placeholder, in.Command)

	for _, opt := range in.Options {
		multiSelect.Options = append(multiSelect.Options, slack.NewOptionBlockObject(opt.Value, b.plainTextBlock(opt.Name), nil))
	}

	for _, opt := range in.InitialOptions {
		multiSelect.InitialOptions = append(multiSelect.InitialOptions, slack.NewOptionBlockObject(opt.Value, b.plainTextBlock(opt.Name), nil))
	}

	desc := in.Description.Plaintext
	if in.Description.CodeBlock != "" {
		desc = formatx.AdaptiveCodeBlock(in.Description.CodeBlock)
	}

	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, desc, false, false),
		nil,
		slack.NewAccessory(multiSelect),
	)
}

func (b *SlackRenderer) renderButton(btn interactive.Button) slack.BlockElement {
	return &slack.ButtonBlockElement{
		Type:     slack.METButton,
		Text:     slack.NewTextBlockObject(slack.PlainTextType, btn.Name, true, false),
		Value:    btn.Command,
		ActionID: b.genBtnActionID(btn),
		Style:    convertToSlackStyle(btn.Style),

		// NOTE: We'll still receive an interaction payload and will need to send an acknowledgement response.
		URL: btn.URL,
	}
}

func (b *SlackRenderer) genBtnActionID(btn interactive.Button) string {
	if btn.Command != "" {
		return cmdButtonActionIDPrefix + btn.Command
	}
	return urlButtonActionIDPrefix + btn.URL
}

func (*SlackRenderer) mdTextSection(in string, args ...any) *slack.SectionBlock {
	msg := fmt.Sprintf(in, args...)
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, msg, false, false),
		nil, nil,
	)
}

func (*SlackRenderer) plainTextBlock(msg string) *slack.TextBlockObject {
	return slack.NewTextBlockObject(slack.PlainTextType, msg, false, false)
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
		Footer: "Botkube",
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
		Footer: "Botkube",
	}
}

func convertToSlackStyle(in interactive.ButtonStyle) slack.Style {
	switch in {
	case interactive.ButtonStyleDefault:
		return slack.StyleDefault
	case interactive.ButtonStylePrimary:
		return slack.StylePrimary
	case interactive.ButtonStyleDanger:
		return slack.StyleDanger
	}
	return slack.StyleDefault
}
