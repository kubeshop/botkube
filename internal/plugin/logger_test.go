package plugin

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/internal/loggerx"
)

func TestGetPluginLogLevel(t *testing.T) {
	const (
		pluginKey  = "botkube/kubectl"
		pluginType = TypeExecutor
	)
	tests := []struct {
		name        string
		logLevel    logrus.Level
		envVariable string
	}{
		{
			name:        "should use default log level when env is not exported",
			envVariable: "FOO=bar",
			logLevel:    logrus.InfoLevel,
		},
		{
			name:        "should use warning log level as defined by env variable",
			envVariable: "LOG_LEVEL_EXECUTOR_BOTKUBE_KUBECTL=warn",
			logLevel:    logrus.WarnLevel,
		},
		{
			name:        "should use debug log level as defined by env variable",
			envVariable: "LOG_LEVEL_EXECUTOR_BOTKUBE_KUBECTL=debug",
			logLevel:    logrus.DebugLevel,
		},
		{
			name:        "should use error log level as defined by env variable",
			envVariable: "LOG_LEVEL_EXECUTOR_BOTKUBE_KUBECTL=error",
			logLevel:    logrus.ErrorLevel,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			name, val, _ := strings.Cut(tc.envVariable, "=")
			t.Setenv(name, val)

			// when
			gotLvl := getPluginLogLevel(loggerx.NewNoop(), pluginKey, pluginType)

			// then
			assert.Equal(t, tc.logLevel.String(), gotLvl.String())
		})
	}
}

func TestGetPluginLogLevelSpecialCharacters(t *testing.T) {
	tests := []struct {
		name        string
		logLevel    logrus.Level
		pluginKey   string
		envVariable string
	}{
		{
			name:        "should replace dash with underscore",
			pluginKey:   "botkube/cm-watcher",
			envVariable: "LOG_LEVEL_SOURCE_BOTKUBE_CM_WATCHER=warn",
			logLevel:    logrus.WarnLevel,
		},

		{
			name:        "should replace dot with underscore",
			pluginKey:   "botkube/cm.watcher",
			envVariable: "LOG_LEVEL_SOURCE_BOTKUBE_CM_WATCHER=warn",
			logLevel:    logrus.WarnLevel,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			name, val, _ := strings.Cut(tc.envVariable, "=")
			t.Setenv(name, val)

			// when
			gotLvl := getPluginLogLevel(loggerx.NewNoop(), tc.pluginKey, TypeSource)

			// then
			assert.Equal(t, tc.logLevel.String(), gotLvl.String())
		})
	}
}
