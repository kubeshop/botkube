package analytics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/event"
)

func TestAnonymizedEventDetailsFrom(t *testing.T) {
	// given
	testCases := []struct {
		Name           string
		InputEvent     event.Event
		ExpectedOutput analytics.EventDetails
	}{
		{
			Name: "Allowed API Version",
			InputEvent: event.Event{
				Type: "create",
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
			},
			ExpectedOutput: analytics.EventDetails{
				Type:       "create",
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
		},
		{
			Name: "Disallowed API Version",
			InputEvent: event.Event{
				Type: "create",
				TypeMeta: metav1.TypeMeta{
					APIVersion: "commercial.example.com/v1",
					Kind:       "Commercial Event",
				},
			},
			ExpectedOutput: analytics.EventDetails{
				Type:       "create",
				APIVersion: "other",
				Kind:       "other",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// when
			res := analytics.AnonymizedEventDetailsFrom(tc.InputEvent)

			// then
			assert.Equal(t, tc.ExpectedOutput, res)
		})
	}
}
