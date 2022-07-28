package execute

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestNotifierExecutor_Do_Success(t *testing.T) {
	// given
	log, _ := logtest.NewNullLogger()
	platform := config.SlackCommPlatformIntegration
	clusterName := "cluster-name"
	statusArgs := []string{"notifier", "status"}
	cfg := config.Config{
		Settings: config.Settings{
			ClusterName: "foo",
		},
	}

	testCases := []struct {
		Name                 string
		InputArgs            []string
		InputNotifierHandler NotifierHandler
		ConversationID       string
		ExpectedResult       string
		ExpectedStatusAfter  string
		ExpectedErrorMessage string
	}{
		{
			Name:           "Start",
			InputArgs:      []string{"notifier", "start"},
			ConversationID: "conv-id",
			InputNotifierHandler: &fakeNotifierHandler{
				conf: map[string]bool{"conv-id": false},
			},
			ExpectedResult:      `Brace yourselves, incoming notifications from cluster 'cluster-name'.`,
			ExpectedStatusAfter: `Notifications from cluster 'cluster-name' are enabled here.`,
		},
		{
			Name:           "Start for non-configured channel",
			InputArgs:      []string{"notifier", "start"},
			ConversationID: "non-existing",
			InputNotifierHandler: &fakeNotifierHandler{
				conf: map[string]bool{"conv-id": false},
			},
			ExpectedResult:      `I'm not configured to send notifications here ('non-existing') from cluster 'cluster-name', so you cannot turn them on or off.`,
			ExpectedStatusAfter: `Notifications from cluster 'cluster-name' are disabled here.`,
		},
		{
			Name:           "Stop",
			ConversationID: "conv-id",
			InputArgs:      []string{"notifier", "stop"},
			InputNotifierHandler: &fakeNotifierHandler{
				conf: map[string]bool{"conv-id": true},
			},
			ExpectedResult:      `Sure! I won't send you notifications from cluster 'cluster-name' here.`,
			ExpectedStatusAfter: `Notifications from cluster 'cluster-name' are disabled here.`,
		},
		{
			Name:           "Stop for non-configured channel",
			ConversationID: "non-existing",
			InputArgs:      []string{"notifier", "stop"},
			InputNotifierHandler: &fakeNotifierHandler{conf: map[string]bool{
				"conv-id": true,
			}},
			ExpectedResult:      `I'm not configured to send notifications here ('non-existing') from cluster 'cluster-name', so you cannot turn them on or off.`,
			ExpectedStatusAfter: `Notifications from cluster 'cluster-name' are disabled here.`,
		},
		{
			Name:                 "Show config",
			ConversationID:       "conv-id",
			InputArgs:            []string{"notifier", "showconfig"},
			InputNotifierHandler: &fakeNotifierHandler{},
			ExpectedResult: heredoc.Doc(`
				Showing config for cluster "cluster-name":

				sources: {}
				executors: {}
				communications: {}
				analytics:
				    installationID: ""
				    disable: false
				settings:
				    clusterName: foo
				    configWatcher: false
				    upgradeNotifier: false
				    metricsPort: ""
				    log:
				        level: ""
				        disableColors: false
				    informersResyncPeriod: 0s
				    kubeconfig: ""
			`),
			ExpectedStatusAfter: `Notifications from cluster 'cluster-name' are disabled here.`,
		},
		{
			Name:                 "Invalid verb",
			InputArgs:            []string{"notifier", "foo"},
			ExpectedErrorMessage: "unsupported command",
		},
		{
			Name:                 "Invalid command 1",
			InputArgs:            []string{"notifier"},
			ExpectedErrorMessage: "invalid notifier command",
		},
		{
			Name:                 "Invalid command 2",
			InputArgs:            []string{"notifier", "stop", "stop", "stop", "please", "stop!!!!1111111oneoneone"},
			ExpectedErrorMessage: "invalid notifier command",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewNotifierExecutor(log, cfg, &fakeAnalyticsReporter{})

			// execute command

			// when
			actual, err := e.Do(tc.InputArgs, platform, tc.ConversationID, clusterName, tc.InputNotifierHandler)

			// then

			if tc.ExpectedErrorMessage != "" {
				// error case
				require.NotNil(t, err)
				assert.EqualError(t, err, tc.ExpectedErrorMessage)
				return
			}

			// success case
			require.Nil(t, err)
			assert.Equal(t, tc.ExpectedResult, actual)

			// get status after executing a given command

			// when
			actual, err = e.Do(statusArgs, platform, tc.ConversationID, clusterName, tc.InputNotifierHandler)
			// then
			require.Nil(t, err)
			assert.Equal(t, tc.ExpectedStatusAfter, actual)
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

type fakeAnalyticsReporter struct{}

func (f fakeAnalyticsReporter) ReportCommand(_ config.CommPlatformIntegration, _ string) error {
	return nil
}
