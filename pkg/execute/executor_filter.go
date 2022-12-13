package execute

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/pflag"
)

const (
	incorrectFilterFlag     = "incorrect use of --filter flag: %s"
	filterFlagParseErrorMsg = `incorrect use of --filter flag: could not parse flag in %s.

error: %s
Use --filter="value" or --filter value`
	missingCmdFilterValue = `incorrect use of --filter flag: an argument is missing. use --filter="value" or --filter value`
	multipleFilters       = "incorrect use of --filter flag: found more than one filter flag."
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
			out.Write(scanned)
			out.WriteString("\n")
		}
	}

	return strings.TrimSuffix(out.String(), "\n")
}

// extractExecutorFilter extracts an executorFilter based on
// the presence or absence of the "--filter=xxx" flag.
// It also returns passed in executor command minus the
// flag to be executed by downstream executors and if a filter flag was detected.
// ignore unknown flags errors, e.g. `--cluster-name` etc.
func extractExecutorFilter(cmd string) (executorFilter, error) {
	var filters []string

	filters, err := parseAndValidateAnyFilters(cmd)
	if err != nil {
		return nil, err
	}

	if len(filters) == 0 {
		return newExecutorEchoFilter(cmd), nil
	}

	if len(filters[0]) == 0 {
		return nil, errors.New(missingCmdFilterValue)
	}

	filterVal := filters[0]
	escapedFilterVal := regexp.QuoteMeta(filterVal)
	filterFlagRegex, err := regexp.Compile(fmt.Sprintf(`--filter[=|(' ')]*('%s'|"%s"|%s)("|')*`,
		escapedFilterVal,
		escapedFilterVal,
		escapedFilterVal))
	if err != nil {
		return nil, errors.New("could not extract provided filter")
	}

	matches := filterFlagRegex.FindStringSubmatch(cmd)
	if len(matches) == 0 {
		return nil, fmt.Errorf(filterFlagParseErrorMsg, cmd, "it contains unsupported characters.")
	}
	return newExecutorTextFilter(filterVal, strings.ReplaceAll(cmd, fmt.Sprintf(" %s", matches[0]), "")), nil
}

// parseAndValidateAnyFilters parses any filter flags returning their values or an error.
func parseAndValidateAnyFilters(cmd string) ([]string, error) {
	var out []string

	args, err := shellwords.Parse(cmd)
	if err != nil {
		return nil, fmt.Errorf(filterFlagParseErrorMsg, cmd, err.Error())
	}

	f := pflag.NewFlagSet("extract-filters", pflag.ContinueOnError)
	f.ParseErrorsWhitelist.UnknownFlags = true

	f.StringArrayVar(&out, "filter", []string{}, "Output filter")
	if err := f.Parse(args); err != nil {
		return nil, fmt.Errorf(incorrectFilterFlag, err)
	}

	if len(out) > 1 {
		return nil, errors.New(multipleFilters)
	}

	if len(out) == 1 && (strings.HasPrefix(out[0], "-")) {
		return nil, errors.New(missingCmdFilterValue)
	}

	return out, nil
}
