package kubectl

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/config"
)

func TestSetDefaultNamespace(t *testing.T) {
	tests := []struct {
		name         string
		givenConfig  string
		givenCommand string
		expCommand   string
	}{
		{
			name:         "Short namespace flag is set",
			givenCommand: "kubectl get pods -n test",
			expCommand:   "kubectl get pods -n test",
		},
		{
			name:         "Long namespace flag is set",
			givenCommand: "kubectl get pods --namespace test",
			expCommand:   "kubectl get pods --namespace test",
		},
		{
			name:         "All namespaces flag is set",
			givenCommand: "kubectl get pods -A",
			expCommand:   "kubectl get pods -A",
		},
		{
			name: "No namespace flag is set, use config",
			givenConfig: heredoc.Doc(`
			defaultNamespace: "cfg-test"
			`),
			givenCommand: "kubectl get pods",
			expCommand:   "kubectl -n cfg-test get pods",
		},
		{
			name:         "No namespace flag is set, no config",
			givenCommand: "kubectl get pods",
			expCommand:   "kubectl -n default get pods",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var gotCmd string
			mockFn := NewMockedBinaryRunner(func(_ context.Context, rawCmd string, _ map[string]string) (string, error) {
				gotCmd = rawCmd
				return "mocked", nil
			})

			exec := NewExecutor(loggerx.NewNoop(), "dev", mockFn)
			// when
			_, err := exec.Execute(context.Background(), executor.ExecuteInput{
				Command: tc.givenCommand,
				Configs: []*executor.Config{
					{
						RawYAML: []byte(tc.givenConfig),
					},
				},
			})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expCommand, gotCmd)
		})
	}
}

func TestSetOptionsCommand(t *testing.T) {
	tests := []struct {
		name         string
		givenCommand string
		expCommand   string
	}{
		{
			name:         "Normal kubectl options",
			givenCommand: "kubectl options",
		},
		{
			name:         "Handle whitespaces",
			givenCommand: "kubectl         options",
		},
		{
			name:         "Handle whitespaces all around",
			givenCommand: "kubectl         options            ",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var wasKubectlCalled bool
			mockFn := NewMockedBinaryRunner(func(_ context.Context, rawCmd string, _ map[string]string) (string, error) {
				wasKubectlCalled = true
				return "mocked", nil
			})

			exec := NewExecutor(loggerx.NewNoop(), "dev", mockFn)

			// when
			out, err := exec.Execute(context.Background(), executor.ExecuteInput{
				Command: tc.givenCommand,
			})

			// then
			require.NoError(t, err)
			assert.False(t, wasKubectlCalled)
			assert.Equal(t, optionsCommandOutput(), out.Message.BaseBody.CodeBlock)
		})
	}
}

func TestNotSupportedCommandsAndFlags(t *testing.T) {
	tests := []struct {
		name         string
		givenCommand string
		expErr       string
	}{
		{
			name:         "Not supported proxy",
			givenCommand: "kubectl proxy --www=/my/files --www-prefix=/static/ --api-prefix=/api/",
			expErr:       `The "proxy" command is not supported by the Botkube kubectl plugin.`,
		},
		{
			name:         "Not supported edit",
			givenCommand: "kubectl       edit     pod/foo",
			expErr:       `The "edit" command is not supported by the Botkube kubectl plugin.`,
		},
		{
			name:         "Not supported flags",
			givenCommand: "kubectl get pod --as foo-account",
			expErr:       `The "as" flag is not supported by the Botkube kubectl plugin. Please remove it.`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var wasKubectlCalled bool
			mockFn := NewMockedBinaryRunner(func(_ context.Context, rawCmd string, _ map[string]string) (string, error) {
				wasKubectlCalled = true
				return "mocked", nil
			})

			exec := NewExecutor(loggerx.NewNoop(), "dev", mockFn)

			// when
			out, err := exec.Execute(context.Background(), executor.ExecuteInput{
				Command: tc.givenCommand,
			})

			// then
			assert.False(t, wasKubectlCalled)
			assert.Empty(t, out)
			assert.EqualError(t, err, tc.expErr)
		})
	}
}

func TestCommandsBuilder(t *testing.T) {
	tests := []struct {
		name         string
		givenCommand string
		platform     config.CommPlatformIntegration
		expMsg       string
	}{
		{
			name:         "Should respond specify full command",
			givenCommand: "kubectl",
			expMsg:       "Please specify the kubectl command",
			platform:     config.DiscordCommPlatformIntegration,
		},
		{
			name:         "Should start interactive mode",
			givenCommand: "kubectl",
			expMsg:       "interactivity not yet implemented",
			platform:     config.SocketSlackCommPlatformIntegration,
		},
		{
			name:         "Should respond specify full command",
			givenCommand: "kubectl @builder",
			expMsg:       "Please specify the kubectl command",
			platform:     config.DiscordCommPlatformIntegration,
		},
		{
			name:         "Should continue interactive mode",
			givenCommand: "kubectl @builder",
			expMsg:       "interactivity not yet implemented",
			platform:     config.SocketSlackCommPlatformIntegration,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var wasKubectlCalled bool
			mockFn := NewMockedBinaryRunner(func(_ context.Context, rawCmd string, _ map[string]string) (string, error) {
				wasKubectlCalled = true
				return "mocked", nil
			})

			exec := NewExecutor(loggerx.NewNoop(), "dev", mockFn)

			// when
			out, err := exec.Execute(context.Background(), executor.ExecuteInput{
				Command: tc.givenCommand,
				Context: executor.ExecuteInputContext{
					IsInteractivitySupported: tc.platform.IsInteractive(),
				},
			})

			// then
			require.NoError(t, err)
			assert.False(t, wasKubectlCalled)
			assert.Equal(t, api.NewPlaintextMessage(tc.expMsg, true), out.Message)
		})
	}
}
