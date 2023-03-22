package helm

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/golden"

	"github.com/kubeshop/botkube/pkg/api/executor"
)

const kc = "KUBECONFIG"

func TestExecutorHelmInstall(t *testing.T) {
	tests := []struct {
		name         string
		inputCommand string
		expCommand   string
	}{
		{
			name:         "install by absolute URL with custom name",
			inputCommand: "helm install postgresql https://charts.bitnami.com/bitnami/postgresql-12.1.0.tgz --create-namespace -n test2 --set clusterDomain='testing.local'",
			expCommand:   "helm install postgresql https://charts.bitnami.com/bitnami/postgresql-12.1.0.tgz --create-namespace -n test2 --set clusterDomain='testing.local'",
		},
		{
			name:         "install by absolute URL with generate name",
			inputCommand: "helm install https://charts.bitnami.com/bitnami/postgresql-12.1.0.tgz --create-namespace -n test2 --generate-name --set clusterDomain='testing.local'",
			expCommand:   "helm install https://charts.bitnami.com/bitnami/postgresql-12.1.0.tgz --create-namespace -n test2 --generate-name --set clusterDomain='testing.local'",
		},
		{
			name:         "install by chart reference and repo URL",
			inputCommand: "helm install --repo https://example.com/charts/ mynginx nginx",
			expCommand:   "helm install --repo https://example.com/charts/ mynginx nginx -n default",
		},
		{
			name:         "install by chart reference and repo URL and with a given version",
			inputCommand: "helm install --repo https://example.com/charts/ mynginx nginx --version 1.2.3",
			expCommand:   "helm install --repo https://example.com/charts/ mynginx nginx --version 1.2.3 -n default",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			const execOutput = "mocked"

			hExec := NewExecutor("testing")

			var (
				gotCmd  string
				gotEnvs map[string]string
			)
			hExec.executeCommandWithEnvs = func(_ context.Context, rawCmd string, envs map[string]string) (string, error) {
				gotCmd = rawCmd
				gotEnvs = envs
				return execOutput, nil
			}

			// when
			out, err := hExec.Execute(context.Background(), executor.ExecuteInput{
				Command: tc.inputCommand,
				Context: executor.ExecuteInputContext{
					KubeConfig: []byte("not empty"),
				},
			})

			// then
			require.NoError(t, err)

			assert.Equal(t, execOutput, out.Data)

			assert.Equal(t, tc.expCommand, gotCmd)

			_, ok := gotEnvs[kc]
			assert.Equal(t, true, ok)
			delete(gotEnvs, kc)
			assert.Equal(t, map[string]string{
				"HELM_DRIVER":      "secret",
				"HELM_CACHE_HOME":  "/tmp/helm/.cache",
				"HELM_CONFIG_HOME": "/tmp/helm/",
			}, gotEnvs)
		})
	}
}

func TestExecutorHelmInstallFlagsErrors(t *testing.T) {
	tests := []struct {
		name         string
		inputCommand string
		expErrMsg    string
	}{
		{
			name:         "report issue about unknown flag",
			inputCommand: "helm install postgresql https://charts.bitnami.com/bitnami/postgresql-12.1.0.tgz --some-random-flag",
			expErrMsg:    "while parsing input command: unknown argument --some-random-flag",
		},
		{
			name:         "report issue known but not supported flag",
			inputCommand: "helm install https://charts.bitnami.com/bitnami/postgresql-12.1.0.tgz --generate-name --wait",
			expErrMsg:    `The "--wait" flag is not supported by the Botkube Helm plugin. Please remove it.`,
		},
		{
			name:         "install by OCI registry",
			inputCommand: "helm install mynginx --version 1.2.3 oci://example.com/charts/nginx",
			expErrMsg:    "Installing Helm chart from OCI registry is not supported.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			hExec := NewExecutor("testing")
			hExec.executeCommandWithEnvs = noopExecuteCommand

			// when
			out, err := hExec.Execute(context.Background(), executor.ExecuteInput{
				Command: tc.inputCommand,
				Context: executor.ExecuteInputContext{
					KubeConfig: []byte("not empty"),
				},
			})

			// then
			require.EqualError(t, err, tc.expErrMsg)
			assert.Empty(t, out.Data)
		})
	}
}

func TestExecutorHelmInstallHelp(t *testing.T) {
	goldenFilepath := fmt.Sprintf("%s.txt", t.Name())
	tests := []struct {
		name         string
		inputCommand string
	}{
		{
			name:         "should detect help flag",
			inputCommand: "helm install --help",
		},
		{
			name:         "detect help flag when other parameters are also specified",
			inputCommand: "helm install postgresql https://charts.bitnami.com/bitnami/postgresql-12.1.0.tgz --help",
		},
		{
			name:         "detect short version of help flag",
			inputCommand: "helm install https://charts.bitnami.com/bitnami/postgresql-12.1.0.tgz -h",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			hExec := NewExecutor("testing")
			hExec.executeCommandWithEnvs = noopExecuteCommand

			// when
			out, err := hExec.Execute(context.Background(), executor.ExecuteInput{
				Command: tc.inputCommand,
				Context: executor.ExecuteInputContext{
					KubeConfig: []byte("not empty"),
				},
			})

			// then
			require.NoError(t, err)
			golden.Assert(t, out.Data, goldenFilepath)
		})
	}
}

func TestExecutorConfigMerging(t *testing.T) {
	// given
	hExec := NewExecutor("testing")
	var gotEnvs map[string]string
	hExec.executeCommandWithEnvs = func(_ context.Context, _ string, envs map[string]string) (string, error) {
		gotEnvs = envs
		return "", nil
	}

	configA := Config{
		HelmDriver: "configmap",
	}
	configB := Config{
		HelmDriver: "secret",
	}

	// when
	_, err := hExec.Execute(context.Background(), executor.ExecuteInput{
		Command: "helm install",
		Configs: []*executor.Config{
			{
				RawYAML: mustYAMLMarshal(t, configA),
			},
			{
				RawYAML: mustYAMLMarshal(t, configB),
			},
		},
		Context: executor.ExecuteInputContext{
			KubeConfig: []byte("not empty"),
		},
	})

	// then
	require.NoError(t, err)

	_, ok := gotEnvs[kc]
	assert.Equal(t, true, ok)
	delete(gotEnvs, kc)
	assert.Equal(t, map[string]string{
		"HELM_DRIVER":      "secret",
		"HELM_CACHE_HOME":  "/tmp/helm/.cache",
		"HELM_CONFIG_HOME": "/tmp/helm/",
	}, gotEnvs)
}

func TestExecutorConfigMergingErrors(t *testing.T) {
	// given
	hExec := NewExecutor("testing")
	hExec.executeCommandWithEnvs = noopExecuteCommand

	configA := Config{
		HelmDriver: "unknown-value",
	}

	// when
	_, err := hExec.Execute(context.Background(), executor.ExecuteInput{
		Command: "helm install",
		Configs: []*executor.Config{
			{
				RawYAML: mustYAMLMarshal(t, configA),
			},
		},
	})

	// then
	require.EqualError(t, err, "while merging input configs: while validating merged configuration: The unknown-value is invalid. Allowed values are configmap, secret, memory.")
}
func mustYAMLMarshal(t *testing.T, in any) []byte {
	t.Helper()

	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	return out
}

func noopExecuteCommand(_ context.Context, _ string, _ map[string]string) (string, error) {
	return "", nil
}
