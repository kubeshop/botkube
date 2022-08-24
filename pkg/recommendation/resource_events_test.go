package recommendation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/ptr"
	"github.com/kubeshop/botkube/pkg/recommendation"
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
					LabelsSet: ptr.Bool(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.PodResourceName(): config.CreateEvent,
			},
		},
		{
			Name: "Pod No Latest Image Tag",
			RecCfg: config.Recommendations{
				Pod: config.PodRecommendations{
					NoLatestImageTag: ptr.Bool(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.PodResourceName(): config.CreateEvent,
			},
		},
		{
			Name: "Ingress Backend Service Valid",
			RecCfg: config.Recommendations{
				Ingress: config.IngressRecommendations{
					BackendServiceValid: ptr.Bool(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.IngressResourceName(): config.CreateEvent,
			},
		},
		{
			Name: "Ingress TLS Secret Valid",
			RecCfg: config.Recommendations{
				Ingress: config.IngressRecommendations{
					TLSSecretValid: ptr.Bool(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.IngressResourceName(): config.CreateEvent,
			},
		},
		{
			Name: "All",
			RecCfg: config.Recommendations{
				Pod: config.PodRecommendations{
					LabelsSet: ptr.Bool(true),
				},
				Ingress: config.IngressRecommendations{
					TLSSecretValid: ptr.Bool(true),
				},
			},
			Expected: map[string]config.EventType{
				recommendation.PodResourceName():     config.CreateEvent,
				recommendation.IngressResourceName(): config.CreateEvent,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// when
			actual := recommendation.ResourceEventsForConfig(testCase.RecCfg)

			// then
			assert.Equal(t, testCase.Expected, actual)
		})
	}
}

func TestShouldIgnoreEvent(t *testing.T) {
	// given
	sources := fixSources()
	testCases := []struct {
		Name                string
		InputConfig         config.Recommendations
		InputSourceBindings []string
		InputEvent          events.Event
		Expected            bool
	}{
		{
			Name: "Has recommendations",
			InputEvent: events.Event{
				Recommendations: []string{"message"},
			},
			Expected: false,
		},
		{
			Name: "Has warnings",
			InputEvent: events.Event{
				Warnings: []string{"message"},
			},
			Expected: false,
		},
		{
			Name:        "Different resource",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: events.Event{
				Resource: "v1/deployments",
				Type:     config.CreateEvent,
			},
			Expected: false,
		},
		{
			Name:        "Different event",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: events.Event{
				Resource: recommendation.PodResourceName(),
				Type:     config.UpdateEvent,
			},
			Expected: false,
		},
		{
			Name:        "User configured such event",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: events.Event{
				Resource:  recommendation.PodResourceName(),
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			InputSourceBindings: []string{"deployments", "pods"},
			Expected:            false,
		},
		{
			Name:        "User didn't configure such resource",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: events.Event{
				Resource:  recommendation.IngressResourceName(),
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			InputSourceBindings: []string{"deployments", "pods"},
			Expected:            true,
		},
		{
			Name:        "User didn't configure such event - different namespace",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: events.Event{
				Resource:  recommendation.PodResourceName(),
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			InputSourceBindings: []string{"deployments", "pods-ns"},
			Expected:            true,
		},
		{
			Name:        "User didn't configure such event - different events",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: events.Event{
				Resource:  recommendation.PodResourceName(),
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
			actual := recommendation.ShouldIgnoreEvent(testCase.InputConfig, sources, testCase.InputSourceBindings, testCase.InputEvent)

			// then
			assert.Equal(t, testCase.Expected, actual)
		})
	}
}

func fixFullRecommendationConfig() config.Recommendations {
	return config.Recommendations{
		Pod: config.PodRecommendations{
			NoLatestImageTag: ptr.Bool(true),
			LabelsSet:        ptr.Bool(true),
		},
		Ingress: config.IngressRecommendations{
			BackendServiceValid: ptr.Bool(true),
			TLSSecretValid:      ptr.Bool(true),
		},
	}
}

func fixSources() map[string]config.Sources {
	return map[string]config.Sources{
		"deployments": {
			Kubernetes: config.KubernetesSource{
				Resources: []config.Resource{
					{
						Name:       "v1/deployments",
						Namespaces: config.Namespaces{},
						Events:     []config.EventType{config.AllEvent},
					},
				},
			},
		},
		"pods": {
			Kubernetes: config.KubernetesSource{
				Resources: []config.Resource{
					{
						Name: recommendation.PodResourceName(),
						Namespaces: config.Namespaces{
							Include: []string{".*"},
						},
						Events: []config.EventType{config.AllEvent},
					},
				},
			},
		},
		"pods-ns": {
			Kubernetes: config.KubernetesSource{
				Resources: []config.Resource{
					{
						Name: recommendation.PodResourceName(),
						Namespaces: config.Namespaces{
							Include: []string{"kube-system"},
						},
						Events: []config.EventType{config.AllEvent},
					},
				},
			},
		},
		"pods-update": {
			Kubernetes: config.KubernetesSource{
				Resources: []config.Resource{
					{
						Name: recommendation.PodResourceName(),
						Namespaces: config.Namespaces{
							Include: []string{".*"},
						},
						Events: []config.EventType{config.UpdateEvent},
					},
				},
			},
		},
	}
}
