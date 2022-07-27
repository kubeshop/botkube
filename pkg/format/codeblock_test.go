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
