package bot

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"

	"github.com/kubeshop/botkube/pkg/api"
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

func TestSlackActionID(t *testing.T) {
	longDesc := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
	tests := []struct {
		name string
		btn  api.Button
	}{
		{
			name: "Btn command",
			btn: api.Button{
				Name:    longDesc,
				Command: longDesc,
			},
		},
		{
			name: "Btn URL",
			btn: api.Button{
				Name: longDesc,
				URL:  fmt.Sprintf("http://localhost?state='%s'", longDesc),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			renderer := NewSlackRenderer()

			// when
			btnID := renderer.genBtnActionID(tc.btn)

			// then
			if tc.btn.Command != "" {
				assert.True(t, len(tc.btn.Command) > 255)
			}
			if tc.btn.URL != "" {
				assert.True(t, len(tc.btn.URL) > 255)
			}
			assert.True(t, len(btnID) < 255)
		})
	}
}
