package execute

import (
	"testing"

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

	testCases := []struct {
		Name                 string
		InputArgs            []string
		InputNotifierHandler NotifierHandler
		ExpectedResult       string
		ExpectedStatusAfter  string
	}{
		{
			Name:                 "Start",
			InputArgs:            []string{"notifier", "start"},
			InputNotifierHandler: &fakeNotifierHandler{enabled: false},
			ExpectedResult:       `Brace yourselves, incoming notifications from cluster "cluster-name".`,
			ExpectedStatusAfter:  `Notifications from cluster "cluster-name" are enabled.`,
		},
		{
			Name:                 "Stop",
			InputArgs:            []string{"notifier", "stop"},
			InputNotifierHandler: &fakeNotifierHandler{enabled: true},
			ExpectedResult:       `Sure! I won't send you notifications from cluster "cluster-name" anymore.`,
			ExpectedStatusAfter:  `Notifications from cluster "cluster-name" are disabled.`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			e := NewNotifierExecutor(log, config.Config{}, &fakeAnalyticsReporter{})

			// execute command

			// when
			actual, err := e.Do(tc.InputArgs, platform, clusterName, tc.InputNotifierHandler)

			// then
			require.Nil(t, err)
			assert.Equal(t, tc.ExpectedResult, actual)

			// get status after executing a given command

			// when
			actual, err = e.Do(statusArgs, platform, clusterName, tc.InputNotifierHandler)
			// then
			require.Nil(t, err)
			assert.Equal(t, tc.ExpectedStatusAfter, actual)
		})
	}
}

func TestNotifierExecutor_Do_Error(t *testing.T) {
	// given
	log, _ := logtest.NewNullLogger()
	platform := config.SlackCommPlatformIntegration
	clusterName := "cluster-name"

	testCases := []struct {
		Name                 string
		InputArgs            []string
		ExpectedErrorMessage string
	}{
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
			e := NewNotifierExecutor(log, config.Config{}, &fakeAnalyticsReporter{})

			// when
			_, err := e.Do(tc.InputArgs, platform, clusterName, &fakeNotifierHandler{})

			// then
			require.NotNil(t, err)
			assert.EqualError(t, err, tc.ExpectedErrorMessage)
		})
	}
}

type fakeNotifierHandler struct {
	enabled bool
}

func (f *fakeNotifierHandler) Enabled() bool {
	return f.enabled
}

func (f *fakeNotifierHandler) SetEnabled(value bool) error {
	f.enabled = value
	return nil
}

type fakeAnalyticsReporter struct{}

func (f fakeAnalyticsReporter) ReportCommand(_ config.CommPlatformIntegration, _ string) error {
	return nil
}
