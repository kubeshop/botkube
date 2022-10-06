package interactive

import (
	"fmt"
	"testing"

	"gotest.tools/v3/golden"

	formatx "github.com/kubeshop/botkube/pkg/format"
)

// go test -run=TestInteractiveMessageToMarkdownMultiSelect ./pkg/bot/interactive/... -test.update-golden
func TestInteractiveMessageToMarkdownMultiSelect(t *testing.T) {
	// given
	message := Message{
		Base: Base{
			Header: "Adjust notifications",
		},

		Sections: []Section{
			{
				MultiSelect: MultiSelect{
					Name: "Adjust notifications",
					Description: Body{
						Plaintext: "Select notification sources",
					},
					Command: "@Botkube edit SourceBindings",
					Options: []OptionItem{
						{
							Name:  "K8s all events",
							Value: "k8s-all-events",
						},
						{
							Name:  "K8s recommendations",
							Value: "k8s-recommendations",
						},
					},
				},
			},
			{
				Selects: Selects{
					Items: []Select{
						{
							Name:    "Commands",
							Command: "@Botkube kcc",
							OptionGroups: []OptionGroup{
								{
									Name: "Workloads",
									Options: []OptionItem{
										{
											Name:  "pods",
											Value: "pods",
										},
										{
											Name:  "deployments",
											Value: "deployments",
										},
									},
								},
								{
									Name: "Data",
									Options: []OptionItem{
										{
											Name:  "configmap",
											Value: "configmap",
										},
										{
											Name:  "secrets",
											Value: "secrets",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// when
	out := RenderMessage(DefaultMDFormatter(), message)

	// then
	golden.Assert(t, out, fmt.Sprintf("%s.golden.txt", t.Name()))
}

// go test -run=TestInteractiveMessageToMarkdown ./pkg/bot/interactive/... -test.update-golden
func TestInteractiveMessageToMarkdown(t *testing.T) {
	formatterForCustomNewLines := MDFormatter{
		newlineFormatter: func(msg string) string {
			return fmt.Sprintf("%s<br>", msg)
		},
		headerFormatter:            MdHeaderFormatter,
		codeBlockFormatter:         formatx.CodeBlock,
		adaptiveCodeBlockFormatter: formatx.AdaptiveCodeBlock,
	}

	formatterForCustomHeaders := MDFormatter{
		newlineFormatter: NewlineFormatter,
		headerFormatter: func(msg string) string {
			return fmt.Sprintf("*%s*", msg)
		},
		codeBlockFormatter:         formatx.CodeBlock,
		adaptiveCodeBlockFormatter: formatx.AdaptiveCodeBlock,
	}
	tests := []struct {
		name        string
		mdFormatter MDFormatter
	}{
		{
			name:        "render with custom new lines and default headers",
			mdFormatter: formatterForCustomNewLines,
		},
		{
			name:        "render with custom headers and default new lines",
			mdFormatter: formatterForCustomHeaders,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			given := Help("platform", "testing", "@Botkube")

			// when
			out := RenderMessage(tc.mdFormatter, given)

			// then
			golden.Assert(t, out, fmt.Sprintf("%s.golden.txt", t.Name()))
		})
	}
}
