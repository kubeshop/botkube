package recommendation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/internal/ptr"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
)

func TestResourceEventsForConfig(t *testing.T) {
	// given
	testCases := []struct {
		Name     string
		RecCfg   config.Recommendations
		Expected map[string]config.EventType
	}{
		{
			Name: "Pod Labels Set",
			RecCfg: config.Recommendations{
				Pod: config.PodRecommendations{
					LabelsSet: ptr.FromType(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.PodResourceType(): config.CreateEvent,
			},
		},
		{
			Name: "Pod No Latest Image Tag",
			RecCfg: config.Recommendations{
				Pod: config.PodRecommendations{
					NoLatestImageTag: ptr.FromType(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.PodResourceType(): config.CreateEvent,
			},
		},
		{
			Name: "Ingress Backend Service Valid",
			RecCfg: config.Recommendations{
				Ingress: config.IngressRecommendations{
					BackendServiceValid: ptr.FromType(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.IngressResourceType(): config.CreateEvent,
			},
		},
		{
			Name: "Ingress TLS Secret Valid",
			RecCfg: config.Recommendations{
				Ingress: config.IngressRecommendations{
					TLSSecretValid: ptr.FromType(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.IngressResourceType(): config.CreateEvent,
			},
		},
		{
			Name: "All",
			RecCfg: config.Recommendations{
				Pod: config.PodRecommendations{
					LabelsSet: ptr.FromType(true),
				},
				Ingress: config.IngressRecommendations{
					TLSSecretValid: ptr.FromType(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.PodResourceType():     config.CreateEvent,
				recommendation.IngressResourceType(): config.CreateEvent,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// when
			actual := recommendation.ResourceEventsForConfig(&testCase.RecCfg)

			// then
			assert.Equal(t, testCase.Expected, actual)
		})
	}
}

func TestShouldIgnoreEvent(t *testing.T) {
	// given
	testCases := []struct {
		Name                string
		InputConfig         config.Recommendations
		InputSourceBindings []string
		InputEvent          event.Event
		Expected            bool
	}{
		{
			Name: "Has recommendations",
			InputEvent: event.Event{
				Recommendations: []string{"message"},
			},
			Expected: false,
		},
		{
			Name: "Has warnings",
			InputEvent: event.Event{
				Warnings: []string{"message"},
			},
			Expected: false,
		},
		{
			Name:        "Different resource",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: event.Event{
				Resource: "v1/deployments",
				Type:     config.CreateEvent,
			},
			Expected: false,
		},
		{
			Name:        "Different event",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: event.Event{
				Resource: recommendation.PodResourceType(),
				Type:     config.UpdateEvent,
			},
			Expected: false,
		},
		{
			Name:        "User configured such event",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: event.Event{
				Resource:  recommendation.PodResourceType(),
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			InputSourceBindings: []string{"deployments", "pods"},
			Expected:            true,
		},
		{
			Name:        "User didn't configure such resource",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: event.Event{
				Resource:  recommendation.IngressResourceType(),
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			InputSourceBindings: []string{"deployments", "pods"},
			Expected:            true,
		},
		{
			Name:        "User didn't configure such event - different namespace",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: event.Event{
				Resource:  recommendation.PodResourceType(),
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			InputSourceBindings: []string{"deployments", "pods-ns"},
			Expected:            true,
		},
		{
			Name:        "User didn't configure such event - different events",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: event.Event{
				Resource:  recommendation.PodResourceType(),
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			InputSourceBindings: []string{"deployments", "pods-update"},
			Expected:            true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// when
			actual := recommendation.ShouldIgnoreEvent(&testCase.InputConfig, testCase.InputEvent)

			// then
			assert.Equal(t, testCase.Expected, actual)
		})
	}
}

func fixFullRecommendationConfig() config.Recommendations {
	return config.Recommendations{
		Pod: config.PodRecommendations{
			NoLatestImageTag: ptr.FromType(true),
			LabelsSet:        ptr.FromType(true),
		},
		Ingress: config.IngressRecommendations{
			BackendServiceValid: ptr.FromType(true),
			TLSSecretValid:      ptr.FromType(true),
		},
	}
}
