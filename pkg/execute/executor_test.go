package execute

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestDefaultExecutor_getSortedEnabledCommands(t *testing.T) {
	testCases := []struct {
		Name           string
		InputHeader    string
		InputMap       map[string]bool
		ExpectedOutput string
	}{
		{
			Name:        "All commands disabled",
			InputHeader: "test",
			InputMap: map[string]bool{
				"cmd1": false,
				"cmd2": false,
			},
			ExpectedOutput: "test: []",
		},
		{
			Name:        "All commands enabled",
			InputHeader: "foo",
			InputMap: map[string]bool{
				"b1": true,
				"a2": false,
				"a3": true,
				"a4": true,
				"a5": true,
				"a6": false,
			},
			ExpectedOutput: heredoc.Doc(`
				foo:
				  - a3
				  - a4
				  - a5
				  - b1
			`),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			e := &DefaultExecutor{}
			res := e.getSortedEnabledCommands(testCase.InputHeader, testCase.InputMap)

			assert.Equal(t, testCase.ExpectedOutput, res)
		})
	}
}

func TestDefaultExecutor_getSortedNamespaceConfig(t *testing.T) {
	testCases := []struct {
		name      string
		nsConfig  config.Namespaces
		expOutput string
	}{
		{
			name: "All namespaces enabled",
			nsConfig: config.Namespaces{
				Include: []string{config.AllNamespaceIndicator},
			},
			expOutput: heredoc.Doc(`
				allowed namespaces:
				  include:
				    - all
			`),
		},
		{
			name: "All namespace enabled and a few ignored",
			nsConfig: config.Namespaces{
				Include: []string{config.AllNamespaceIndicator},
				Ignore:  []string{"demo", "abc", "ns-*-test"},
			},
			expOutput: heredoc.Doc(`
				allowed namespaces:
				  include:
				    - all
				  ignore:
				    - demo
				    - abc
				    - ns-*-test
				`),
		},
		{
			name: "Only some namespace enabled",
			nsConfig: config.Namespaces{
				Include: []string{"demo", "abc"},
			},
			expOutput: heredoc.Doc(`
				allowed namespaces:
				  include:
				    - demo
				    - abc
			 `),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			e := &DefaultExecutor{cfg: config.Config{
				Executors: config.IndexableMap[config.Executors]{
					"default": config.Executors{
						Kubectl: config.Kubectl{
							Namespaces: tc.nsConfig,
						},
					},
				},
			}}

			// when
			res, err := e.getNamespaceConfig()

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expOutput, res)
		})
	}
}
