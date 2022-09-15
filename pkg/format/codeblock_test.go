package format_test

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/format"
)

func TestCodeBlock(t *testing.T) {
	// given
	in := "\t  hello there\ntesting!  "
	expected := "```\n" + heredoc.Doc(`
		hello there
		testing!
	`) + "```"

	// when
	actual := format.CodeBlock(in)

	// then
	assert.Equal(t, expected, actual)
}

func TestAdaptiveCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{
			name: "Multiline string",
			in:   "\t  hello there\ntesting!  ",
			expected: "```\n" + heredoc.Doc(`
					hello there
					testing!
				`) + "```",
		},
		{
			name:     "Single line string",
			in:       "\t  hello there - testing!  ",
			expected: "`hello there - testing!`",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			actual := format.AdaptiveCodeBlock(tc.in)

			// then
			assert.Equal(t, tc.expected, actual)
		})
	}
}
