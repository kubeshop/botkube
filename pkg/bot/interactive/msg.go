package interactive

import (
	"fmt"
	"regexp"
	"strings"

	formatx "github.com/kubeshop/botkube/pkg/format"
)

// mdEmojiTag finds the emoji tags
var mdEmojiTag = regexp.MustCompile(`:(\w+):`)

// MSTeamsLineFmt represents new line formatting for MS Teams.
// Unfortunately, it's different from all others integrations.
func MSTeamsLineFmt(msg string) string {
	// e.g. `:rocket:` is not supported by MS Teams, so we need to replace it with actual emoji
	msg = replaceEmoijTagsWithActualOne(msg)
	return fmt.Sprintf("%s<br>", msg)
}

// MDLineFmt represents a Markdown new line formatting.
func MDLineFmt(msg string) string {
	return fmt.Sprintf("%s\n", msg)
}

// MessageToMarkdown returns interactive message as a plaintext with Markdown syntax.
func MessageToMarkdown(lineFmt func(msg string) string, msg Message) string {
	var out strings.Builder
	addLine := func(in string) {
		out.WriteString(lineFmt(in))
	}

	if msg.Header != "" {
		addLine(mdHeader(msg.Header))
	}
	if msg.Description != "" {
		addLine(msg.Description)
	}

	if msg.Body.Plaintext != "" {
		addLine(msg.Body.Plaintext)
	}

	if msg.Body.CodeBlock != "" {
		addLine(formatx.CodeBlock(msg.Body.CodeBlock))
	}

	for _, section := range msg.Sections {
		addLine("") // padding between sections

		if section.Header != "" {
			addLine(mdHeader(section.Header))
		}
		if section.Description != "" {
			addLine(section.Description)
		}

		if section.Body.Plaintext != "" {
			addLine(section.Body.Plaintext)
		}

		if section.Body.CodeBlock != "" {
			// not using the adaptive code block is on purpose, we always want to have
			// a multiline code block to improve readability
			addLine(formatx.CodeBlock(section.Body.CodeBlock))
		}

		for _, btn := range section.Buttons {
			if btn.URL != "" {
				addLine(fmt.Sprintf("%s: %s", btn.Name, btn.URL))
				continue
			}
			if btn.Command != "" {
				addLine(fmt.Sprintf("  - %s", formatx.AdaptiveCodeBlock(btn.Command)))
				continue
			}
		}
	}

	return out.String()
}

// emojiMapping holds mapping between emoji tags and actual ones.
var emojiMapping = map[string]string{
	":rocket:": "🚀",
}

// replaceEmoijTagsWithActualOne replaces the emoji tag with actual emoji.
func replaceEmoijTagsWithActualOne(content string) string {
	return mdEmojiTag.ReplaceAllStringFunc(content, func(s string) string {
		return emojiMapping[s]
	})
}

func mdHeader(msg string) string {
	return fmt.Sprintf("*%s*", msg)
}
