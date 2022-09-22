package interactive

import (
	"fmt"
	"strings"

	formatx "github.com/kubeshop/botkube/pkg/format"
)

// DefaultMDLineFormatter represents a Markdown new line formatting.
func DefaultMDLineFormatter(msg string) string {
	return fmt.Sprintf("%s\n", msg)
}

// DefaultMDHeaderFormatter represents a Markdown header formatting.
func DefaultMDHeaderFormatter(msg string) string {
	return fmt.Sprintf("**%s**", msg)
}

// MDFormatter represents the capability of Markdown Formatter
type MDFormatter struct {
	lineFormatter   func(msg string) string
	headerFormatter func(msg string) string
}

// NewMDFormatter is for initializing custom Markdown formatter
func NewMDFormatter(lineFormatter, headerFormatter func(msg string) string) MDFormatter {
	return MDFormatter{
		lineFormatter:   lineFormatter,
		headerFormatter: headerFormatter,
	}
}

// DefaultMDFormatter is for initializing built-in Markdown formatter
func DefaultMDFormatter() MDFormatter {
	return NewMDFormatter(DefaultMDLineFormatter, DefaultMDHeaderFormatter)
}

// MessageToMarkdown returns interactive message as a plaintext with Markdown syntax.
func MessageToMarkdown(mdFormatter MDFormatter, msg Message) string {
	var out strings.Builder
	addLine := func(in string) {
		out.WriteString(mdFormatter.lineFormatter(in))
	}

	if msg.Header != "" {
		addLine(mdFormatter.headerFormatter(msg.Header))
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
			addLine(mdFormatter.headerFormatter(section.Header))
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

		if section.MultiSelect.AreOptionsDefined() {
			ms := section.MultiSelect

			if ms.Description.Plaintext != "" {
				addLine(ms.Description.Plaintext)
			}

			if ms.Description.CodeBlock != "" {
				addLine(formatx.AdaptiveCodeBlock(ms.Description.CodeBlock))
			}

			addLine("") // new line
			addLine("Available options:")

			for _, opt := range ms.Options {
				addLine(fmt.Sprintf(" - %s", formatx.AdaptiveCodeBlock(opt.Value)))
			}
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
