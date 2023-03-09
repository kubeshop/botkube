package bot

import (
	"fmt"
	"strings"
	"time"

	cards "github.com/DanielTitkov/go-adaptive-cards"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

// TeamsRenderer provides functionality to render MS Teams specific messages from a generic models.
type TeamsRenderer struct {
	mdFormatter interactive.MDFormatter
}

// NewTeamsRenderer return a new TeamsRenderer instance.
func NewTeamsRenderer() *TeamsRenderer {
	return &TeamsRenderer{
		mdFormatter: interactive.NewMDFormatter(msNewLineFormatter, interactive.MdHeaderFormatter),
	}
}

// MessageToMarkdown renders message in Markdown format.
func (r *TeamsRenderer) MessageToMarkdown(in interactive.CoreMessage) string {
	return interactive.RenderMessage(r.mdFormatter, in)
}

// NonInteractiveSectionToCard returns AdaptiveCard for the given message.
// Note: It cannot be used for other messages as we take into account only first message section with limited primitives:
// - TextFields
// - BulletLists
// - Timestamp
// It should be removed once we will add support for a proper message renderer.
func (r *TeamsRenderer) NonInteractiveSectionToCard(msg interactive.CoreMessage) (*cards.Card, error) {
	if err := IsValidNonInteractiveSingleSection(msg); err != nil {
		return nil, err
	}

	event := msg.Sections[0]
	var nodes []cards.Node

	if event.Base.Header != "" {
		nodes = append(nodes, &cards.TextBlock{
			Size: "Large",
			Text: event.Base.Header,
		})
	}

	nodes = r.appendIfNotNil(nodes, r.renderTextFields(event.TextFields))
	nodes = r.appendIfNotNil(nodes, r.renderBulletLists(event.BulletLists)...)
	nodes = r.appendIfNotNil(nodes, r.renderTimestamp(msg.Timestamp))

	card := cards.New(nodes, []cards.Node{}).
		WithSchema(cards.DefaultSchema).
		WithVersion(cards.Version12)

	if err := card.Prepare(); err != nil {
		return nil, fmt.Errorf("while preparing event card message: %w", err)
	}

	return card, nil
}

// the cards.Prepare() method panics on nil items, so we need to filter them out.
func (r *TeamsRenderer) appendIfNotNil(slice []cards.Node, elems ...cards.Node) []cards.Node {
	for _, elem := range elems {
		if elem == nil {
			continue
		}
		slice = append(slice, elem)
	}
	return slice
}

func (r *TeamsRenderer) renderTextFields(item api.TextFields) cards.Node {
	var facts []*cards.Fact
	for _, field := range item {
		if field.IsEmpty() {
			continue
		}
		facts = append(facts, &cards.Fact{
			Title: field.Key,
			Value: field.Value,
		})
	}
	if len(facts) == 0 {
		return nil
	}
	return &cards.FactSet{
		Facts: facts,
	}
}

func (r *TeamsRenderer) renderBulletLists(in api.BulletLists) []cards.Node {
	var out []cards.Node
	for _, list := range in {
		out = append(out, r.renderSingleBulletList(list)...)
	}
	return out
}

func (r *TeamsRenderer) renderSingleBulletList(item api.BulletList) []cards.Node {
	return []cards.Node{
		&cards.TextBlock{
			Text: fmt.Sprintf("**%s**", item.Title),
		},
		&cards.TextBlock{
			Text: r.bulletList(item.Items),
			Wrap: cards.TruePtr(),
		},
	}
}

// https://learn.microsoft.com/en-us/adaptive-cards/authoring-cards/text-features#datetime-function-rules
func (r *TeamsRenderer) renderTimestamp(in time.Time) cards.Node {
	if in.IsZero() {
		return nil
	}
	timestamp := in.UTC().Format("2006-01-02T15:04:05Z")
	return &cards.TextBlock{
		Text: fmt.Sprintf("_{{DATE(%s, SHORT)}} at {{TIME(%s)}}_", timestamp, timestamp),
	}
}

// https://learn.microsoft.com/en-us/adaptive-cards/authoring-cards/text-features#markdown-commonmark-subset
func (r *TeamsRenderer) bulletList(msgs []string) string {
	for idx, item := range msgs {
		// We need to change the new line encoding, otherwise it will be printed in a single line. Example use-case:
		//
		// spec.template.spec.containers[*].image:
		//  -: ghcr.io/kubeshop/botkube:v9.99.9-dev
		//  +: ghcr.io/kubeshop/botkube:v1.0.0
		msgs[idx] = strings.ReplaceAll(item, "\n", "\n\n\t\t")
	}
	return fmt.Sprintf("- %s", strings.Join(msgs, "\r- "))
}

func msNewLineFormatter(msg string) string {
	// e.g. `:rocket:` is not supported by MS Teams, so we need to replace it with actual emoji
	msg = replaceEmojiTagsWithActualOne(msg)
	return fmt.Sprintf("%s\n\n", msg)
}
