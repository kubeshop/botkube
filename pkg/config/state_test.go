package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionsSetEnabled(t *testing.T) {
	tests := map[string]struct {
		name    string
		enabled bool
		actions *ActionsRuntimeState
		err     error
	}{
		"Enable - changed": {
			name:    "test",
			enabled: true,
			actions: &ActionsRuntimeState{"test": ActionRuntimeState{Enabled: false}},
			err:     nil,
		},
		"Enable - already enabled": {
			name:    "test",
			enabled: true,
			actions: &ActionsRuntimeState{"test": ActionRuntimeState{Enabled: true}},
			err:     nil,
		},
		"Fail": {
			name:    "test",
			enabled: false,
			actions: &ActionsRuntimeState{"bogus": ActionRuntimeState{Enabled: false}},
			err:     fmt.Errorf("action with name %q not found", "test"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			e := test.actions.SetEnabled(test.name, test.enabled)
			assert.Equal(t, test.err, e)
			assert.Equal(t, test.enabled, (*test.actions)[test.name].Enabled)
		})
	}
}
