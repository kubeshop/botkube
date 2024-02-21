package execute

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
)

const (
	configTestClusterName = "foo"
)

func TestConfigExecutorShowConfig(t *testing.T) {
	testCases := []struct {
		Name           string
		CmdCtx         CommandContext
		Cfg            config.Config
		ExpectedResult string
	}{
		{
			Name: "Print config",
			CmdCtx: CommandContext{
				Args:           []string{"config"},
				Conversation:   Conversation{Alias: channelAlias, ID: "conv-id"},
				Platform:       config.SocketSlackCommPlatformIntegration,
				ClusterName:    configTestClusterName,
				ExecutorFilter: newExecutorTextFilter(""),
			},
			Cfg: config.Config{
				Settings: config.Settings{
					ClusterName: configTestClusterName,
				},
			},
			ExpectedResult: heredoc.Doc(`
						actions: {}
						sources: {}
						executors: {}
						aliases: {}
						communications: {}
						analytics:
						    disable: false
						settings:
						    clusterName: foo
						    upgradeNotifier: false
						    systemConfigMap: {}
						    persistentConfig:
						        startup:
						            fileName: ""
						            configMap: {}
						        runtime:
						            fileName: ""
						            configMap: {}
						    metricsPort: ""
						    healthPort: ""
						    lifecycleServer:
						        enabled: false
						        port: 0
						    log:
						        level: ""
						        disableColors: false
						        formatter: ""
						    informersResyncPeriod: 0s
						    kubeconfig: ""
						    saCredentialsPathPrefix: ""
						configWatcher:
						    enabled: false
						    remote:
						        pollInterval: 0s
						    inCluster:
						        informerResyncPeriod: 0s
						    deployment: {}
						plugins:
						    cacheDir: ""
						    repositories: {}
						    incomingWebhook:
						        enabled: false
						        port: 0
						        inClusterBaseURL: ""
						    restartPolicy:
						        type: ""
						        threshold: 0
						    healthCheckInterval: 0s
						`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewConfigExecutor(loggerx.NewNoop(), tc.Cfg)
			msg, err := e.Show(context.Background(), tc.CmdCtx)
			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedResult, msg.BaseBody.CodeBlock)
		})
	}
}
