//go:build cloud_slack_dev_e2e

package cloud_slack_dev_e2e

import (
	gqlModel "github.com/kubeshop/botkube/internal/remote/graphql"
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
func CreateActionUpdateInput() []*gqlModel.ActionCreateUpdateInput {
	var actions []*gqlModel.ActionCreateUpdateInput
	source1 := "kubernetes_config"
	executor1 := "kubectl_config"
	actions = append(actions, &gqlModel.ActionCreateUpdateInput{
		Name:        "action_xxx22",
		DisplayName: "Action Name",
		Enabled:     true,
		Command:     "kc get pods",
		Bindings: &gqlModel.ActionCreateUpdateInputBindings{
			Sources:   []string{source1},
			Executors: []string{executor1},
		},
	})

	return actions
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
