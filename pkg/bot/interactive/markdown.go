package interactive

import (
	"fmt"
	"strings"
	"time"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/formatx"
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
func RenderMessage(mdFormatter MDFormatter, msg CoreMessage) string {
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

	if msg.BaseBody.Plaintext != "" {
		addLine(msg.BaseBody.Plaintext)
	}

	if msg.BaseBody.CodeBlock != "" {
		addLine(mdFormatter.codeBlockFormatter(msg.BaseBody.CodeBlock))
	}

	for i, section := range msg.Sections {
		// do not include empty line when there is no base content
		var empty api.Body
		if i != 0 || msg.BaseBody != empty || msg.Description != "" || msg.Header != "" {
			addLine("") // padding between sections
		}

		if section.Header != "" {
			addLine(mdFormatter.headerFormatter(section.Header))
		}

		if len(section.TextFields) > 0 {
			addLine(mdFormatter.headerFormatter("Fields"))
			for _, field := range section.TextFields {
				addLine(fmt.Sprintf(" • %s: %s", mdFormatter.headerFormatter(field.Key), field.Value))
			}
			addLine("") // new line to separate other
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

		if section.BulletLists.AreItemsDefined() {
			for _, item := range section.BulletLists {
				addLine("") // new line
				addLine(mdFormatter.headerFormatter(item.Title))

				for _, opt := range item.Items {
					addLine(fmt.Sprintf(" • %s", opt))
				}
			}
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
			addLine(mdFormatter.headerFormatter("Available options"))

			for _, opt := range ms.Options {
				addLine(fmt.Sprintf(" • %s", mdFormatter.adaptiveCodeBlockFormatter(opt.Value)))
			}
		}

		if section.Selects.AreOptionsDefined() {
			addLine("") // new line
			addLine(mdFormatter.headerFormatter("Available options"))

			for _, item := range section.Selects.Items {
				for _, group := range item.OptionGroups {
					addLine(fmt.Sprintf(" • %s", group.Name))
					for _, opt := range group.Options {
						addLine(fmt.Sprintf("    • %s", mdFormatter.adaptiveCodeBlockFormatter(opt.Value)))
					}
				}
			}
		}

		for _, btn := range section.Buttons {
			if btn.DescriptionStyle == api.ButtonDescriptionStyleBold && btn.Description != "" {
				addLine(mdFormatter.headerFormatter(btn.Description))
			}

			if btn.URL != "" {
				addLine(fmt.Sprintf("%s: %s", btn.Name, btn.URL))
				continue
			}
			if btn.Command != "" {
				addLine(fmt.Sprintf("  • %s", mdFormatter.adaptiveCodeBlockFormatter(btn.Command)))
				continue
			}
		}

		for _, ctxItem := range section.Context {
			addLine(ctxItem.Text)
		}
	}
	if !msg.Timestamp.IsZero() {
		addLine("") // new line
		addLine(msg.Timestamp.Format(time.RFC1123))
	}

	return out.String()
}
