package execute

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
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
				Platform:       config.SlackCommPlatformIntegration,
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
						    informersResyncPeriod: 0s
						    kubeconfig: ""
						configWatcher:
						    enabled: false
						    remote:
						        pollInterval: 0s
						    initialSyncTimeout: 0s
						    tmpDir: ""
						    deployment: {}
						plugins:
						    cacheDir: ""
						    repositories: {}
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
