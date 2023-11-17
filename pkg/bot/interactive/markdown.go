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
	NewlineFormatter           func(msg string) string
	HeaderFormatter            func(msg string) string
	CodeBlockFormatter         func(msg string) string
	AdaptiveCodeBlockFormatter func(msg string) string
}

// NewMDFormatter is for initializing custom Markdown formatter
func NewMDFormatter(newlineFormatter, headerFormatter func(msg string) string) MDFormatter {
	return MDFormatter{
		NewlineFormatter:           newlineFormatter,
		HeaderFormatter:            headerFormatter,
		CodeBlockFormatter:         formatx.CodeBlock,
		AdaptiveCodeBlockFormatter: formatx.AdaptiveCodeBlock,
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
		out.WriteString(mdFormatter.NewlineFormatter(in))
	}

	if msg.Header != "" {
		addLine(mdFormatter.HeaderFormatter(msg.Header))
	}

	if msg.Description != "" {
		addLine(msg.Description)
	}

	if msg.BaseBody.Plaintext != "" {
		addLine(msg.BaseBody.Plaintext)
	}

	if msg.BaseBody.CodeBlock != "" {
		addLine(mdFormatter.CodeBlockFormatter(msg.BaseBody.CodeBlock))
	}

	for i, section := range msg.Sections {
		// do not include empty line when there is no base content
		var empty api.Body
		if i != 0 || msg.BaseBody != empty || msg.Description != "" || msg.Header != "" {
			addLine("") // padding between sections
		}

		if section.Header != "" {
			addLine(mdFormatter.HeaderFormatter(section.Header))
		}

		if len(section.TextFields) > 0 {
			addLine(mdFormatter.HeaderFormatter("Fields"))
			for _, field := range section.TextFields {
				addLine(fmt.Sprintf(" • %s: %s", mdFormatter.HeaderFormatter(field.Key), field.Value))
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
			addLine(mdFormatter.CodeBlockFormatter(section.Body.CodeBlock))
		}

		if section.BulletLists.AreItemsDefined() {
			for _, item := range section.BulletLists {
				addLine("") // new line
				addLine(mdFormatter.HeaderFormatter(item.Title))

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
				addLine(mdFormatter.AdaptiveCodeBlockFormatter(ms.Description.CodeBlock))
			}

			addLine("") // new line
			addLine(mdFormatter.HeaderFormatter("Available options"))

			for _, opt := range ms.Options {
				addLine(fmt.Sprintf(" • %s", mdFormatter.AdaptiveCodeBlockFormatter(opt.Value)))
			}
		}

		if section.Selects.AreOptionsDefined() {
			addLine("") // new line
			addLine(mdFormatter.HeaderFormatter("Available options"))

			for _, item := range section.Selects.Items {
				for _, group := range item.OptionGroups {
					addLine(fmt.Sprintf(" • %s", group.Name))
					for _, opt := range group.Options {
						addLine(fmt.Sprintf("    • %s", mdFormatter.AdaptiveCodeBlockFormatter(opt.Value)))
					}
				}
			}
		}

		for _, btn := range section.Buttons {
			if btn.DescriptionStyle == api.ButtonDescriptionStyleBold && btn.Description != "" {
				addLine(mdFormatter.HeaderFormatter(btn.Description))
			}

			if btn.URL != "" {
				addLine(fmt.Sprintf("%s: %s", btn.Name, btn.URL))
				continue
			}
			if btn.Command != "" {
				addLine(fmt.Sprintf("  • %s", mdFormatter.AdaptiveCodeBlockFormatter(btn.Command)))
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
