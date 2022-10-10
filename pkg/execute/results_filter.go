package execute

import (
	"bufio"
	"bytes"
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
			out.Write(scanned)
			out.WriteString("\n")
		}
	}

	return strings.TrimSuffix(out.String(), "\n")
}
