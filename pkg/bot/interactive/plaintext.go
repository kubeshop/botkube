package interactive

import (
	"fmt"
	"strings"
)

// MessageToPlaintext returns interactive message as a plaintext.
func MessageToPlaintext(msg Message, newlineFormatter func(in string) string) string {
	var out strings.Builder
	addLine := func(in string) {
		out.WriteString(newlineFormatter(in))
	}

	if msg.Header != "" {
		addLine(msg.Header)
	}

	if msg.Body.Plaintext != "" {
		addLine(msg.Body.Plaintext)
	}

	if msg.Body.CodeBlock != "" {
		addLine(msg.Body.CodeBlock)
	}

	for _, section := range msg.Sections {
		addLine("") // padding between sections

		if section.Header != "" {
			addLine(section.Header)
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
			addLine(section.Body.CodeBlock)
		}

		if section.MultiSelect.AreOptionsDefined() {
			ms := section.MultiSelect

			if ms.Description.Plaintext != "" {
				addLine(ms.Description.Plaintext)
			}

			if ms.Description.CodeBlock != "" {
				addLine(ms.Description.CodeBlock)
			}

			addLine("") // new line
			addLine("Available options:")

			for _, opt := range ms.Options {
				addLine(fmt.Sprintf(" - %s", opt.Value))
			}
		}

		for _, btn := range section.Buttons {
			if btn.URL != "" {
				addLine(fmt.Sprintf("%s: %s", btn.Name, btn.URL))
				continue
			}
			if btn.Command != "" {
				addLine(fmt.Sprintf("  - %s", btn.Command))
				continue
			}
		}
	}

	return out.String()
}
