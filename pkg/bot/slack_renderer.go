package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/formatx"
)

const (
	urlButtonActionIDPrefix = "url:"
	cmdButtonActionIDPrefix = "cmd:"
	maxActionIDLen          = 254
)

// SlackRenderer provides functionality to render Slack specific messages from a generic models.
type SlackRenderer struct {
	mdFormatter interactive.MDFormatter
}

// NewSlackRenderer returns new SlackRenderer instance.
func NewSlackRenderer() *SlackRenderer {
	return &SlackRenderer{
		mdFormatter: interactive.NewMDFormatter(interactive.NewlineFormatter, func(msg string) string {
			return fmt.Sprintf("*%s*", msg)
		}),
	}
}

// MessageToMarkdown renders message in Markdown format.
func (b *SlackRenderer) MessageToMarkdown(in interactive.CoreMessage) string {
	return interactive.RenderMessage(b.mdFormatter, in)
}

// RenderModal returns a modal request view based on a given message.
func (b *SlackRenderer) RenderModal(msg interactive.CoreMessage) slack.ModalViewRequest {
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
func (b *SlackRenderer) RenderInteractiveMessage(msg interactive.CoreMessage) slack.MsgOption {
	if msg.HasSections() || msg.HasInputs() {
		blocks := b.RenderAsSlackBlocks(msg)
		return slack.MsgOptionBlocks(blocks...)
	}
	return b.renderAsSimpleTextSection(msg)
}

// RenderAsSlackBlocks returns the Slack message blocks for a given input message.
func (b *SlackRenderer) RenderAsSlackBlocks(msg interactive.CoreMessage) []slack.Block {
	var blocks []slack.Block
	if msg.Header != "" {
		blocks = append(blocks, b.mdTextSection("*%s*", msg.Header))
	}

	if msg.Description != "" {
		blocks = append(blocks, b.mdTextSection(msg.Description))
	}

	if msg.BaseBody.Plaintext != "" {
		blocks = append(blocks, b.mdTextSection(msg.BaseBody.Plaintext))
	}

	if msg.BaseBody.CodeBlock != "" {
		blocks = append(blocks, b.mdTextSection(formatx.AdaptiveCodeBlock(msg.BaseBody.CodeBlock)))
	}

	all := len(msg.Sections)
	for idx, s := range msg.Sections {
		blocks = append(blocks, b.renderSection(s)...)
		if !(idx == all-1) { // if not the last one, append divider
			blocks = append(blocks, slack.NewDividerBlock())
		}
	}
	for _, i := range msg.PlaintextInputs {
		blocks = append(blocks, b.renderInput(i))
	}

	if !msg.Timestamp.IsZero() {
		fallbackTimestampText := msg.Timestamp.Format(time.RFC1123)
		timestampText := fmt.Sprintf("<!date^%d^{date_num} {time_secs}|%s>", msg.Timestamp.Unix(), fallbackTimestampText)
		blocks = append(blocks, b.renderContext([]api.ContextItem{{
			Text: timestampText,
		}})...)
	}

	return blocks
}

func (b *SlackRenderer) renderSelects(s api.Selects) slack.Block {
	var elems []slack.BlockElement
	for _, s := range s.Items {
		placeholder := slack.NewTextBlockObject(slack.PlainTextType, s.Name, false, false)
		singleSelect := slack.NewOptionsSelectBlockElement(convertToSlackSelectType(s.Type), placeholder, s.Command)

		if singleSelect.Type == slack.OptTypeExternal {
			// override the default 3 characters. In this way, the call to our backend is triggered even if user only
			// opens a given dropdown and not when he types at least 3 characters.
			minLen := 0
			singleSelect.MinQueryLength = &minLen
		}

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

func (b *SlackRenderer) renderAsSimpleTextSection(msg interactive.CoreMessage) slack.MsgOption {
	var out strings.Builder
	if msg.Header != "" {
		out.WriteString(msg.Header + "\n")
	}
	if msg.Description != "" {
		out.WriteString(msg.Description + "\n")
	}

	if msg.BaseBody.Plaintext != "" {
		out.WriteString(msg.BaseBody.Plaintext)
	}

	if msg.BaseBody.CodeBlock != "" {
		// we don't use the AdaptiveCodeBlock as we want to have a code block even for single lines
		// to make it more readable in the wide view.
		out.WriteString(formatx.CodeBlock(msg.BaseBody.CodeBlock))
	}

	return slack.MsgOptionText(out.String(), false)
}

func (b *SlackRenderer) renderSection(in api.Section) []slack.Block {
	var out []slack.Block
	if in.Header != "" {
		out = append(out, b.mdTextSection("*%s*", in.Header))
	}

	if in.Description != "" {
		out = append(out, b.mdTextSection(in.Description))
	}

	if len(in.TextFields) > 0 {
		out = append(out, b.renderTextFields(in.TextFields))
	}

	if in.Body.Plaintext != "" {
		out = append(out, b.mdTextSection(in.Body.Plaintext))
	}

	if in.Body.CodeBlock != "" {
		out = append(out, b.mdTextSection(formatx.AdaptiveCodeBlock(in.Body.CodeBlock)))
	}

	for _, item := range in.PlaintextInputs {
		out = append(out, b.renderInput(item))
	}

	if in.BulletLists.AreItemsDefined() {
		out = append(out, b.renderBulletLists(in.BulletLists))
	}

	out = append(out, b.renderButtons(in.Buttons)...)
	if in.MultiSelect.AreOptionsDefined() {
		sec := b.renderMultiselectWithDescription(in.MultiSelect)
		out = append(out, sec)
	}

	if in.Selects.AreOptionsDefined() {
		out = append(out, b.renderSelects(in.Selects))
	}

	if len(in.Context) > 0 {
		out = append(out, b.renderContext(in.Context)...)
	}

	return out
}

func (b *SlackRenderer) renderTextFields(in api.TextFields) slack.Block {
	var textBlockObjs []*slack.TextBlockObject
	for _, item := range in {
		if item.IsEmpty() {
			// Skip empty sections
			continue
		}

		field := fmt.Sprintf("*%s:* %s", item.Key, item.Value)
		textBlockObjs = append(textBlockObjs, slack.NewTextBlockObject(slack.MarkdownType, field, false, false))
	}

	return slack.NewSectionBlock(
		nil,
		textBlockObjs,
		nil,
	)
}

func (b *SlackRenderer) renderContext(in []api.ContextItem) []slack.Block {
	var blocks []slack.Block

	for _, item := range in {
		if item.Text == "" {
			// Skip empty sections
			continue
		}

		blocks = append(blocks, slack.NewContextBlock(
			"",
			slack.NewTextBlockObject(slack.MarkdownType, item.Text, false, false),
		))
	}

	return blocks
}

// renderButtons renders button section.
//
//  1. With description, renders one per row. For example:
//     `@Botkube get pods` [Button "Get Pods"]
//     `@Botkube get deploys` [Button "Get Deployments"]
//
//  2. Without description: all in the same row. For example:
//     [Button "Get Pods"] [Button "Get Deployments"]
func (b *SlackRenderer) renderButtons(in api.Buttons) []slack.Block {
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

func (b *SlackRenderer) renderButtonsWithDescription(in api.Buttons) []slack.Block {
	var out []slack.Block
	for _, btn := range in {
		desc := btn.Description
		switch btn.DescriptionStyle {
		case api.ButtonDescriptionStyleBold:
			desc = fmt.Sprintf("*%s*", desc)
		case api.ButtonDescriptionStyleCode:
			fallthrough
		default:
			// keep backward compatibility
			desc = formatx.AdaptiveCodeBlock(desc)
		}

		out = append(out, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, desc, false, false),
			nil,
			slack.NewAccessory(b.renderButton(btn)),
		))
	}
	return out
}

func (b *SlackRenderer) renderInput(s api.LabelInput) slack.Block {
	var placeholder *slack.TextBlockObject
	if s.Placeholder != "" {
		placeholder = slack.NewTextBlockObject(slack.PlainTextType, s.Placeholder, false, false)
	}

	// label is required
	var label = slack.NewTextBlockObject(slack.PlainTextType, "Input", false, false)
	if s.Text != "" {
		label = slack.NewTextBlockObject(slack.PlainTextType, s.Text, false, false)
	}

	input := slack.NewPlainTextInputBlockElement(placeholder, s.Command)
	block := slack.NewInputBlock(s.Command, label, nil, input)

	if s.DispatchedAction != "" {
		input.DispatchActionConfig = &slack.DispatchActionConfig{
			TriggerActionsOn: []string{string(s.DispatchedAction)},
		}
		block.DispatchAction = true
	}

	return block
}

func (b *SlackRenderer) renderMultiselectWithDescription(in api.MultiSelect) slack.Block {
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

func (b *SlackRenderer) renderButton(btn api.Button) slack.BlockElement {
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

// id must be less than 256 characters, see:
// https://api.slack.com/reference/block-kit/block-elements#button
func (b *SlackRenderer) genBtnActionID(btn api.Button) string {
	if btn.Command != "" {
		stop := b.intWithMax(maxActionIDLen-len(cmdButtonActionIDPrefix), len(btn.Command))
		return cmdButtonActionIDPrefix + btn.Command[:stop]
	}

	// must be less than 256 characters
	stop := b.intWithMax(maxActionIDLen-len(urlButtonActionIDPrefix), len(btn.URL))
	return urlButtonActionIDPrefix + btn.URL[:stop]
}

func (*SlackRenderer) intWithMax(a, max int) int {
	if a > max {
		return max
	}
	return a
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

func (b *SlackRenderer) renderBulletLists(in api.BulletLists) slack.Block {
	strBuilder := strings.Builder{}
	for _, item := range in {
		if len(item.Items) == 0 {
			continue
		}
		strBuilder.WriteString(fmt.Sprintf("*%s*\n%s", item.Title, formatx.BulletPointListFromMessages(item.Items)))
		strBuilder.WriteString("\n")
	}

	return b.mdTextSection(strBuilder.String())
}

func convertToSlackStyle(in api.ButtonStyle) slack.Style {
	switch in {
	case api.ButtonStyleDefault:
		return slack.StyleDefault
	case api.ButtonStylePrimary:
		return slack.StylePrimary
	case api.ButtonStyleDanger:
		return slack.StyleDanger
	}
	return slack.StyleDefault
}

func convertToSlackSelectType(in api.SelectType) string {
	switch in {
	case api.StaticSelect:
		return slack.OptTypeStatic
	case api.ExternalSelect:
		return slack.OptTypeExternal
	}
	return slack.OptTypeStatic
}
