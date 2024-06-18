//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	"strings"

	gqlModel "github.com/kubeshop/botkube-cloud/botkube-cloud-backend/pkg/graphql"
)

// AuditEvent represents audit event.
type AuditEvent struct {
	CommandExecutedEvent    `graphql:"... on CommandExecutedEvent"`
	SourceEventEmittedEvent `graphql:"... on SourceEventEmittedEvent"`
	Type                    gqlModel.AuditEventType `json:"type"`
	PluginName              string                  `json:"pluginName"`
}

// CommandExecutedEvent represents command executed event.
type CommandExecutedEvent struct {
	Command     string                `json:"command"`
	BotPlatform *gqlModel.BotPlatform `json:"botPlatform"`
	Channel     string                `json:"channel"`
	PluginName  string                `json:"pluginName"`
}

// SourceEventEmittedEvent represents source event emitted event.
type SourceEventEmittedEvent struct {
	Source *gqlModel.SourceEventDetails `json:"source"`
}

// AuditEventPage represents audit event page.
type AuditEventPage struct {
	Data       []AuditEvent
	TotalCount int
}

// CreateActionUpdateInput returns action create update input.
func CreateActionUpdateInput(deploy *gqlModel.Deployment) []*gqlModel.ActionCreateUpdateInput {
	source, executor := DeploymentSourceAndExecutor(deploy)
	return []*gqlModel.ActionCreateUpdateInput{
		{
			Name:        "action_xxx22",
			DisplayName: "Action Name",
			Enabled:     true,
			Command:     "kc get pods",
			Bindings: &gqlModel.ActionCreateUpdateInputBindings{
				Sources:   []string{source},
				Executors: []string{executor},
			},
		},
	}
}

// DeploymentSourceAndExecutor returns last 'kubernetes' source and 'kubectl' executor plugin found under plugins.
func DeploymentSourceAndExecutor(deploy *gqlModel.Deployment) (source string, executor string) {
	for _, plugin := range deploy.Plugins {
		if plugin.Type == gqlModel.PluginTypeSource && strings.Contains(plugin.Name, "kubernetes"){
			source = plugin.ConfigurationName
		}
		if plugin.Type == gqlModel.PluginTypeExecutor && strings.Contains(plugin.Name, "kubectl"){
			executor = plugin.ConfigurationName
		}
	}

	return source, executor
}

// ExpectedCommandExecutedEvents returns expected command executed events.
func ExpectedCommandExecutedEvents(commands []string, botPlatform *gqlModel.BotPlatform, channel string) []gqlModel.CommandExecutedEvent {
	var out = make([]gqlModel.CommandExecutedEvent, 0, len(commands))
	for _, c := range commands {
		out = append(out, gqlModel.CommandExecutedEvent{
			Command:     c,
			BotPlatform: botPlatform,
			Channel:     channel,
		})
	}

	return out
}

// CommandExecutedEventsFromAuditResponse returns command executed events from audit response.
func CommandExecutedEventsFromAuditResponse(auditPage AuditEventPage) []gqlModel.CommandExecutedEvent {
	var out = make([]gqlModel.CommandExecutedEvent, 0, auditPage.TotalCount)
	for _, a := range auditPage.Data {
		if a.Type != gqlModel.AuditEventTypeCommandExecuted {
			continue
		}
		out = append(out, gqlModel.CommandExecutedEvent{
			Command:     a.Command,
			BotPlatform: a.BotPlatform,
			Channel:     a.Channel,
		})
	}

	return out
}

// SourceEmittedEventsFromAuditResponse returns source emitted events from audit response.
func SourceEmittedEventsFromAuditResponse(auditPage AuditEventPage) []gqlModel.SourceEventEmittedEvent {
	var out = make([]gqlModel.SourceEventEmittedEvent, 0, auditPage.TotalCount)
	for _, a := range auditPage.Data {
		if a.Type != gqlModel.AuditEventTypeSourceEventEmitted {
			continue
		}
		out = append(out, gqlModel.SourceEventEmittedEvent{
			Source:     a.Source,
			PluginName: a.PluginName,
		})
	}

	return out
}
