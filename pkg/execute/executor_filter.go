package execute

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

var _ executorFilter = &executorTextFilter{}

// executorTextFilter filters executor text results by a given text value.
type executorTextFilter struct {
	value []byte
}

// IsActive whether this filter will actually mutate the output or not.
func (f *executorTextFilter) IsActive() bool {
	return len(f.value) > 0
}

// newExecutorTextFilter creates a new executorTextFilter.
func newExecutorTextFilter(val string) *executorTextFilter {
	return &executorTextFilter{
		value: []byte(val),
	}
}

// Apply implements executorFilter to apply filtering.
func (f *executorTextFilter) Apply(text string) string {
	if !f.IsActive() {
		return text
	}

	var out strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		scanned := scanner.Bytes()
		if bytes.Contains(scanned, f.value) {
			out.Write(scanned)
			out.WriteString("\n")
		}
	}

	return strings.TrimSuffix(out.String(), "\n")
}

func appendInteractiveFilterIfNeeded(body string, msg interactive.CoreMessage, cmdCtx CommandContext) interactive.CoreMessage {
	if !cmdCtx.Platform.IsInteractive() {
		return msg
	}
	if len(strings.SplitN(body, "\n", lineLimitToShowFilter)) < lineLimitToShowFilter {
		return msg
	}

	msg.PlaintextInputs = append(msg.PlaintextInputs, filterInput(cmdCtx.CleanCmd))
	return msg
}
