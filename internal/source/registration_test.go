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
	allNsCfg := config.RegexConstraints{Include: []string{".*"}}
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
						Reason: config.RegexConstraints{
							Include: []string{"^NodeNotReady"},
						},
					},
					namespaces: allNsCfg,
				},
				{
					source: "success-empty",
					event: config.KubernetesEvent{
						Reason: config.RegexConstraints{},
					},
					namespaces: allNsCfg,
				},
				{
					source: "fail",
					event: config.KubernetesEvent{
						Reason: config.RegexConstraints{
							Include: []string{"^Created"},
						},
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name:   "test-one",
				Reason: "NodeNotReady",
			},
			ExpectedResult: []string{"success", "success-empty"},
		},
		{
			Name: "Event reason - error",
			Routes: []route{
				{
					source: "success",
					event: config.KubernetesEvent{
						Reason: config.RegexConstraints{
							Include: []string{"^NodeNotReady"},
						},
					},
					namespaces: allNsCfg,
				},
				{
					source: "error",
					event: config.KubernetesEvent{
						Reason: config.RegexConstraints{
							Exclude: []string{"["},
						},
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name:   "test-one",
				Reason: "NodeNotReady",
			},
			ExpectedResult: []string{"success"},
			ExpectedErrMessage: heredoc.Docf(`
				1 error occurred:
					* while matching "NodeNotReady" with exclude regex "[": error parsing regexp: missing closing ]: %s`, "`[`"),
		},
		{
			Name: "Event message - success",
			Routes: []route{
				{
					source: "success",
					event: config.KubernetesEvent{
						Message: config.RegexConstraints{
							Include: []string{"^Status.*"},
						},
					},
					namespaces: allNsCfg,
				},
				{
					source: "success2",
					event: config.KubernetesEvent{
						Message: config.RegexConstraints{
							Include: []string{"^Second.*"},
						},
					},
					namespaces: allNsCfg,
				},
				{
					source: "success3",
					event: config.KubernetesEvent{
						Message: config.RegexConstraints{
							Include: []string{".*"},
							Exclude: []string{"^Something.*"},
						},
					},
					namespaces: allNsCfg,
				},
				{
					source: "success-empty",
					event: config.KubernetesEvent{
						Message: config.RegexConstraints{},
					},
					namespaces: allNsCfg,
				},
				{
					source: "fail",
					event: config.KubernetesEvent{
						Message: config.RegexConstraints{
							Include: []string{"^Resource"},
						},
					},
					namespaces: allNsCfg,
				},
				{
					source: "fail2",
					event: config.KubernetesEvent{
						Message: config.RegexConstraints{
							Include: []string{".*"},
							Exclude: []string{"^Second", "^Status", "^Third"},
						},
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
			ExpectedResult: []string{"success", "success2", "success3", "success-empty"},
		},
		{
			Name: "Event message - error",
			Routes: []route{
				{
					source: "success",
					event: config.KubernetesEvent{
						Message: config.RegexConstraints{
							Include: []string{"^Status.*"},
						},
					},
					namespaces: allNsCfg,
				},
				{
					source: "error",
					event: config.KubernetesEvent{
						Message: config.RegexConstraints{
							Include: []string{"["},
						},
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
					* while matching "Third" with include regex "[": error parsing regexp: missing closing ]: %s`, "`[`"),
		},
		{
			Name: "Resource name - success",
			Routes: []route{
				{
					source: "success",
					resourceName: config.RegexConstraints{
						Include: []string{"^test-.*"},
					},
					namespaces: allNsCfg,
				},
				{
					source:       "success-empty",
					resourceName: config.RegexConstraints{},
					namespaces:   allNsCfg,
				},
				{
					source: "fail",
					resourceName: config.RegexConstraints{
						Include: []string{"^one-.*"},
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name: "test-one",
			},
			ExpectedResult: []string{"success", "success-empty"},
		},
		{
			Name: "Resource name - error",
			Routes: []route{
				{
					source: "success",
					resourceName: config.RegexConstraints{
						Include: []string{"^test-.*"},
					},
					namespaces: allNsCfg,
				},
				{
					source: "error",
					resourceName: config.RegexConstraints{
						Include: []string{"["},
					},
					namespaces: allNsCfg,
				},
			},
			Event: event.Event{
				Name: "test-one",
			},
			ExpectedResult: []string{"success"},
			ExpectedErrMessage: heredoc.Docf(`
				1 error occurred:
					* while matching "test-one" with include regex "[": error parsing regexp: missing closing ]: %s`, "`[`"),
		},
		{
			Name: "Namespace",
			Routes: []route{
				{
					source:     "success",
					namespaces: config.RegexConstraints{Include: []string{"^botkube-.*"}},
				},
				{
					source: "success-empty",
					namespaces: config.RegexConstraints{},
				},
				{
					source:     "fail",
					namespaces: config.RegexConstraints{Include: []string{"^kube-.*"}},
				},
			},
			Event: event.Event{
				Name:      "test-one",
				Namespace: "botkube-one",
			},
			ExpectedResult: []string{"success", "success-empty"},
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
