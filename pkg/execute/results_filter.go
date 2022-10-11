package execute

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ResultsFilter interface {
	Apply(string) string
}

type EchoFilter struct{}

func (f *EchoFilter) Apply(text string) string {
	return text
}

func NewEchoFilter() *EchoFilter {
	return &EchoFilter{}
}

type TextFilter struct {
	value []byte
}

func NewTextFilter(val string) *TextFilter {
	return &TextFilter{
		value: []byte(val),
	}
}

func (f *TextFilter) Apply(text string) string {
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

func extractResultsFilter(cmd string) (ResultsFilter, string) {
	r, _ := regexp.Compile(`--filter[=|(' ')]('.*?'|".*?"|\S+)`)

	var filter ResultsFilter
	var cmdMinusFilter string

	matchedArray := r.FindStringSubmatch(cmd)
	if len(matchedArray) >= 2 {
		match, err := strconv.Unquote(matchedArray[1])
		if err != nil {
			match = matchedArray[1]
		}
		filter = NewTextFilter(match)
		cmdMinusFilter = strings.ReplaceAll(cmd, fmt.Sprintf(" %s", matchedArray[0]), "")
	} else {
		filter = NewEchoFilter()
		cmdMinusFilter = cmd
	}

	return filter, cmdMinusFilter
}
