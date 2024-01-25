package teamsxold

import (
	"encoding/json"
	"fmt"
	"github.com/kubeshop/botkube/pkg/ptr"
	"github.com/markbates/errx"
	"reflect"
	"regexp"
	"strings"
	"time"

	cards "github.com/DanielTitkov/go-adaptive-cards"
	"github.com/google/uuid"
	"github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/muesli/reflow/ansi"
	"github.com/sirupsen/logrus"
)

const (
	// The cardVersion represents the AdaptiveCard version. Although Teams currently supports version 1.5 (as of 13.12.2023),
	// we've set 1.6, as we use new features already, e.g., the 'refresh.expires' property.
	// When Teams starts supporting v1.6, the expected refresh behavior will automatically apply to older messages as well.
	// For more information, see: https://learn.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference#support-for-adaptive-cards
	cardVersion = "1.6"
	// smallCardMaxSize represents the maximum width of the small card that still fits into standard view without being wrapped.
	// After reaching that point, we switch to 'Full' width.
	// source: https://learn.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format?tabs=adaptive-md%2Cdesktop%2Cconnector-html#full-width-adaptive-card
	smallCardMaxSize = 56
	cardContentType  = "application/vnd.microsoft.card.adaptive"
	// botPrefix represents the prefix for the Bot ID in Teams. The bot's id is formatted as 28:<MicrosoftAppId>.
	// source: https://learn.microsoft.com/en-us/microsoftteams/platform/bots/how-to/conversations/subscribe-to-conversation-events?tabs=dotnet#members-added
	botPrefix = "28:"
	// mentionHandleFmt represents the format for the user/bot mention handle.
	// source: https://learn.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format?tabs=adaptive-md%2Cdesktop%2Cconnector-html#mention-support-within-adaptive-cards
	mentionHandleFmt = "<at>%s</at>"
	inputTextCmdKey  = "command"
	inputTextKeyName = "input-key"
	originKeyName    = "originName"
)

var (
	// mdEmojiTag finds the emoji tags
	mdEmojiTag = regexp.MustCompile(`:(\w+):`)

	// codeBlockRegex finds code blocks
	codeBlockRegex = regexp.MustCompile("`([^`]+)`")
)

// MessageRendererAdapter provides functionality to render MS Teams specific messages from a generic models.
type MessageRendererAdapter struct {
	mdFormatter        interactive.MDFormatter
	uuidGenerator      func() string
	botID              string
	botName            string
	log                logrus.FieldLogger
	logCardMessage     bool
	knownTableCommands *knownTableCommandsChecker
	tableParser        formatx.TableSpace
}

// NewMessageRendererAdapter return a new messageRenderer instance.
func NewMessageRendererAdapter(log logrus.FieldLogger, botID, botName string) *MessageRendererAdapter {
	return &MessageRendererAdapter{
		botID:              fmt.Sprintf("%s%s", botPrefix, botID),
		log:                log.WithField("service", "msg-renderer"),
		botName:            botName,
		logCardMessage:     isDebugEnabled(log),
		tableParser:        formatx.TableSpace{},
		mdFormatter:        interactive.NewMDFormatter(msNewLineFormatter, interactive.MdHeaderFormatter),
		knownTableCommands: newKnownTableCommandsChecker(),
		uuidGenerator: func() string {
			return uuid.NewString()
		},
	}
}

func isDebugEnabled(fieldLogger logrus.FieldLogger) bool {
	log, ok := fieldLogger.(*logrus.Entry)
	if ok {
		return log.Logger.Level >= logrus.DebugLevel
	}
	return false
}

// MessageToMarkdown renders message in Markdown format.
func (r *MessageRendererAdapter) MessageToMarkdown(in interactive.CoreMessage) string {
	return interactive.RenderMessage(r.mdFormatter, in)
}

func (r *MessageRendererAdapter) MDFormatter() interactive.MDFormatter {
	return r.mdFormatter
}

// RenderMessage returns card message based on the input msg.
func (r *MessageRendererAdapter) RenderMessage(msg api.Message) ([]activity.MsgOption, error) {
	return r.RenderCoreMessage(interactive.CoreMessage{Message: msg})
}

// RenderCoreMessage returns card message based on the input msg.
func (r *MessageRendererAdapter) RenderCoreMessage(msg interactive.CoreMessage) ([]activity.MsgOption, error) {
	opts, _, err := r.RenderCoreMessageCardAndOptions(msg)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func (r *MessageRendererAdapter) RenderCoreMessageCardAndOptions(msg interactive.CoreMessage) ([]activity.MsgOption, Card, error) {
	//botHandle := fmt.Sprintf(mentionHandleFmt, r.botName)
	botHandle := r.botName
	//msg.ReplaceBotNamePlaceholder(botHandle)

	// Note: due to problems with code blocks e.g. for `kubectl describe ...` we try to handle that similar to file upload
	// This is a workaround until we will learn how to deal with code blocks in Adaptive Cards.
	if !msg.HasSections() && msg.BaseBody.CodeBlock != "" {
		if r.isSuitableForAlphaTable(msg) {
			return r.renderCard(msg, botHandle, true)
		}

		return r.renderSimplifiedCard(msg, botHandle)
	}
	return r.renderCard(msg, botHandle, false)
}

// MessageToPlaintext renders message in as close as possible plain text format.
func (r *MessageRendererAdapter) MessageToPlaintext(in interactive.CoreMessage) string {
	return interactive.RenderMessage(interactive.MDFormatter{
		NewlineFormatter:           interactive.NewlineFormatter,
		HeaderFormatter:            func(msg string) string { return msg },
		CodeBlockFormatter:         func(msg string) string { return msg },
		AdaptiveCodeBlockFormatter: func(msg string) string { return msg },
	}, in)
}

func (r *MessageRendererAdapter) renderSimplifiedCard(msg interactive.CoreMessage, botHandle string) ([]activity.MsgOption, Card, error) {
	opts := []activity.MsgOption{r.renderAsSimpleTextSection(msg)}

	if len(msg.PlaintextInputs) == 0 {
		return opts, Card{}, nil
	}

	if msg.PlaintextInputs[0].Text == "Filter output" {
		msg.PlaintextInputs[0].Text = "Filter output of the previous message"
	}

	inputs := interactive.CoreMessage{
		Message: api.Message{
			PlaintextInputs: msg.PlaintextInputs,
		},
	}
	card, content, err := r.renderCard(inputs, botHandle, false)
	if err != nil {
		return nil, Card{}, err
	}
	opts = append(opts, card...)
	return opts, content, nil
}
func (r *MessageRendererAdapter) renderCard(msg interactive.CoreMessage, botHandle string, forceTable bool) ([]activity.MsgOption, Card, error) {
	interactiveRenderer := newMessageRenderer(r.log, botHandle, r.uuidGenerator)
	var body []cards.Node
	if forceTable {
		body = r.renderExperimentalTableCard(msg, interactiveRenderer)
	} else {
		body = interactiveRenderer.RenderAsAdaptiveCard(msg)
	}

	card := cards.New(body, nil).
		WithSchema(cards.DefaultSchema).
		WithVersion(cardVersion)
	err := card.Prepare()
	if err != nil {
		raw, _ := json.MarshalIndent(card, "", "  ")
		r.log.WithError(err).Debugf("Failed to render Teams card: %s", string(raw))

		return nil, Card{}, errx.Wrap(err, "while preparing card")
	}

	content := Card{
		Card: card,
		MsTeams: CardMSTeamsData{
			Width: interactiveRenderer.GetCardPreferredWith(),
		},
	}

	date, cmd, found := ExtractRefreshMessageMetadata(msg)
	if found {
		cmd = strings.Replace(cmd, api.MessageBotNamePlaceholder, botHandle, 1)
		content.Refresh = &CardRefresh{
			Action: CardActionRefresh{
				Verb: cmd,
				Data: cmd,
			},
			Expires: date,
		}
	}

	if interactiveRenderer.wasBotMentionUsed {
		// we can add it only if bot mention was used in a text, otherwise, we will get 400 (Bad Request).
		content.MsTeams.Entities = []CardMSTeamsEntity{
			{
				Type: "mention",
				Text: fmt.Sprintf(mentionHandleFmt, r.botName),
				Mentioned: CardMentioned{
					ID:   r.botID,
					Name: r.botName,
				},
			},
		}
	}

	if r.logCardMessage {
		// only on debug level we play additional effort of marshaling
		raw, _ := json.MarshalIndent(content, "", "  ")
		r.log.Debugf("Rendered Teams card: %s", string(raw))
	}
	return []activity.MsgOption{activity.MsgOptionAttachments([]schema.Attachment{
		{
			ContentType: cardContentType,
			Content:     content,
		},
	})}, content, nil
}

// messageRenderer provides functionality to render MS Teams specific messages from a generic models.
type messageRenderer struct {
	log           logrus.FieldLogger
	botHandle     string
	uuidGenerator func() string

	maxLineSize       int
	wasBotMentionUsed bool
}

func newMessageRenderer(log logrus.FieldLogger, botHandle string, uuidGenerator func() string) *messageRenderer {
	return &messageRenderer{log: log, botHandle: botHandle, uuidGenerator: uuidGenerator}
}

func (r *messageRenderer) storeTextMeta(in string) {
	// was bot mention used
	if !r.wasBotMentionUsed {
		r.wasBotMentionUsed = strings.Contains(in, r.botHandle)
	}

	// max line size
	inLen := width(in)
	if r.maxLineSize < inLen {
		r.maxLineSize = inLen
	}
}

func (r *messageRenderer) lineWithCodeBlock(markdown string, asBold bool) cards.Node {
	markdown = replaceEmojiTagsWithActualOne(markdown)
	r.storeTextMeta(markdown)

	weight := ""
	if asBold {
		weight = "bolder"
	}
	var richBlock cards.RichTextBlock
	appendAsText := func(text string) {
		richBlock.Inlines = append(richBlock.Inlines, &cards.TextRun{
			Text:   text,
			Weight: weight,
		})
	}

	appendAsMono := func(codeText string) {
		richBlock.Inlines = append(richBlock.Inlines, &cards.TextRun{
			Text:     codeText,
			FontType: "monospace",
		})
	}

	if !strings.Contains(markdown, "`") {
		if asBold {
			appendAsText(markdown)
			return &richBlock
		}

		// we don't need to use regex
		return &cards.TextBlock{
			Text: markdown,
			Wrap: ptr.FromType(true),
		}
	}

	// Find all matches for code blocks
	lastIndex := 0
	matches := codeBlockRegex.FindAllStringSubmatchIndex(markdown, -1)
	for _, match := range matches {
		text := markdown[lastIndex:match[0]]
		if len(text) > 0 {
			appendAsText(text)
		}

		codeText := markdown[match[2]:match[3]]
		appendAsMono(codeText)

		lastIndex = match[1]
	}

	// Process remaining text
	remainingText := markdown[lastIndex:]
	if len(remainingText) > 0 {
		appendAsText(remainingText)
	}

	return &richBlock
}

// RenderAsAdaptiveCard returns the AdaptiveCard message for a given input message.
func (r *messageRenderer) RenderAsAdaptiveCard(msg interactive.CoreMessage) []cards.Node {
	var blocks []cards.Node
	if msg.Header != "" {
		blocks = append(blocks, r.lineWithCodeBlock(msg.Header, true))
	}

	if msg.Description != "" {
		blocks = append(blocks, r.lineWithCodeBlock(msg.Description, false))
	}

	if msg.BaseBody.Plaintext != "" {
		blocks = append(blocks, r.mdTextSection(msg.BaseBody.Plaintext))
	}

	if msg.BaseBody.CodeBlock != "" {
		blocks = append(blocks, r.mdCodeBlockSection(msg.BaseBody.CodeBlock))
	}

	all := len(msg.Sections)
	for idx, s := range msg.Sections {
		// [1, N)
		isNotLast := idx > 0 && idx < all
		section := r.renderSection(s, isNotLast)
		blocks = append(blocks, section...)
	}

	for _, i := range msg.PlaintextInputs {
		blocks = append(blocks, r.renderInput(i))
	}

	if !msg.Timestamp.IsZero() {
		blocks = append(blocks, r.renderTimestamp(msg.Timestamp))
	}

	return blocks
}

func (r *messageRenderer) renderInput(s api.LabelInput) cards.Node {
	id := r.uuidGenerator()
	return &cards.InputText{
		Label:       s.Text,
		ID:          id,
		Placeholder: s.Placeholder,
		InlineAction: &cards.ActionSubmit{
			Data: map[string]interface{}{
				inputTextCmdKey:  s.Command,
				inputTextKeyName: id,
				originKeyName:    command.PlainTextInputOrigin,
			},
			IconURL: "https://adaptivecards.io/content/send.png",
		},
	}
}

func (r *messageRenderer) renderSection(in api.Section, isNotLastOne bool) []cards.Node {
	var out []cards.Node
	if in.Header != "" {
		out = append(out, r.lineWithCodeBlock(in.Header, true))
	}

	if in.Description != "" {
		out = append(out, r.lineWithCodeBlock(in.Description, false))
	}

	if len(in.TextFields) > 0 {
		out = append(out, r.renderTextFields(in.TextFields))
	}

	if in.Body.Plaintext != "" {
		out = append(out, r.mdTextSection(in.Body.Plaintext))
	}

	if in.Body.CodeBlock != "" {
		out = append(out, r.mdCodeBlockSection(in.Body.CodeBlock))
	}

	for _, item := range in.PlaintextInputs {
		out = append(out, r.renderInput(item))
	}

	if in.BulletLists.AreItemsDefined() {
		out = append(out, r.renderBulletLists(in.BulletLists)...)
	}

	var (
		inlineBtns, standaloneBtns       = r.renderButtons(in.Buttons)
		inlineSelects, standaloneSelects = r.renderSelects(in.Selects, len(in.Buttons) <= 0)
	)

	// standalone
	out = r.appendIfNotNil(out, standaloneBtns...)
	out = r.appendIfNotNil(out, standaloneSelects)

	// inlined
	var acts cards.ActionSet
	acts.Actions = r.appendIfNotNil(acts.Actions, inlineSelects.Actions...)
	acts.Actions = r.appendIfNotNil(acts.Actions, inlineBtns.Actions...)
	if len(acts.Actions) > 0 {
		out = append(out, &acts)
	}

	if in.MultiSelect.AreOptionsDefined() {
		sec := r.renderMultiselectWithDescription(in.MultiSelect)
		out = append(out, sec...)
	}

	if len(in.Context) > 0 {
		out = append(out, r.renderContext(in.Context)...)
	}

	if !isNotLastOne || len(out) == 0 {
		return out
	}

	return r.enableSeparator(out)
}

func (r *messageRenderer) renderSelects(in api.Selects, unwindFirstButton bool) (*cards.ActionSet, *cards.ColumnSet) {
	switch len(in.Items) {
	case 0:
		return &cards.ActionSet{}, nil
	case 1:
		return r.renderSelectAsOverflowMenu(in.Items[0], unwindFirstButton), nil
	default:
		return &cards.ActionSet{}, r.renderSelectsAsOverflowMenu(in)
	}
}

func (r *messageRenderer) appendIfNotNil(slice []cards.Node, elems ...cards.Node) []cards.Node {
	for _, elem := range elems {
		if isNil(elem) {
			continue
		}
		slice = append(slice, elem)
	}
	return slice
}

func isNil(i any) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

func (r *messageRenderer) enableSeparator(out []cards.Node) []cards.Node {
	obj := out[0]
	switch item := obj.(type) {
	case *cards.TextBlock:
		item.Separator = ptr.FromType(true)
		out[0] = item
	case *cards.ActionSet:
		item.Separator = ptr.FromType(true)
		out[0] = item
	case *cards.FactSet:
		item.Separator = ptr.FromType(true)
		out[0] = item
	case *cards.RichTextBlock:
		item.Separator = ptr.FromType(true)
		out[0] = item
	case *cards.ColumnSet:
		item.Separator = ptr.FromType(true)
		out[0] = item
	case *cards.InputText:
		item.Separator = ptr.FromType(true)
		out[0] = item
	case *cards.InputChoiceSet:
		item.Separator = ptr.FromType(true)
		out[0] = item
	default:
		r.log.Debugf("cannot append separator to unknown type %T", item)
	}
	return out
}

func (r *messageRenderer) renderMultiselectWithDescription(in api.MultiSelect) []cards.Node {
	desc := in.Description.Plaintext
	if in.Description.CodeBlock != "" {
		desc = in.Description.CodeBlock
	}
	var initValues []string
	for _, opt := range in.InitialOptions {
		initValues = append(initValues, opt.Value)
	}

	var options []*cards.InputChoice
	for _, opt := range in.Options {
		options = append(options, &cards.InputChoice{
			Title: opt.Name,
			Value: opt.Value,
		})
	}

	id := r.uuidGenerator()
	return []cards.Node{
		&cards.InputChoiceSet{
			ID:            id,
			Choices:       options,
			IsMultiSelect: ptr.FromType(true),
			Placeholder:   in.Name,
			Value:         strings.Join(initValues, ","),
			Wrap:          ptr.FromType(true),
			Label:         desc,
		},
		&cards.ActionSet{
			Actions: []cards.Node{
				&cards.ActionSubmit{
					Title:            "Submit",
					AssociatedInputs: id,
					Data: map[string]any{
						// Following the Slack approach, we maintain a space between arguments.
						// https://github.com/kubeshop/botkube/blob/9f08ea6be9abc5062e43119f1493eade3a43b6aa/pkg/bot/slack_socket.go#L669-L675
						inputTextCmdKey:  fmt.Sprintf("%s ", in.Command),
						inputTextKeyName: id,
						originKeyName:    command.MultiSelectValueChangeOrigin,
					},
				},
			},
		},
	}
}

// renderButtons renders button section.
//
//  1. With description, renders one per row. For example:
//     `@Botkube get pods` [Button "Get Pods"]
//     `@Botkube get deploys` [Button "Get Deployments"]
//
//  2. Without description: all in the same row. For example:
//     [Button "Get Pods"] [Button "Get Deployments"]
func (r *messageRenderer) renderButtons(in api.Buttons) (*cards.ActionSet, []cards.Node) {
	if in.AtLeastOneButtonHasDescription() {
		return &cards.ActionSet{}, r.renderButtonsWithDescription(in)
	}

	var btns cards.ActionSet
	for _, btn := range in {
		btns.Actions = append(btns.Actions, r.renderButton(btn))
	}

	return &btns, nil
}

func (r *messageRenderer) renderButtonsWithDescription(in api.Buttons) []cards.Node {
	var out []cards.Node
	for _, btn := range in {
		desc := cards.TextBlock{
			Text: btn.Description,
			Wrap: ptr.FromType(true),
		}
		switch btn.DescriptionStyle {
		case api.ButtonDescriptionStyleBold:
			desc.Weight = "bolder"
		case api.ButtonDescriptionStyleCode:
			desc.FontType = "monospace"
		default:
			desc.FontType = "monospace"
		}

		out = append(out, &cards.ColumnSet{
			Columns: []*cards.Column{
				{
					Items:                    []cards.Node{&desc},
					Width:                    "stretch",
					VerticalContentAlignment: "center",
				},
				{
					Items: []cards.Node{
						&cards.ActionSet{
							Actions: []cards.Node{r.renderButton(btn)},
						},
					},
					VerticalContentAlignment: "center",
					Width:                    "stretch",
				},
			},
		})

		r.storeTextMeta(desc.Text + btn.Name)
	}
	return out
}

func (r *messageRenderer) renderButton(btn api.Button) cards.Node {
	if btn.URL != "" {
		return &cards.ActionOpenURL{
			URL:   btn.URL,
			Title: btn.Name,
			Style: r.convertToStyle(btn.Style),
		}
	}
	return &cards.ActionExecute{
		Title: btn.Name,
		Verb:  btn.Command,
		Data: map[string]any{
			originKeyName: command.ButtonClickOrigin,
		},
		Style: r.convertToStyle(btn.Style),
	}
}

// The problem with the ChoiceSet is that they require a dedicated button Action.Submit, to submit selected choices.
func (r *messageRenderer) renderSelectsAsOverflowMenu(s api.Selects) *cards.ColumnSet {
	var out cards.ColumnSet
	for _, s := range s.Items {
		out.Columns = append(out.Columns, &cards.Column{
			VerticalContentAlignment: "center",
			Items: []cards.Node{
				r.mdTextSection(s.Name),
				r.renderSelectAsOverflowMenu(s, true),
			},
			Width: "stretch",
		})
	}

	return &out
}

func (r *messageRenderer) renderSelectAsOverflowMenu(s api.Select, unwindFirstButton bool) *cards.ActionSet {
	var acts cards.ActionSet
	for _, group := range s.OptionGroups {
		for optIdx, opt := range group.Options {
			mode := "secondary"
			if optIdx == 0 && unwindFirstButton {
				// when we only have selects, the first button can be visible
				mode = "primary"
			}
			acts.Actions = append(acts.Actions, &cards.ActionExecute{
				Data:  fmt.Sprintf("%s %s", s.Command, opt.Value),
				Title: opt.Name,
				Mode:  mode,
			})
		}
	}
	return &acts
}

func (r *messageRenderer) mdCodeBlockSection(in string, args ...any) *cards.TextBlock {
	text := r.mdTextSection(in, args...)
	text.Text = strings.ReplaceAll(text.Text, "\n", "\n\n")
	text.FontType = "monospace"
	text.Weight = "lighter"
	text.Wrap = ptr.FromType(true)
	text.IsSubtle = ptr.FromType(true)
	return text
}
func (r *messageRenderer) mdTextSection(in string, args ...any) *cards.TextBlock {
	text := replaceEmojiTagsWithActualOne(fmt.Sprintf(in, args...))
	r.storeTextMeta(text)

	return &cards.TextBlock{
		Wrap: ptr.FromType(true),
		Text: text,
	}
}

func (r *MessageRendererAdapter) renderAsSimpleTextSection(msg interactive.CoreMessage) activity.MsgOption {
	var out strings.Builder
	if msg.Header != "" {
		out.WriteString(replaceEmojiTagsWithActualOne(msg.Header) + "\n")
	}
	if msg.Description != "" {
		out.WriteString(replaceEmojiTagsWithActualOne(msg.Description) + "\n")
	}

	if msg.BaseBody.Plaintext != "" {
		out.WriteString(replaceEmojiTagsWithActualOne(msg.BaseBody.Plaintext))
	}

	if msg.BaseBody.CodeBlock != "" {
		// we don't use the AdaptiveCodeBlock as we want to have a code block even for single lines
		// to make it more readable in the wide view.
		out.WriteString(formatx.CodeBlock(msg.BaseBody.CodeBlock))
	}

	return activity.MsgOptionText(out.String())
}

func (r *messageRenderer) renderTextFields(item api.TextFields) cards.Node {
	var facts []*cards.Fact
	for _, field := range item {
		if field.IsEmpty() {
			continue
		}
		facts = append(facts, &cards.Fact{
			Title: replaceEmojiTagsWithActualOne(field.Key),
			Value: replaceEmojiTagsWithActualOne(field.Value),
		})
	}
	if len(facts) == 0 {
		return nil
	}
	return &cards.FactSet{
		Facts: facts,
	}
}

func (r *messageRenderer) renderBulletLists(in api.BulletLists) []cards.Node {
	var out []cards.Node
	for _, list := range in {
		out = append(out, r.renderSingleBulletList(list)...)
	}
	return out
}

func (r *messageRenderer) renderSingleBulletList(item api.BulletList) []cards.Node {
	var out []cards.Node
	item.Title = strings.TrimSpace(item.Title)
	if item.Title != "" {
		out = append(out, &cards.TextBlock{
			Text: fmt.Sprintf("**%s**", replaceEmojiTagsWithActualOne(item.Title)),
			Wrap: ptr.FromType(true),
		})
	}

	if len(item.Items) > 0 {
		out = append(out, &cards.TextBlock{
			Text: r.bulletList(item.Items),
			Wrap: ptr.FromType(true),
		})
	}
	return out
}

// https://learn.microsoft.com/en-us/adaptive-cards/authoring-cards/text-features#datetime-function-rules
func (r *messageRenderer) renderTimestamp(in time.Time) cards.Node {
	if in.IsZero() {
		return nil
	}

	return &cards.TextBlock{
		Text: ConvertToTeamsTimeRepresentation(in),
	}
}

// https://learn.microsoft.com/en-us/adaptive-cards/authoring-cards/text-features#markdown-commonmark-subset
func (r *messageRenderer) bulletList(msgs []string) string {
	for idx, item := range msgs {
		// We need to change the new line encoding, otherwise it will be printed in a single line. Example use-case:
		//
		// spec.template.spec.containers[*].image:
		//  -: ghcr.io/kubeshop/botkube:v9.99.9-dev
		//  +: ghcr.io/kubeshop/botkube:v1.0.0
		msgs[idx] = strings.ReplaceAll(item, "\n", "\n\n\t")
		r.storeTextMeta(msgs[idx])
	}

	return replaceEmojiTagsWithActualOne(fmt.Sprintf("- %s", strings.Join(msgs, "\r- ")))
}

func (r *messageRenderer) renderContext(in []api.ContextItem) []cards.Node {
	var blocks []cards.Node

	for _, item := range in {
		if item.Text == "" {
			// Skip empty sections
			continue
		}

		text := r.mdTextSection("_%s_", item.Text)
		text.Weight = "lighter"
		text.IsSubtle = ptr.FromType(true)
		blocks = append(blocks, text)
	}

	return blocks
}

func msNewLineFormatter(msg string) string {
	// e.g. `:rocket:` is not supported by MS Teams, so we need to replace it with actual emoji
	msg = replaceEmojiTagsWithActualOne(msg)
	return fmt.Sprintf("%s\n\n", msg)
}

// replaceEmojiTagsWithActualOne replaces the emoji tag with actual emoji.
func replaceEmojiTagsWithActualOne(content string) string {
	return mdEmojiTag.ReplaceAllStringFunc(content, func(s string) string {
		emoji, found := emojiMapping[s]
		if !found {
			return s
		}
		return emoji
	})
}

// emojiMapping holds mapping between emoji tags and actual ones.
var emojiMapping = map[string]string{
	":rocket:":                  "🚀",
	":warning:":                 "⚠️",
	":white_check_mark:":        "✅",
	":arrows_counterclockwise:": "🔄",
	":exclamation:":             "❗",
	":cricket:":                 "🦗",
	":no_entry_sign:":           "🚫",
	":large_green_circle:":      "🟢",
	":new:":                     "🆕",
	":bulb:":                    "💡",
	":crossed_swords:":          "⚔️",
	":tada:":                    "🎉",
}

func (*messageRenderer) convertToStyle(in api.ButtonStyle) string {
	switch in {
	case api.ButtonStyleDefault:
		return ""
	case api.ButtonStylePrimary:
		// Action is displayed with a positive style (typically the button becomes accent color)
		return "positive"
	case api.ButtonStyleDanger:
		// Action is displayed with a destructive style (typically the button becomes red)
		return "destructive"
	}
	return ""
}

func (r *messageRenderer) GetCardPreferredWith() string {
	r.log.WithField("maxLineSize", r.maxLineSize).Debug("Card max line width")
	if r.maxLineSize > smallCardMaxSize {
		return "full"
	}
	return ""
}

// width returns the cell width of characters in the string. ANSI sequences are
// ignored and characters wider than one cell (such as Chinese characters and
// emojis) are appropriately measured.
//
// You should use this instead of len(string) len([]rune(string) as neither
// will give you accurate results.
// copied from: https://github.com/charmbracelet/lipgloss/blob/49671292f7b87676e1854232eac64b3b454434a2/size.go#L15
func width(str string) (width int) {
	for _, l := range strings.Split(str, "\n") {
		w := ansi.PrintableRuneWidth(l)
		if w > width {
			width = w
		}
	}

	return width
}

// ConvertToTeamsTimeRepresentation converts time to Teams time representation.
func ConvertToTeamsTimeRepresentation(in time.Time) string {
	timestamp := in.UTC().Format("2006-01-02T15:04:05Z")
	return fmt.Sprintf("_{{DATE(%s, SHORT)}} at {{TIME(%s)}}_", timestamp, timestamp)
}
