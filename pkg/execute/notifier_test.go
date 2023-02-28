package execute

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
)

const (
	channelAlias  = "alias"
	commGroupName = "comm-group"
	clusterName   = "cluster-name"
	testPlatform  = config.SlackCommPlatformIntegration
)

var (
	notifierTestCfg = config.Config{
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
				ExecutorFilter: newExecutorTextFilter(""),
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
				ExecutorFilter: newExecutorTextFilter(""),
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
				ExecutorFilter: newExecutorTextFilter(""),
			},
			ExpectedResult: "",
			ExpectedError:  "while persisting configuration: different alias",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewNotifierExecutor(loggerx.NewNoop(), &fakeCfgPersistenceManager{expectedAlias: channelAlias}, notifierTestCfg)
			msg, err := e.Enable(context.Background(), tc.CmdCtx)
			if err != nil {
				assert.EqualError(t, err, tc.ExpectedError)
				return
			}
			assert.Equal(t, msg.BaseBody.CodeBlock, tc.ExpectedResult)

			msg, err = e.Status(context.Background(), tc.CmdCtx)
			require.NoError(t, err)
			assert.Contains(t, msg.BaseBody.CodeBlock, tc.Status)
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
				ExecutorFilter: newExecutorTextFilter(""),
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
				ExecutorFilter: newExecutorTextFilter(""),
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
				ExecutorFilter: newExecutorTextFilter(""),
			},
			ExpectedResult: "",
			ExpectedError:  "while persisting configuration: different alias",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewNotifierExecutor(loggerx.NewNoop(), &fakeCfgPersistenceManager{expectedAlias: channelAlias}, notifierTestCfg)
			msg, err := e.Disable(context.Background(), tc.CmdCtx)
			if err != nil {
				assert.EqualError(t, err, tc.ExpectedError)
				return
			}
			assert.Equal(t, msg.BaseBody.CodeBlock, tc.ExpectedResult)

			msg, err = e.Status(context.Background(), tc.CmdCtx)
			require.NoError(t, err)
			assert.Contains(t, msg.BaseBody.CodeBlock, tc.Status)
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
				ExecutorFilter: newExecutorTextFilter(""),
			},
			Status: "disabled",
		},
		{
			Name: "Contains help message",
			CmdCtx: CommandContext{
				Args:         []string{"status"},
				Conversation: Conversation{ID: "conv-id"},
				Platform:     testPlatform,
				ClusterName:  clusterName,
				NotifierHandler: &fakeNotifierHandler{
					conf: map[string]bool{"conv-id": false},
				},
				ExecutorFilter: newExecutorTextFilter(""),
			},
			Status: "notifications",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewNotifierExecutor(loggerx.NewNoop(), &fakeCfgPersistenceManager{expectedAlias: channelAlias}, notifierTestCfg)
			mapping, err := NewCmdsMapping([]CommandExecutor{e})
			require.NoError(t, err)
			tc.CmdCtx.Mapping = mapping
			msg, err := e.Status(context.Background(), tc.CmdCtx)
			require.NoError(t, err)
			assert.Contains(t, msg.BaseBody.CodeBlock, tc.Status)
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
