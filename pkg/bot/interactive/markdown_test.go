package interactive

import (
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/golden"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/formatx"
)

// go test -run=TestInteractiveMessageToMarkdownMultiSelect ./pkg/bot/interactive/... -test.update-golden
func TestInteractiveMessageToMarkdownMultiSelect(t *testing.T) {
	// given
	message := CoreMessage{
		Header:      "Adjust notifications",
		Description: "Adjust notifications description",
		Message: api.Message{
			Timestamp: time.Date(2022, 04, 21, 2, 43, 0, 0, time.UTC),
			Sections: []api.Section{
				{
					MultiSelect: api.MultiSelect{
						Name: "Adjust notifications",
						Description: api.Body{
							Plaintext: "Select notification sources",
						},
						Command: "@Botkube edit SourceBindings",
						Options: []api.OptionItem{
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
					TextFields: api.TextFields{
						{
							Key:   "Kind",
							Value: "pod",
						},
						{
							Key:   "Namespace",
							Value: "botkube",
						},
						{
							Key:   "Name",
							Value: "webapp-server-68c5c57f6f",
						},
						{
							Key:   "Reason",
							Value: "BackOff",
						},
					},
				},
				{
					BulletLists: api.BulletLists{
						{
							Title: "Messages",
							Items: []string{
								"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt",
								"Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium",
								"At vero eos et accusamus et iusto odio dignissimos ducimus qui blanditiis praesentium",
							},
						},
						{
							Title: "Issues",
							Items: []string{
								"Issue item 1",
								"Issue item 2",
								"Issue item 3",
							},
						},
					},
				},
				{
					Selects: api.Selects{
						Items: []api.Select{
							{
								Name:    "Commands",
								Command: "@Botkube kcc",
								OptionGroups: []api.OptionGroup{
									{
										Name: "Workloads",
										Options: []api.OptionItem{
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
										Options: []api.OptionItem{
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
		NewlineFormatter: func(msg string) string {
			return fmt.Sprintf("%s<br>", msg)
		},
		HeaderFormatter:            MdHeaderFormatter,
		CodeBlockFormatter:         formatx.CodeBlock,
		AdaptiveCodeBlockFormatter: formatx.AdaptiveCodeBlock,
	}

	formatterForCustomHeaders := MDFormatter{
		NewlineFormatter: NewlineFormatter,
		HeaderFormatter: func(msg string) string {
			return fmt.Sprintf("*%s*", msg)
		},
		CodeBlockFormatter:         formatx.CodeBlock,
		AdaptiveCodeBlockFormatter: formatx.AdaptiveCodeBlock,
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
			given := NewHelpMessage("platform", "testing", []string{"botkube/kubectl"}).Build()
			given.ReplaceBotNamePlaceholder("@Botkube")

			// when
			out := RenderMessage(tc.mdFormatter, given)

			// then
			golden.Assert(t, out, fmt.Sprintf("%s.golden.txt", t.Name()))
		})
	}
}
