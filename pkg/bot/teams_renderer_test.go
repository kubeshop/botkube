package bot

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

func TestTeamsNonInteractiveSectionToCard(t *testing.T) {
	// given
	renderer := NewTeamsRenderer()
	singleSectionMessage := FixNonInteractiveSingleSection()

	// when
	out, err := renderer.NonInteractiveSectionToCard(interactive.CoreMessage{
		Message: singleSectionMessage,
	})

	// then
	require.NoError(t, err)

	raw, err := json.MarshalIndent(out, "", "  ")
	require.NoError(t, err)

	golden.AssertBytes(t, raw, fmt.Sprintf("%s.golden.json", t.Name()))
}
