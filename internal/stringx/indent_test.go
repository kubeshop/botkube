package stringx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndentAfterLine(t *testing.T) {
	tests := []struct {
		name     string
		Input    string
		Indent   string
		After    int
		Expected string
	}{
		{
			name: "No-op, should pass through",

			Input:    "foobar",
			Indent:   "\t",
			After:    1,
			Expected: "foobar",
		},
		{
			name: "Basic indentation",

			Input:    "foobar",
			Indent:   "\t",
			After:    0,
			Expected: "\tfoobar",
		},
		{
			name: "Multi-line indentation",

			Input:    "foo\nbar\nbaz",
			Indent:   "\t",
			After:    1,
			Expected: "foo\n\tbar\n\tbaz",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotOut := IndentAfterLine(
				tc.Input,
				tc.After,
				tc.Indent,
			)

			// then
			assert.Equal(t, tc.Expected, gotOut)
		})
	}
}
