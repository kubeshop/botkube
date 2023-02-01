package source

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
)

func TestSourcesForEvent(t *testing.T) {
	// given
	allNsCfg := config.Namespaces{Include: []string{".*"}}
	testCases := []struct {
		Name               string
		Routes             []route
		Event              event.Event
		ExpectedResult     []string
		ExpectedErrMessage string
	}{
		{
			Name: "Event reason - success",
			Routes: []route{
				{
					source: "success",
					event: config.KubernetesEvent{
						Reason: "^NodeNotReady",
					},
					namespaces: allNsCfg,
				},
				{
					source:       "fail",
					resourceName: "^Created",
					namespaces:   allNsCfg,
				},
				{
					source: "fail2",
					event: config.KubernetesEvent{
						Reason: "^(?!NodeNotReady)$",
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name:   "test-one",
				Reason: "NodeNotReady",
			},
			ExpectedResult: []string{"success"},
		},
		{
			Name: "Event reason - error",
			Routes: []route{
				{
					source: "success",
					event: config.KubernetesEvent{
						Reason: "^NodeNotReady",
					},
					namespaces: allNsCfg,
				},
				{
					source:       "error",
					resourceName: "[",
					namespaces:   allNsCfg,
				},
			},
			Event: event.Event{
				Name:   "test-one",
				Reason: "NodeNotReady",
			},
			ExpectedResult: []string{"success"},
			ExpectedErrMessage: heredoc.Docf(`
				1 error occurred:
					* while compiling regex: error parsing regexp: unterminated [] set in %s`, "`[`"),
		},
		{
			Name: "Event message - success",
			Routes: []route{
				{
					source: "success",
					event: config.KubernetesEvent{
						Message: "^Status.*",
					},
					namespaces: allNsCfg,
				},
				{
					source: "success2",
					event: config.KubernetesEvent{
						Message: "^Second.*",
					},
					namespaces: allNsCfg,
				},
				{
					source: "fail",
					event: config.KubernetesEvent{
						Message: "^Resource",
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name: "test-one",
				Messages: []string{
					"Status one",
					"Second message",
					"Third",
				},
			},
			ExpectedResult: []string{"success", "success2"},
		},
		{
			Name: "Event message - negative lookahead",
			Routes: []route{
				{
					source: "success",
					event: config.KubernetesEvent{
						Message: "^(?!Back-off).*$",
					},
					namespaces: allNsCfg,
				},
				{
					source: "success2",
					event: config.KubernetesEvent{
						Message: "^(?!Back-off restarting failed container)$",
					},
					namespaces: allNsCfg,
				},
				{
					source: "empty",
					event: config.KubernetesEvent{
						Message: "",
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name: "test-one",
				Messages: []string{
					"Back-off restarting failed container",
				},
			},
			ExpectedResult: []string{"empty"},
		},
		{
			Name: "Event message - empty",
			Routes: []route{
				{
					source: "success",
					event: config.KubernetesEvent{
						Message: "^(?!Back-off).*$",
					},
					namespaces: allNsCfg,
				},
				{
					source: "success2",
					event: config.KubernetesEvent{
						Message: "^(?!Back-off restarting failed container)$",
					},
					namespaces: allNsCfg,
				},
				{
					source: "success3",
					event: config.KubernetesEvent{
						Message: "",
					},
					namespaces: allNsCfg,
				},
				{
					source: "fail",
					event: config.KubernetesEvent{
						Message: "^Back-off",
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name:     "test-one",
				Messages: nil,
			},
			ExpectedResult: []string{"success", "success2", "success3"},
		},
		{
			Name: "Event message - error",
			Routes: []route{
				{
					source: "success",
					event: config.KubernetesEvent{
						Message: "^Status.*",
					},
					namespaces: allNsCfg,
				},
				{
					source: "error",
					event: config.KubernetesEvent{
						Message: "[",
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name: "test-one",
				Messages: []string{
					"Status one",
					"Second message",
					"Third",
				},
			},
			ExpectedResult: []string{"success"},
			ExpectedErrMessage: heredoc.Docf(`
				1 error occurred:
					* while compiling regex: error parsing regexp: unterminated [] set in %s`, "`[`"),
		},
		{
			Name: "Resource name - success",
			Routes: []route{
				{
					source:       "success",
					resourceName: "^test-.*",
					namespaces:   allNsCfg,
				},
				{
					source:       "fail",
					resourceName: "^one-.*",
					namespaces:   allNsCfg,
				},
				{
					source:       "fail2",
					resourceName: "^(?!^test-).*$",
					namespaces:   allNsCfg,
				},
			},
			Event: event.Event{
				Name: "test-one",
			},
			ExpectedResult: []string{"success"},
		},
		{
			Name: "Resource name - error",
			Routes: []route{
				{
					source:       "success",
					resourceName: "^test-.*",
					namespaces:   allNsCfg,
				},
				{
					source:       "error",
					resourceName: "[",
					namespaces:   allNsCfg,
				},
			},
			Event: event.Event{
				Name: "test-one",
			},
			ExpectedResult: []string{"success"},
			ExpectedErrMessage: heredoc.Docf(`
				1 error occurred:
					* while compiling regex: error parsing regexp: unterminated [] set in %s`, "`[`"),
		},
		{
			Name: "Namespace",
			Routes: []route{
				{
					source:     "success",
					namespaces: config.Namespaces{Include: []string{"^botkube-.*"}},
				},
				{
					source:     "fail",
					namespaces: config.Namespaces{Include: []string{"^kube-.*"}},
				},
			},
			Event: event.Event{
				Name:      "test-one",
				Namespace: "botkube-one",
			},
			ExpectedResult: []string{"success"},
		},
		{
			Name: "Labels",
			Routes: []route{
				{
					source:     "success",
					namespaces: allNsCfg,
					labels: map[string]string{
						"my-label":  "my-value",
						"my-label2": "my-value2",
					},
				},
				{
					source: "success2",
					labels: map[string]string{
						"my-label": "my-value",
					},
					namespaces: allNsCfg,
				},
				{
					source: "fail",
					labels: map[string]string{
						"my-different-label": "my-value",
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name:      "test-one",
				Namespace: "botkube-one",
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"my-label":  "my-value",
						"my-label2": "my-value2",
						"my-label3": "my-value3",
					},
				},
			},
			ExpectedResult: []string{"success", "success2"},
		},
		{
			Name: "Annotations",
			Routes: []route{
				{
					source:     "success",
					namespaces: allNsCfg,
					annotations: map[string]string{
						"my-annotation":  "my-value",
						"my-annotation2": "my-value2",
					},
				},
				{
					source: "success2",
					annotations: map[string]string{
						"my-annotation": "my-value",
					},
					namespaces: allNsCfg,
				},
				{
					source: "fail",
					annotations: map[string]string{
						"my-different-annotation": "my-value",
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name:      "test-one",
				Namespace: "botkube-one",
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"my-annotation":  "my-value",
						"my-annotation2": "my-value2",
						"my-annotation3": "my-value3",
					},
				},
			},
			ExpectedResult: []string{"success", "success2"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			reg := registration{
				log: loggerx.NewNoop(),
			}

			// when
			res, err := reg.sourcesForEvent(testCase.Routes, testCase.Event)

			// then
			if testCase.ExpectedErrMessage != "" {
				assert.EqualError(t, err, testCase.ExpectedErrMessage)
				// continue anyway as there could be errors and results
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, testCase.ExpectedResult, res)
		})
	}
}
