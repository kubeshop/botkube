package event

import (
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnonymizedEventDetailsFrom(t *testing.T) {
	// given
	testCases := []struct {
		Name           string
		InputEvent     Event
		ExpectedOutput map[string]interface{}
	}{
		{
			Name: "Allowed API Version",
			InputEvent: Event{
				Type:       config.CreateEvent,
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			ExpectedOutput: map[string]interface{}{
				"Type":       config.CreateEvent,
				"APIVersion": "apps/v1",
				"Kind":       "Deployment",
			},
		},
		{
			Name: "Disallowed API Version",
			InputEvent: Event{
				Type:       config.CreateEvent,
				APIVersion: "commercial.example.com/v1",
				Kind:       "Commercial Data",
			},
			ExpectedOutput: map[string]interface{}{
				"Type":       config.CreateEvent,
				"APIVersion": "other",
				"Kind":       "other",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// when
			res := AnonymizedEventDetailsFrom(tc.InputEvent)

			// then
			assert.Equal(t, tc.ExpectedOutput, res)
		})
	}
}
