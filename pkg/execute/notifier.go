package execute

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

const (
	notifierStartMsgFmt                = "Brace yourselves, incoming notifications from cluster '%s'."
	notifierStopMsgFmt                 = "Sure! I won't send you notifications from cluster '%s' here."
	notifierStatusMsgFmt               = "Notifications from cluster '%s' are %s here."
	notifierNotConfiguredMsgFmt        = "I'm not configured to send notifications here ('%s') from cluster '%s', so you cannot turn them on or off."
	notifierPersistenceNotSupportedFmt = "Platform %q doesn't support persistence for notifications. When Botkube Pod restarts, default notification settings will be applied for this platform."
)

var (
	notifierStatusStrings = map[bool]string{
		true:  "enabled",
		false: "disabled",
	}
)

// NotifierHandler handles disabling and enabling notifications for a given communication platform.
type NotifierHandler interface {
	// NotificationsEnabled returns current notification status for a given conversation ID.
	NotificationsEnabled(conversationID string) bool

	// SetNotificationsEnabled sets a new notification status for a given conversation ID.
	SetNotificationsEnabled(conversationID string, enabled bool) error

	BotName() string
}

var (
	// ErrNotificationsNotConfigured describes an error when user wants to toggle on/off the notifications for not configured channel.
	ErrNotificationsNotConfigured = errors.New("notifications not configured for this channel")
	notifierResourcesNames        = []string{"notification", "notifications", "notif", ""}
)

// NotifierExecutor executes all commands that are related to notifications.
type NotifierExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cfgManager        ConfigPersistenceManager

	// Used for deprecated showControllerConfig function.
	cfg config.Config
}

// NewNotifierExecutor creates a new instance of NotifierExecutor
func NewNotifierExecutor(log logrus.FieldLogger, cfg config.Config, cfgManager ConfigPersistenceManager, analyticsReporter AnalyticsReporter) *NotifierExecutor {
	return &NotifierExecutor{
		log:               log,
		cfg:               cfg,
		cfgManager:        cfgManager,
		analyticsReporter: analyticsReporter,
	}
}

// ResourceNames returns slice of resources the executor supports
func (e *NotifierExecutor) ResourceNames() []string {
	return notifierResourcesNames
}

// Commands returns slice of commands the executor supports
func (e *NotifierExecutor) Commands() map[CommandVerb]CommandFn {
	return map[CommandVerb]CommandFn{
		CommandStart:      e.Start,
		CommandStop:       e.Stop,
		CommandStatus:     e.Status,
		CommandShowConfig: e.ShowConfig,
	}
}

// Start starts the notifier
func (e *NotifierExecutor) Start(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, cmdRes := cmdCtx.Args[0], cmdCtx.Args[1]
	defer e.reportCommand(fmt.Sprintf("%s %s", cmdVerb, cmdRes), cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)

	const enabled = true
	err := cmdCtx.NotifierHandler.SetNotificationsEnabled(cmdCtx.Conversation.ID, enabled)
	if err != nil {
		if errors.Is(err, ErrNotificationsNotConfigured) {
			msg := fmt.Sprintf(notifierNotConfiguredMsgFmt, cmdCtx.Conversation.ID, cmdCtx.ClusterName)
			return respond(msg, cmdCtx), nil
		}
		return interactive.Message{}, fmt.Errorf("while setting notifications to %t: %w", enabled, err)
	}
	successMessage := fmt.Sprintf(notifierStartMsgFmt, cmdCtx.ClusterName)
	err = e.cfgManager.PersistNotificationsEnabled(ctx, cmdCtx.CommGroupName, cmdCtx.Platform, cmdCtx.Conversation.Alias, enabled)
	if err != nil {
		if err == config.ErrUnsupportedPlatform {
			e.log.Warnf(notifierPersistenceNotSupportedFmt, cmdCtx.Platform)
			return respond(successMessage, cmdCtx), nil
		}
		return interactive.Message{}, fmt.Errorf("while persisting configuration: %w", err)
	}
	return respond(successMessage, cmdCtx), nil
}

// Stop stops the notifier
func (e *NotifierExecutor) Stop(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, cmdRes := cmdCtx.Args[0], cmdCtx.Args[1]
	defer e.reportCommand(fmt.Sprintf("%s %s", cmdVerb, cmdRes), cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)

	const enabled = false
	err := cmdCtx.NotifierHandler.SetNotificationsEnabled(cmdCtx.Conversation.ID, enabled)
	if err != nil {
		if errors.Is(err, ErrNotificationsNotConfigured) {
			msg := fmt.Sprintf(notifierNotConfiguredMsgFmt, cmdCtx.Conversation.ID, cmdCtx.ClusterName)
			return respond(msg, cmdCtx), nil
		}
		return interactive.Message{}, fmt.Errorf("while setting notifications to %t: %w", enabled, err)
	}
	successMessage := fmt.Sprintf(notifierStopMsgFmt, cmdCtx.ClusterName)
	err = e.cfgManager.PersistNotificationsEnabled(ctx, cmdCtx.CommGroupName, cmdCtx.Platform, cmdCtx.Conversation.Alias, enabled)
	if err != nil {
		if err == config.ErrUnsupportedPlatform {
			e.log.Warnf(notifierPersistenceNotSupportedFmt, cmdCtx.Platform)
			return respond(successMessage, cmdCtx), nil
		}
		return interactive.Message{}, fmt.Errorf("while persisting configuration: %w", err)
	}
	return respond(successMessage, cmdCtx), nil
}

// Status returns the status of a notifier (per channel)
func (e *NotifierExecutor) Status(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb := cmdCtx.Args[0]
	defer e.reportCommand(cmdVerb, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)

	enabled := cmdCtx.NotifierHandler.NotificationsEnabled(cmdCtx.Conversation.ID)
	enabledStr := notifierStatusStrings[enabled]
	msg := fmt.Sprintf(notifierStatusMsgFmt, cmdCtx.ClusterName, enabledStr)
	return respond(msg, cmdCtx), nil
}

// ShowConfig returns Config in yaml format
func (e *NotifierExecutor) ShowConfig(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb := cmdCtx.Args[0]
	defer e.reportCommand(cmdVerb, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)

	out, err := e.showControllerConfig()
	if err != nil {
		return interactive.Message{}, fmt.Errorf("while executing 'showconfig' command: %w", err)
	}
	msg := fmt.Sprintf("Showing config for cluster %q:\n\n%s", cmdCtx.ClusterName, out)
	return respond(msg, cmdCtx), nil
}

func (e *NotifierExecutor) reportCommand(cmdToReport string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting edit command: %s", err.Error())
	}
}

const redactedSecretStr = "*** REDACTED ***"

// Deprecated: this function doesn't fit in the scope of notifier. It was moved from legacy reasons, but it will be removed in future.
func (e *NotifierExecutor) showControllerConfig() (string, error) {
	cfg := e.cfg

	// hide sensitive info
	// TODO: avoid printing sensitive data without need to resetting them manually (which is an error-prone approach)
	for key, old := range cfg.Communications {
		old.Slack.Token = redactedSecretStr
		old.SocketSlack.AppToken = redactedSecretStr
		old.SocketSlack.BotToken = redactedSecretStr
		old.Elasticsearch.Password = redactedSecretStr
		old.Discord.Token = redactedSecretStr
		old.Mattermost.Token = redactedSecretStr
		old.Teams.AppPassword = redactedSecretStr

		// maps are not addressable: https://stackoverflow.com/questions/42605337/cannot-assign-to-struct-field-in-a-map
		cfg.Communications[key] = old
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
