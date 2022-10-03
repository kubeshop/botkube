package interactive

import (
	"fmt"
	"strings"

	formatx "github.com/kubeshop/botkube/pkg/format"
)

// MDFormatter represents the capability of Markdown Formatter
type MDFormatter struct {
	newlineFormatter           func(msg string) string
	headerFormatter            func(msg string) string
	codeBlockFormatter         func(msg string) string
	adaptiveCodeBlockFormatter func(msg string) string
}

// NewMDFormatter is for initializing custom Markdown formatter
func NewMDFormatter(newlineFormatter, headerFormatter func(msg string) string) MDFormatter {
	return MDFormatter{
		newlineFormatter:           newlineFormatter,
		headerFormatter:            headerFormatter,
		codeBlockFormatter:         formatx.CodeBlock,
		adaptiveCodeBlockFormatter: formatx.AdaptiveCodeBlock,
	}
}

// DefaultMDFormatter is for initializing built-in Markdown formatter
func DefaultMDFormatter() MDFormatter {
	return NewMDFormatter(NewlineFormatter, MdHeaderFormatter)
}

// RenderMessage returns interactive message as a plaintext with Markdown syntax.
func RenderMessage(mdFormatter MDFormatter, msg Message) string {
	var out strings.Builder
	addLine := func(in string) {
		out.WriteString(mdFormatter.newlineFormatter(in))
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
		addLine(mdFormatter.codeBlockFormatter(msg.Body.CodeBlock))
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
			addLine(mdFormatter.codeBlockFormatter(section.Body.CodeBlock))
		}

		if section.MultiSelect.AreOptionsDefined() {
			ms := section.MultiSelect

			if ms.Description.Plaintext != "" {
				addLine(ms.Description.Plaintext)
			}

			if ms.Description.CodeBlock != "" {
				addLine(mdFormatter.adaptiveCodeBlockFormatter(ms.Description.CodeBlock))
			}

			addLine("") // new line
			addLine("Available options:")

			for _, opt := range ms.Options {
				addLine(fmt.Sprintf(" - %s", mdFormatter.adaptiveCodeBlockFormatter(opt.Value)))
			}
		}

		for _, btn := range section.Buttons {
			if btn.URL != "" {
				addLine(fmt.Sprintf("%s: %s", btn.Name, btn.URL))
				continue
			}
			if btn.Command != "" {
				addLine(fmt.Sprintf("  - %s", mdFormatter.adaptiveCodeBlockFormatter(btn.Command)))
				continue
			}
		}
	}

	return out.String()
}
