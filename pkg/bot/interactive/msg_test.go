package interactive

import (
	"fmt"
	"testing"

	"gotest.tools/v3/golden"
)

// go test -run=TestInteractiveMessageToMarkdown ./pkg/bot/interactive/... -test.update-golden
func TestInteractiveMessageToMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		lineFmt func(msg string) string
	}{
		{
			name:    "render with MS Teams new lines",
			lineFmt: MSTeamsLineFmt,
		},
		{
			name:    "render with Markdown new lines",
			lineFmt: MDLineFmt,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			given := Help("platform", "testing", "@BotKube")

			// when
			out := MessageToMarkdown(tc.lineFmt, given)

			// then
			golden.Assert(t, out, fmt.Sprintf("%s.golden.txt", t.Name()))
		})
	}
}
