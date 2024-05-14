package sink

import (
	"context"
	"testing"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/stretchr/testify/require"
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
		//Bindings:     config.SinkBindings{},
	}, "", analytics.NewNoopReporter())

	require.NoError(t, err)

	err = duty.SendEvent(context.Background(), map[string]any{
		"name": "pod",
	}, []string{"kubernetes-err"})
	require.NoError(t, err)

}
