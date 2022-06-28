package execute

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
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
