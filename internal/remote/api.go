package remote

import (
	"fmt"
	"strings"
)

// AuditEventCreateInput contains generic create input
type AuditEventCreateInput struct {
	Type               AuditEventType                `json:"type"`
	CreatedAt          string                        `json:"createdAt"`
	DeploymentID       string                        `json:"deploymentId"`
	PluginName         string                        `json:"pluginName"`
	SourceEventEmitted *AuditEventSourceCreateInput  `json:"sourceEventEmitted"`
	CommandExecuted    *AuditEventCommandCreateInput `json:"commandExecuted"`
}

// AuditEventCommandCreateInput contains create input specific to executor events
type AuditEventCommandCreateInput struct {
	PlatformUser string      `json:"platformUser"`
	Channel      string      `json:"channel"`
	BotPlatform  BotPlatform `json:"botPlatform"`
	Command      string      `json:"command"`
}

// AuditEventSourceCreateInput contains create input specific to source events
type AuditEventSourceCreateInput struct {
	Event    string   `json:"event"`
	Bindings []string `json:"bindings"`
}

// BotPlatform are the supported bot platforms
type BotPlatform string

const (
	// BotPlatformSlack is the slack platform
	BotPlatformSlack BotPlatform = "SLACK"
	// BotPlatformDiscord is the discord platform
	BotPlatformDiscord BotPlatform = "DISCORD"
	// BotPlatformMattermost is the mattermost platform
	BotPlatformMattermost BotPlatform = "MATTERMOST"
	// BotPlatformMsTeams is the teams platform
	BotPlatformMsTeams BotPlatform = "MS_TEAMS"
)

// NewBotPlatform creates new BotPlatform from string
func NewBotPlatform(s string) (BotPlatform, error) {
	switch strings.ToUpper(s) {
	case "SLACK":
		fallthrough
	case "SOCKETSLACK":
		return BotPlatformSlack, nil
	case "DISCORD":
		return BotPlatformDiscord, nil
	case "MATTERMOST":
		return BotPlatformMattermost, nil
	case "TEAMS":
		fallthrough
	case "MS_TEAMS":
		return BotPlatformMsTeams, nil
	default:
		return "", fmt.Errorf("given BotPlatform %s is not supported", s)
	}
}

// AuditEventType is the type of audit events
type AuditEventType string

const (
	// AuditEventTypeCommandExecuted is the executor audit event type
	AuditEventTypeCommandExecuted AuditEventType = "COMMAND_EXECUTED"
	// AuditEventTypeSourceEventEmitted is the source audit event type
	AuditEventTypeSourceEventEmitted AuditEventType = "SOURCE_EVENT_EMITTED"
)

// PatchDeploymentConfigInput contains patch input specific to deployments
type PatchDeploymentConfigInput struct {
	ResourceVersion int                                      `json:"resourceVersion"`
	Notification    *NotificationPatchDeploymentConfigInput  `json:"notification"`
	SourceBinding   *SourceBindingPatchDeploymentConfigInput `json:"sourceBinding"`
}

// NotificationPatchDeploymentConfigInput contains patch input specific to notifications
type NotificationPatchDeploymentConfigInput struct {
	CommunicationGroupName string      `json:"communicationGroupName"`
	Platform               BotPlatform `json:"platform"`
	ChannelAlias           string      `json:"channelAlias"`
	Disabled               bool        `json:"disabled"`
}

// SourceBindingPatchDeploymentConfigInput contains patch input specific to source bindings
type SourceBindingPatchDeploymentConfigInput struct {
	CommunicationGroupName string      `json:"communicationGroupName"`
	Platform               BotPlatform `json:"platform"`
	ChannelAlias           string      `json:"channelAlias"`
	SourceBindings         []string    `json:"sourceBindings"`
}
