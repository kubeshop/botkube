package execute

import (
	"bufio"
	"bytes"
	"strings"
)

// executorFilter interface to implement to filter executor text based results
type executorFilter interface {
	Apply(string) string
	IsActive() bool
}

// executorEchoFilter echos given text when asked to filter executor text results.
// Mainly used when executor commands are missing a "--filter=xxx" flag.
type executorEchoFilter struct {
}

// IsActive whether this filter will actually mutate the output or not.
func (f *executorEchoFilter) IsActive() bool {
	return false
}

// Apply implements executorFilter to apply filtering.
func (f *executorEchoFilter) Apply(text string) string {
	return text
}

// newExecutorEchoFilter creates a new executorEchoFilter.
func newExecutorEchoFilter(command string) *executorEchoFilter {
	return &executorEchoFilter{}
}

// executorTextFilter filters executor text results by a given text value.
type executorTextFilter struct {
	value []byte
}

// IsActive whether this filter will actually mutate the output or not.
func (f *executorTextFilter) IsActive() bool {
	return true
}

// newExecutorTextFilter creates a new executorTextFilter.
func newExecutorTextFilter(val string) *executorTextFilter {
	return &executorTextFilter{
		value: []byte(val),
	}
}

// Apply implements executorFilter to apply filtering.
func (f *executorTextFilter) Apply(text string) string {
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
