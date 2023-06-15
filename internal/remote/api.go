package remote

import (
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
	PlatformUser string       `json:"platformUser"`
	Channel      string       `json:"channel"`
	BotPlatform  *BotPlatform `json:"botPlatform"`
	Command      string       `json:"command"`
}

// AuditEventSourceCreateInput contains create input specific to source events
type AuditEventSourceCreateInput struct {
	Event  string                       `json:"event"`
	Source AuditEventSourceDetailsInput `json:"source"`
}

type AuditEventSourceDetailsInput struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
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
	// BotPlatformUnknown is the unknown platform
	BotPlatformUnknown BotPlatform = "UNKNOWN"
)

// NewBotPlatform creates new BotPlatform from string
func NewBotPlatform(s string) *BotPlatform {
	var platform BotPlatform
	switch strings.ToUpper(s) {
	case "SLACK", "CLOUDSLACK", "SOCKETSLACK":
		platform = BotPlatformSlack
	case "DISCORD":
		platform = BotPlatformDiscord
	case "MATTERMOST":
		platform = BotPlatformMattermost
	case "TEAMS":
		fallthrough
	case "MS_TEAMS":
		platform = BotPlatformMsTeams
	default:
		platform = BotPlatformUnknown
	}

	return &platform
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
	Action          *ActionPatchDeploymentConfigInput        `json:"action"`
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

// ActionPatchDeploymentConfigInput contains action's updatable fields.
type ActionPatchDeploymentConfigInput struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
}

// DeploymentFailureInput represents the input data structure for reporting a deployment failure.
type DeploymentFailureInput struct {
	// ResourceVersion is the deployment version that we want to alter.
	ResourceVersion int `json:"resourceVersion"`
	// Message is a human-readable message describing the deployment failure.
	Message string `json:"message"`
}
