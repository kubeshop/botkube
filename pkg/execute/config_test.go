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
				ExecutorFilter: newExecutorTextFilter("", "config"),
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
						communications: {}
						filters:
						    kubernetes:
						        objectAnnotationChecker: false
						        nodeEventsChecker: false
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
						    lifecycleServer:
						        enabled: false
						        port: 0
						        deployment: {}
						    log:
						        level: ""
						        disableColors: false
						    informersResyncPeriod: 0s
						    kubeconfig: ""
						configWatcher:
						    enabled: false
						    initialSyncTimeout: 0s
						    tmpDir: ""
						plugins:
						    cacheDir: ""
						    repositories: {}`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewConfigExecutor(loggerx.NewNoop(), &fakeAnalyticsReporter{}, tc.Cfg)
			msg, err := e.Config(context.Background(), tc.CmdCtx)
			require.NoError(t, err)
			assert.Equal(t, msg.Body.CodeBlock, tc.ExpectedResult)
		})
	}
}
