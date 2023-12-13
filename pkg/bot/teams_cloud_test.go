package bot

import (
	"testing"

	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/execute/command"
)

func TestExtractExplicitOrigin(t *testing.T) {
	tests := []struct {
		name       string
		givenAct   schema.Activity
		wantOrigin command.Origin
	}{
		{
			name: "Should return explicit origin",
			givenAct: schema.Activity{
				Type:  schema.Message,
				Value: explicitOriginValue(string(command.MultiSelectValueChangeOrigin)),
			},
			wantOrigin: command.MultiSelectValueChangeOrigin,
		},
		{
			name: "Should return typed message resolved from type due to invalid value",
			givenAct: schema.Activity{
				Type:  schema.Message,
				Value: explicitOriginValue("malformed-or-unknown-origin"),
			},
			wantOrigin: command.TypedOrigin,
		},
		{
			name: "Should return btn click origin resolved from type because value is nil",
			givenAct: schema.Activity{
				Type:  schema.Invoke,
				Value: nil,
			},
			wantOrigin: command.ButtonClickOrigin,
		},
		{
			name:       "Should return unknown origin because value does not contain origin key and type is empty",
			givenAct:   schema.Activity{Value: map[string]any{}},
			wantOrigin: command.UnknownOrigin,
		},
	}

	cloudTeam := &CloudTeams{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOrigin := cloudTeam.mapToCommandOrigin(tc.givenAct)
			assert.Equal(t, tc.wantOrigin, gotOrigin)
		})
	}
}

func explicitOriginValue(in string) map[string]any {
	return map[string]any{
		originKeyName: in,
	}
}
