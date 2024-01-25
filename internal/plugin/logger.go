package plugin

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
)

var specialCharsPattern = regexp.MustCompile(`(?i:[^A-Z0-9_])`)

// NewPluginLoggers returns a copy of parent with a log settings dedicated for a given plugin.
// The log level is taken from the environment variable with a pattern: LOG_LEVEL_{pluginType}_{pluginRepo}_{pluginName}.
// If env variable is not set, default value is "info".
// Loggers:
// - hashicorp client logger always has the configured log level
// - binary standard output is logged only if debug level is set, otherwise it is discarded
// - binary standard error is logged always on error level
func NewPluginLoggers(bkLogger logrus.FieldLogger, logConfig config.Logger, pluginKey string, pluginType Type) (hclog.Logger, io.Writer, io.Writer) {
	pluginLogLevel := getPluginLogLevel(bkLogger, pluginKey, pluginType)

	cfg := config.Logger{
		Level:         pluginLogLevel.String(),
		DisableColors: logConfig.DisableColors,
		Formatter:     logConfig.Formatter,
	}
	log := loggerx.New(cfg).WithField("plugin", pluginKey)

	var (
		pluginLogger = loggerx.AsHCLog(log, pluginKey)
		stdoutLogger = io.Discard
		stderrLogger = log.WithField("logger", "stderr").WriterLevel(logrus.ErrorLevel)
	)

	if pluginLogLevel == logrus.DebugLevel {
		stdoutLogger = log.WithField("logger", "stdout").WriterLevel(logrus.DebugLevel)
	}

	return pluginLogger, stdoutLogger, stderrLogger
}

func getPluginLogLevel(logger logrus.FieldLogger, pluginKey string, pluginType Type) logrus.Level {
	repo, name, _ := strings.Cut(pluginKey, "/")
	name = specialCharsPattern.ReplaceAllString(name, "_")

	envName := fmt.Sprintf("LOG_LEVEL_%s_%s_%s", pluginType, repo, name)
	envName = strings.ToUpper(envName)
	logLevel := os.Getenv(envName)

	if logLevel == "" {
		logger.Infof("Explicitly using Info level as custom log level was not set by %q environment variable", envName)
		return logrus.InfoLevel
	}

	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err":           err,
			"givenLogLevel": logLevel,
		}).Info("Explicitly using Info level as we cannot parse specified log level")

		return logrus.InfoLevel
	}

	return lvl
}
