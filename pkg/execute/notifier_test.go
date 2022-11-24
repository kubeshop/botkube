package execute

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
)

const (
	channelAlias  = "alias"
	commGroupName = "comm-group"
	clusterName   = "cluster-name"
	testPlatform  = config.SlackCommPlatformIntegration
)

var (
	log, _ = logtest.NewNullLogger()
	cfg    = config.Config{
		Settings: config.Settings{
			ClusterName: "foo",
		},
	}
)

func TestNotifierExecutorStart(t *testing.T) {
	testCases := []struct {
		Name           string
		CmdCtx         CommandContext
		ExpectedResult string
		ExpectedError  string
		Status         string
	}{
		{
			Name: "Existing channel",
			CmdCtx: CommandContext{
				ClusterName:   clusterName,
				Args:          []string{"start", "notifications"},
				CommGroupName: commGroupName,
				Platform:      testPlatform,
				Conversation:  Conversation{Alias: channelAlias, ID: "conv-id"},
				NotifierHandler: &fakeNotifierHandler{
					conf: map[string]bool{"conv-id": false},
				},
			},
			ExpectedResult: `Brace yourselves, incoming notifications from cluster 'cluster-name'.`,
			Status:         "enabled",
			ExpectedError:  "",
		},
		{
			Name: "Expecting failure: channel wrong ID",
			CmdCtx: CommandContext{
				ClusterName:   clusterName,
				Args:          []string{"start", "notifications"},
				CommGroupName: commGroupName,
				Platform:      testPlatform,
				Conversation:  Conversation{Alias: channelAlias, ID: "bogus"},
				NotifierHandler: &fakeNotifierHandler{
					conf: map[string]bool{"conv-id": false},
				},
			},
			ExpectedResult: `I'm not configured to send notifications here ('bogus') from cluster 'cluster-name', so you cannot turn them on or off.`,
		},
		{
			Name: "Expecting failure: channel wrong alias",
			CmdCtx: CommandContext{
				ClusterName:   clusterName,
				Args:          []string{"start", "notifications"},
				CommGroupName: commGroupName,
				Platform:      testPlatform,
				Conversation:  Conversation{Alias: "bogus", ID: "conv-id"},
				NotifierHandler: &fakeNotifierHandler{
					conf: map[string]bool{"conv-id": false},
				},
			},
			ExpectedResult: "",
			ExpectedError:  "while persisting configuration: different alias",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewNotifierExecutor(log, cfg, &fakeCfgPersistenceManager{expectedAlias: channelAlias}, &fakeAnalyticsReporter{})
			msg, err := e.Start(context.TODO(), tc.CmdCtx)
			if err != nil {
				assert.EqualError(t, err, tc.ExpectedError)
				return
			}
			assert.Equal(t, msg.Body.CodeBlock, tc.ExpectedResult)

			msg, err = e.Status(context.TODO(), tc.CmdCtx)
			require.NoError(t, err)
			assert.Contains(t, msg.Body.CodeBlock, tc.Status)
		})
	}
}

func TestNotifierExecutorStop(t *testing.T) {
	testCases := []struct {
		Name           string
		CmdCtx         CommandContext
		ExpectedResult string
		ExpectedError  string
		Status         string
	}{
		{
			Name: "Existing channel",
			CmdCtx: CommandContext{
				ClusterName:   clusterName,
				Args:          []string{"stop", "notifications"},
				CommGroupName: commGroupName,
				Platform:      testPlatform,
				Conversation:  Conversation{Alias: channelAlias, ID: "conv-id"},
				NotifierHandler: &fakeNotifierHandler{
					conf: map[string]bool{"conv-id": true},
				},
			},
			ExpectedResult: `Sure! I won't send you notifications from cluster 'cluster-name' here.`,
			Status:         "disabled",
			ExpectedError:  "",
		},
		{
			Name: "Expecting failure: channel wrong ID",
			CmdCtx: CommandContext{
				ClusterName:   clusterName,
				Args:          []string{"stop", "notifications"},
				CommGroupName: commGroupName,
				Platform:      testPlatform,
				Conversation:  Conversation{Alias: channelAlias, ID: "bogus"},
				NotifierHandler: &fakeNotifierHandler{
					conf: map[string]bool{"conv-id": false},
				},
			},
			ExpectedResult: `I'm not configured to send notifications here ('bogus') from cluster 'cluster-name', so you cannot turn them on or off.`,
		},
		{
			Name: "Expecting failure: channel wrong alias",
			CmdCtx: CommandContext{
				ClusterName:   clusterName,
				Args:          []string{"stop", "notifications"},
				CommGroupName: commGroupName,
				Platform:      testPlatform,
				Conversation:  Conversation{Alias: "bogus", ID: "conv-id"},
				NotifierHandler: &fakeNotifierHandler{
					conf: map[string]bool{"conv-id": false},
				},
			},
			ExpectedResult: "",
			ExpectedError:  "while persisting configuration: different alias",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewNotifierExecutor(log, cfg, &fakeCfgPersistenceManager{expectedAlias: channelAlias}, &fakeAnalyticsReporter{})
			msg, err := e.Stop(context.TODO(), tc.CmdCtx)
			if err != nil {
				assert.EqualError(t, err, tc.ExpectedError)
				return
			}
			assert.Equal(t, msg.Body.CodeBlock, tc.ExpectedResult)

			msg, err = e.Status(context.TODO(), tc.CmdCtx)
			require.NoError(t, err)
			assert.Contains(t, msg.Body.CodeBlock, tc.Status)
		})
	}
}

// Notifier is mocked here - most of the logic needs to be tested in respective implementations
func TestNotifierExecutorStatus(t *testing.T) {
	testCases := []struct {
		Name   string
		CmdCtx CommandContext
		Status string
	}{
		{
			Name: "Is disabled",
			CmdCtx: CommandContext{
				Args:         []string{"status"},
				Conversation: Conversation{ID: "conv-id"},
				Platform:     testPlatform,
				ClusterName:  clusterName,
				NotifierHandler: &fakeNotifierHandler{
					conf: map[string]bool{"conv-id": false},
				},
			},
			Status: "disabled",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewNotifierExecutor(log, cfg, &fakeCfgPersistenceManager{expectedAlias: channelAlias}, &fakeAnalyticsReporter{})
			msg, err := e.Status(context.TODO(), tc.CmdCtx)
			require.NoError(t, err)
			assert.Contains(t, msg.Body.CodeBlock, tc.Status)
		})
	}
}

func TestNotifierExecutorShowConfig(t *testing.T) {
	testCases := []struct {
		Name           string
		CmdCtx         CommandContext
		ExpectedResult string
	}{
		{
			Name: "Is disabled",
			CmdCtx: CommandContext{
				Args:            []string{"config"},
				Conversation:    Conversation{Alias: channelAlias, ID: "conv-id"},
				Platform:        testPlatform,
				ClusterName:     clusterName,
				NotifierHandler: &fakeNotifierHandler{},
				ExecutorFilter:  newExecutorTextFilter("", "config"),
			},
			ExpectedResult: heredoc.Doc(`
						Showing config for cluster "cluster-name":

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
						    repositories: {}
					`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewNotifierExecutor(log, cfg, &fakeCfgPersistenceManager{expectedAlias: channelAlias}, &fakeAnalyticsReporter{})
			msg, err := e.ShowConfig(context.TODO(), tc.CmdCtx)
			require.NoError(t, err)
			assert.Contains(t, msg.Body.CodeBlock, tc.ExpectedResult)
		})
	}
}

type fakeNotifierHandler struct {
	conf map[string]bool
}

func (f *fakeNotifierHandler) NotificationsEnabled(convID string) bool {
	enabled, exists := f.conf[convID]
	if !exists {
		return false
	}

	return enabled
}

func (f *fakeNotifierHandler) SetNotificationsEnabled(convID string, enabled bool) error {
	_, exists := f.conf[convID]
	if !exists {
		return ErrNotificationsNotConfigured
	}

	f.conf[convID] = enabled
	return nil
}

func (f *fakeNotifierHandler) BotName() string {
	return "fake"
}
