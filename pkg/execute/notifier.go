package execute

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

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
}

var (
	// ErrNotificationsNotConfigured describes an error when user wants to toggle on/off the notifications for not configured channel.
	ErrNotificationsNotConfigured = errors.New("notifications not configured for this channel")
	notifierFeatureName           = FeatureName{
		Name:    "notification",
		Aliases: []string{"notifications", "notif", ""},
	}
)

// NotifierExecutor executes all commands that are related to notifications.
type NotifierExecutor struct {
	log        logrus.FieldLogger
	cfgManager ConfigPersistenceManager
}

// NewNotifierExecutor creates a new instance of NotifierExecutor
func NewNotifierExecutor(log logrus.FieldLogger, cfgManager ConfigPersistenceManager, cfg config.Config) *NotifierExecutor {
	return &NotifierExecutor{
		log:        log,
		cfgManager: cfgManager,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *NotifierExecutor) FeatureName() FeatureName {
	return notifierFeatureName
}

// Commands returns slice of commands the executor supports
func (e *NotifierExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.EnableVerb:  e.Enable,
		command.DisableVerb: e.Disable,
		command.StatusVerb:  e.Status,
	}
}

// Enable starts the notifier
func (e *NotifierExecutor) Enable(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	const enabled = true
	err := cmdCtx.NotifierHandler.SetNotificationsEnabled(cmdCtx.Conversation.ID, enabled)
	if err != nil {
		if errors.Is(err, ErrNotificationsNotConfigured) {
			msg := fmt.Sprintf(notifierNotConfiguredMsgFmt, cmdCtx.Conversation.ID, cmdCtx.ClusterName)
			return respond(msg, cmdCtx), nil
		}
		return interactive.CoreMessage{}, fmt.Errorf("while setting notifications to %t: %w", enabled, err)
	}
	successMessage := fmt.Sprintf(notifierStartMsgFmt, cmdCtx.ClusterName)
	err = e.cfgManager.PersistNotificationsEnabled(ctx, cmdCtx.CommGroupName, cmdCtx.Platform, cmdCtx.Conversation.Alias, enabled)
	if err != nil {
		if err == config.ErrUnsupportedPlatform {
			e.log.Warnf(notifierPersistenceNotSupportedFmt, cmdCtx.Platform)
			return respond(successMessage, cmdCtx), nil
		}
		return interactive.CoreMessage{}, fmt.Errorf("while persisting configuration: %w", err)
	}
	return respond(successMessage, cmdCtx), nil
}

// Disable stops the notifier
func (e *NotifierExecutor) Disable(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	const enabled = false
	err := cmdCtx.NotifierHandler.SetNotificationsEnabled(cmdCtx.Conversation.ID, enabled)
	if err != nil {
		if errors.Is(err, ErrNotificationsNotConfigured) {
			msg := fmt.Sprintf(notifierNotConfiguredMsgFmt, cmdCtx.Conversation.ID, cmdCtx.ClusterName)
			return respond(msg, cmdCtx), nil
		}
		return interactive.CoreMessage{}, fmt.Errorf("while setting notifications to %t: %w", enabled, err)
	}
	successMessage := fmt.Sprintf(notifierStopMsgFmt, cmdCtx.ClusterName)
	err = e.cfgManager.PersistNotificationsEnabled(ctx, cmdCtx.CommGroupName, cmdCtx.Platform, cmdCtx.Conversation.Alias, enabled)
	if err != nil {
		if err == config.ErrUnsupportedPlatform {
			e.log.Warnf(notifierPersistenceNotSupportedFmt, cmdCtx.Platform)
			return respond(successMessage, cmdCtx), nil
		}
		return interactive.CoreMessage{}, fmt.Errorf("while persisting configuration: %w", err)
	}
	return respond(successMessage, cmdCtx), nil
}

// Status returns the status of a notifier (per channel)
func (e *NotifierExecutor) Status(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	cmdVerb, cmdRes := parseCmdVerb(cmdCtx.Args)
	enabled := cmdCtx.NotifierHandler.NotificationsEnabled(cmdCtx.Conversation.ID)
	enabledStr := notifierStatusStrings[enabled]
	msg := fmt.Sprintf(notifierStatusMsgFmt, cmdCtx.ClusterName, enabledStr)
	if cmdRes == "" {
		helpMsg := cmdCtx.Mapping.HelpMessageForVerb(command.Verb(cmdVerb))
		msg = fmt.Sprintf("%s\n\n%s\n", msg, helpMsg)
	}
	return respond(msg, cmdCtx), nil
}
