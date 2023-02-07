package interactive

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/kubeshop/botkube/pkg/api"
)

// go test -run=TestInteractiveMessageToMarkdownMultiSelect ./pkg/bot/interactive/... -test.update-golden
func TestInteractiveMessageToPlaintextMultiSelect(t *testing.T) {
	// given
	message := Message{
		Base: api.Base{
			Header:      "Adjust notifications",
			Description: "Adjust notifications description",
		},
		Message: api.Message{
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
			},
		},
	}

	// when
	out := MessageToPlaintext(message, NewlineFormatter)

	// then
	golden.Assert(t, out, fmt.Sprintf("%s.golden.txt", t.Name()))
}

// go test -run=TestInteractiveMessageToPlaintext ./pkg/bot/interactive/... -test.update-golden
func TestInteractiveMessageToPlaintext(t *testing.T) {
	customNewlineFormatter := func(msg string) string {
		return fmt.Sprintf("%s\r\n", msg)
	}

	// given
	help := NewHelpMessage("platform", "testing", "@Botkube", nil).Build()

	// when
	out := MessageToPlaintext(help, customNewlineFormatter)

	// then
	assert.Assert(t, golden.String(out, fmt.Sprintf("%s.golden.txt", t.Name())))
}
