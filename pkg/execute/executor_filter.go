package execute

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	//filterFlagRegex regular expression used for extracting executor filters
	filterFlagRegex = regexp.MustCompile(`--filter[=|(' ')]('.*?'|".*?"|\S+)`)
)

// executorFilter interface to implement to filter executor text based results
type executorFilter interface {
	Apply(string) string
	FilteredCommand() string
	IsActive() bool
}

// executorEchoFilter echos given text when asked to filter executor text results.
// Mainly used when executor commands are missing a "--filter=xxx" flag.
type executorEchoFilter struct {
	command string
}

// FilteredCommand returns the command whose results the filter will be applied on.
func (f *executorEchoFilter) FilteredCommand() string {
	return f.command
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
	return &executorEchoFilter{
		command: command,
	}
}

// executorTextFilter filters executor text results by a given text value.
type executorTextFilter struct {
	value   []byte
	command string
}

// FilteredCommand returns the command whose results the filter will be applied on.
func (f *executorTextFilter) FilteredCommand() string {
	return f.command
}

// IsActive whether this filter will actually mutate the output or not.
func (f *executorTextFilter) IsActive() bool {
	return true
}

// newExecutorTextFilter creates a new executorTextFilter.
func newExecutorTextFilter(val, command string) *executorTextFilter {
	return &executorTextFilter{
		value:   []byte(val),
		command: command,
	}
}

// Apply implements executorFilter to apply filtering.
func (f *executorTextFilter) Apply(text string) string {
	var out strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		scanned := scanner.Bytes()
		if bytes.Contains(scanned, f.value) {
			out.Write(bytes.TrimSpace(scanned))
			out.WriteString("\n")
		}
	}

	return strings.TrimSuffix(out.String(), "\n")
}

// extractExecutorFilter extracts an executorFilter based on
// the presence or absence of the "--filter=xxx" flag.
// It also returns passed in executor command minus the
// flag to be executed by downstream executors and if a filter flag was detected.
func extractExecutorFilter(cmd string) executorFilter {
	matchedArray := filterFlagRegex.FindStringSubmatch(cmd)
	if len(matchedArray) < 2 {
		return newExecutorEchoFilter(cmd)
	}

	match, err := strconv.Unquote(matchedArray[1])
	if err != nil {
		match = matchedArray[1]
	}
	return newExecutorTextFilter(match, strings.ReplaceAll(cmd, fmt.Sprintf(" %s", matchedArray[0]), ""))
}
