package x

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/executor/x/getter"
	"github.com/kubeshop/botkube/internal/executor/x/state"
	"github.com/kubeshop/botkube/internal/executor/x/template"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/kubeshop/botkube/pkg/plugin"
)

func TestRunnerRawOutput(t *testing.T) {
	// given
	cmd := Command{
		ToExecute:     "test command",
		IsRawRequired: true,
	}
	runExecuted := false
	runFn := func() (string, error) {
		runExecuted = true
		return "command output", nil
	}
	expMsg := api.NewCodeBlockMessage("command output", true)

	runner := NewRunner(loggerx.NewNoop(), nil)

	// when
	output, err := runner.Run(context.Background(), Config{}, nil, cmd, runFn)

	// then
	assert.NoError(t, err)
	assert.True(t, runExecuted)
	assert.Equal(t, expMsg, output.Message)
}

func TestRunnerNoTemplates(t *testing.T) {
	// given
	cmd := Command{
		ToExecute: "test command",
	}
	runExecuted := false
	runFn := func() (string, error) {
		runExecuted = true
		return "command output", nil
	}
	expMsg := api.NewCodeBlockMessage("command output", true)

	runner := NewRunner(loggerx.NewNoop(), nil)

	// when
	output, err := runner.Run(context.Background(), Config{}, nil, cmd, runFn)

	// then
	assert.NoError(t, err)
	assert.True(t, runExecuted)
	assert.Equal(t, expMsg, output.Message)
}

func TestRunnerNoExecuteTemplate(t *testing.T) {
	// given
	cmd := Command{
		ToExecute: "quickstart helm",
	}
	cfg := Config{
		Templates: []getter.Source{
			{
				Ref: filepath.Join("./testdata/", t.Name()),
			},
		},
		TmpDir: plugin.TmpDir(t.TempDir()),
		Logger: config.Logger{},
	}

	runExecuted := false
	runFn := func() (string, error) {
		runExecuted = true
		return "command output", nil
	}

	expMsg := api.NewCodeBlockMessage(cmd.ToExecute, false)

	renderer := NewRenderer()
	err := renderer.Register("tutorial", &MockRenderer{})
	require.NoError(t, err)

	runner := NewRunner(loggerx.NewNoop(), renderer)

	// when
	output, err := runner.Run(context.Background(), cfg, nil, cmd, runFn)

	// then
	assert.NoError(t, err)
	assert.False(t, runExecuted)
	assert.Equal(t, expMsg, output.Message)
}

func TestRunnerExecuteError(t *testing.T) {
	// given
	cmd := Command{
		ToExecute: "test command",
	}
	fixErr := errors.New("fix error")
	runFn := func() (string, error) {
		return "", fixErr
	}

	runner := NewRunner(loggerx.NewNoop(), nil)

	// when
	output, err := runner.Run(context.Background(), Config{}, nil, cmd, runFn)

	// then
	assert.EqualError(t, err, fixErr.Error())
	assert.Empty(t, output)
}

type MockRenderer struct{}

func (r *MockRenderer) RenderMessage(cmd, output string, state *state.Container, msgCtx *template.Template) (api.Message, error) {
	return api.NewCodeBlockMessage(cmd, false), nil
}
