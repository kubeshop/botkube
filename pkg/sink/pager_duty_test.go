package sink

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
)

func Test(t *testing.T) {
	duty, err := NewPagerDuty(loggerx.New(config.Logger{
		Level:     "debug",
		Formatter: "text",
	}), 0, config.PagerDuty{
		Enabled:        true,
		IntegrationKey: "R03E2UWJRUG6IYKSWEZUUFCPUFRXAK4E",
		//AlertAPIURL:  "",
		//ChangeAPIURL: "",
		Bindings: config.SinkBindings{
			Sources: []string{"kubernetes-err"},
		},
	}, "labs", analytics.NewNoopReporter())

	require.NoError(t, err)

	err = duty.SendEvent(context.Background(), map[string]any{
		"APIVersion":      "v1",
		"Kind":            "Pod",
		"Title":           "v1/pods error",
		"Name":            "webapp",
		"Namespace":       "dev",
		"Messages":        []string{"Back-off restarting failed container webapp in pod webapp_dev(0a405592-2615-4d0c-b399-52ada5a9cc1b)"},
		"Type":            "error",
		"Reason":          "BackOff",
		"Level":           "error",
		"Cluster":         "labs",
		"TimeStamp":       time.Date(2024, 5, 14, 19, 32, 42, 0, time.FixedZone("+09:00", 9*60*60)),
		"Count":           int32(1),
		"Action":          "",
		"Skip":            false,
		"Resource":        "v1/pods",
		"Recommendations": []string{},
		"Warnings":        []string{},
	}, []string{"kubernetes-err"})
	require.NoError(t, err)
}
