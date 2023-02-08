package recommendation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
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
				recommendation.PodResourceType(): config.CreateEvent,
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
				recommendation.PodResourceType(): config.CreateEvent,
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
				recommendation.IngressResourceType(): config.CreateEvent,
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
				recommendation.IngressResourceType(): config.CreateEvent,
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
				recommendation.PodResourceType():     config.CreateEvent,
				recommendation.IngressResourceType(): config.CreateEvent,
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
			Expected:            false,
		},
		{
			Name:        "User configured such event with source-wide namespace",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: event.Event{
				Resource:  recommendation.PodResourceType(),
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			InputSourceBindings: []string{"deployments", "pods-source-wide-ns"},
			Expected:            false,
		},
		{
			Name:        "User configured such event with source-wide namespace and resource ns override",
			InputConfig: fixFullRecommendationConfig(),
			InputEvent: event.Event{
				Resource:  recommendation.PodResourceType(),
				Namespace: "kube-system",
				Type:      config.CreateEvent,
			},
			InputSourceBindings: []string{"deployments", "pods-ns-override"},
			Expected:            false,
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
						Type:       "v1/deployments",
						Namespaces: config.RegexConstraints{},
						Event: config.KubernetesEvent{
							Types: []config.EventType{config.AllEvent},
						},
					},
				},
			},
		},
		"pods": {
			Kubernetes: config.KubernetesSource{
				Resources: []config.Resource{
					{
						Type: recommendation.PodResourceType(),
						Namespaces: config.RegexConstraints{
							Include: []string{".*"},
						},
						Event: config.KubernetesEvent{
							Types: []config.EventType{config.AllEvent},
						},
					},
				},
			},
		},
		"pods-source-wide-ns": {
			Kubernetes: config.KubernetesSource{
				Namespaces: config.RegexConstraints{
					Include: []string{".*"},
				},
				Resources: []config.Resource{
					{
						Type: recommendation.PodResourceType(),
						Event: config.KubernetesEvent{
							Types: []config.EventType{config.AllEvent},
						},
					},
				},
			},
		},
		"pods-ns-override": {
			Kubernetes: config.KubernetesSource{
				Namespaces: config.RegexConstraints{
					Include: []string{"default"},
				},
				Resources: []config.Resource{
					{
						Type: recommendation.PodResourceType(),
						Namespaces: config.RegexConstraints{
							Include: []string{"kube-system"},
						},
						Event: config.KubernetesEvent{
							Types: []config.EventType{config.AllEvent},
						},
					},
				},
			},
		},
		"pods-ns": {
			Kubernetes: config.KubernetesSource{
				Resources: []config.Resource{
					{
						Type: recommendation.PodResourceType(),
						Namespaces: config.RegexConstraints{
							Include: []string{"kube-system"},
						},
						Event: config.KubernetesEvent{
							Types: []config.EventType{config.AllEvent},
						},
					},
				},
			},
		},
		"pods-update": {
			Kubernetes: config.KubernetesSource{
				Resources: []config.Resource{
					{
						Type: recommendation.PodResourceType(),
						Namespaces: config.RegexConstraints{
							Include: []string{".*"},
						},
						Event: config.KubernetesEvent{
							Types: []config.EventType{config.UpdateEvent},
						},
					},
				},
			},
		},
	}
}
