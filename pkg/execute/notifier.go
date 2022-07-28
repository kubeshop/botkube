package execute

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/config"
)

const (
	notifierStartMsgFmt  = "Brace yourselves, incoming notifications from cluster %q."
	notifierStopMsgFmt   = "Sure! I won't send you notifications from cluster %q anymore."
	notifierStatusMsgFmt = "Notifications from cluster %q are %s."
)

// NotifierHandler handles disabling and enabling notifications for a given communication platform.
type NotifierHandler interface {
	// NotificationsEnabled returns current notification status.
	NotificationsEnabled() bool

	// SetNotificationsEnabled sets a new notification status.
	SetNotificationsEnabled(enabled bool) error
}

var (
	errInvalidNotifierCommand = errors.New("invalid notifier command")
	errUnsupportedCommand     = errors.New("unsupported command")
)

// NotifierExecutor executes all commands that are related to notifications.
type NotifierExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter

	// Used for deprecated showControllerConfig function.
	cfg config.Config
}

// NewNotifierExecutor creates a new instance of NotifierExecutor.
func NewNotifierExecutor(log logrus.FieldLogger, cfg config.Config, analyticsReporter AnalyticsReporter) *NotifierExecutor {
	return &NotifierExecutor{log: log, cfg: cfg, analyticsReporter: analyticsReporter}
}

// Do executes a given Notifier command based on args.
func (e *NotifierExecutor) Do(args []string, platform config.CommPlatformIntegration, clusterName string, handler NotifierHandler) (string, error) {
	if len(args) != 2 {
		return "", errInvalidNotifierCommand
	}

	var cmdVerb = args[1]
	var isUnknownVerb bool
	defer func() {
		if isUnknownVerb {
			cmdVerb = anonymizedInvalidVerb // prevent passing any personal information
		}
		cmdToReport := fmt.Sprintf("%s %s", args[0], cmdVerb)
		err := e.analyticsReporter.ReportCommand(platform, cmdToReport)
		if err != nil {
			// TODO: Return error when the DefaultExecutor is refactored as a part of https://github.com/kubeshop/botkube/issues/589
			e.log.Errorf("while reporting notifier command: %s", err.Error())
		}
	}()

	switch NotifierAction(cmdVerb) {
	case Start:
		err := handler.SetNotificationsEnabled(true)
		if err != nil {
			return "", fmt.Errorf("while setting notifications to true: %w", err)
		}

		return fmt.Sprintf(notifierStartMsgFmt, clusterName), nil
	case Stop:
		err := handler.SetNotificationsEnabled(false)
		if err != nil {
			return "", fmt.Errorf("while setting notifications to false: %w", err)
		}

		return fmt.Sprintf(notifierStopMsgFmt, clusterName), nil
	case Status:
		enabled := handler.NotificationsEnabled()

		enabledStr := "enabled"
		if !enabled {
			enabledStr = "disabled"
		}

		return fmt.Sprintf(notifierStatusMsgFmt, clusterName, enabledStr), nil
	case ShowConfig:
		out, err := e.showControllerConfig()
		if err != nil {
			return "", fmt.Errorf("while executing 'showconfig' command: %w", err)
		}

		return fmt.Sprintf("Showing config for cluster %q:\n\n%s", clusterName, out), nil
	default:
		isUnknownVerb = true
	}

	return "", errUnsupportedCommand
}

const redactedSecretStr = "*** REDACTED ***"

// Deprecated: this function doesn't fit in the scope of notifier. It was moved from legacy reasons, but it will be removed in future.
func (e *NotifierExecutor) showControllerConfig() (string, error) {
	cfg := e.cfg

	// hide sensitive info
	// TODO: avoid printing sensitive data without need to resetting them manually (which is an error-prone approach)
	for key, old := range cfg.Communications {
		old.Slack.Token = redactedSecretStr
		old.Elasticsearch.Password = redactedSecretStr
		old.Discord.Token = redactedSecretStr
		old.Mattermost.Token = redactedSecretStr

		// maps are not addressable: https://stackoverflow.com/questions/42605337/cannot-assign-to-struct-field-in-a-map
		cfg.Communications[key] = old
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
