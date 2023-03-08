package formatx_test

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/formatx"
)

func TestCodeBlock(t *testing.T) {
	// given
	in := "\t  hello there\ntesting!  "
	expected := "```\n" + heredoc.Doc(`
		hello there
		testing!
	`) + "```"

	// when
	actual := formatx.CodeBlock(in)

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
			actual := formatx.AdaptiveCodeBlock(tc.in)

			// then
			assert.Equal(t, tc.expected, actual)
		})
	}
}
