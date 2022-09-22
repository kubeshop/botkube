package interactive

import (
	"fmt"
	"testing"

	"gotest.tools/v3/golden"
)

// go test -run=TestInteractiveMessageToMarkdown ./pkg/bot/interactive/... -test.update-golden
func TestInteractiveMessageToMarkdown(t *testing.T) {
	formatterForCustomNewLines := MDFormatter{
		lineFormatter: func(msg string) string {
			return fmt.Sprintf("%s<br>", msg)
		},
		headerFormatter: DefaultMDHeaderFormatter,
	}

	formatterForCustomHeaders := MDFormatter{
		lineFormatter: DefaultMDLineFormatter,
		headerFormatter: func(msg string) string {
			return fmt.Sprintf("*%s*", msg)
		},
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
			given := Help("platform", "testing", "@BotKube")

			// when
			out := MessageToMarkdown(tc.mdFormatter, given)

			// then
			golden.Assert(t, out, fmt.Sprintf("%s.golden.txt", t.Name()))
		})
	}
}
