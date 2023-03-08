package bot

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

func TestSlackNonInteractiveSectionMessage(t *testing.T) {
	// given
	renderer := NewSlackRenderer()
	singleSectionMessage := FixNonInteractiveSingleSection()

	// when
	out := renderer.RenderAsSlackBlocks(interactive.CoreMessage{
		Message: singleSectionMessage,
	})

	// then
	raw, err := json.MarshalIndent(out, "", "  ")
	require.NoError(t, err)

	golden.AssertBytes(t, raw, fmt.Sprintf("%s.golden.json", t.Name()))
}
