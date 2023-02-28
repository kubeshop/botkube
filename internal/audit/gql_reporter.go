package audit

import (
	"context"
	"fmt"
	"strings"

	"github.com/hasura/go-graphql-client"
	"github.com/sirupsen/logrus"

	gql "github.com/kubeshop/botkube/internal/graphql"
)

var _ AuditReporter = (*GraphQLAuditReporter)(nil)

// GraphQLAuditReporter is the graphql audit reporter
type GraphQLAuditReporter struct {
	log logrus.FieldLogger
	gql *gql.Gql
}

func newGraphQLAuditReporter(logger logrus.FieldLogger, client *gql.Gql) *GraphQLAuditReporter {
	return &GraphQLAuditReporter{
		log: logger,
		gql: client,
	}
}

// ReportExecutorAuditEvent reports executor audit event using graphql interface
func (r *GraphQLAuditReporter) ReportExecutorAuditEvent(ctx context.Context, e ExecutorAuditEvent) error {
	r.log.Debugf("Reporting executor audit event for ID: %s", r.gql.DeploymentID)
	var mutation struct {
		CreateAuditEvent struct {
			ID graphql.ID
		} `graphql:"createAuditEvent(input: $input)"`
	}
	variables := map[string]interface{}{
		"input": AuditEventCreateInput{
			CreatedAt:    e.CreatedAt,
			PluginName:   e.PluginName,
			DeploymentID: r.gql.DeploymentID,
			Type:         AuditEventTypeCommandExecuted,
			CommandExecuted: &AuditEventCommandCreateInput{
				PlatformUser: e.PlatformUser,
				BotPlatform:  e.BotPlatform,
				Command:      e.Command,
				Channel:      e.Channel,
			},
		},
	}

	return r.gql.Cli.Mutate(ctx, &mutation, variables)
}

// ReportSourceAuditEvent reports source audit event using graphql interface
func (r *GraphQLAuditReporter) ReportSourceAuditEvent(ctx context.Context, e SourceAuditEvent) error {
	r.log.Debugf("Reporting source audit event for ID: %s", r.gql.DeploymentID)
	var mutation struct {
		CreateAuditEvent struct {
			ID graphql.ID
		} `graphql:"createAuditEvent(input: $input)"`
	}
	variables := map[string]interface{}{
		"input": AuditEventCreateInput{
			CreatedAt:    e.CreatedAt,
			PluginName:   e.PluginName,
			DeploymentID: r.gql.DeploymentID,
			Type:         AuditEventTypeSourceEventEmitted,
			SourceEventEmitted: &AuditEventSourceCreateInput{
				Event:    e.Event,
				Bindings: e.Bindings,
			},
		},
	}

	return r.gql.Cli.Mutate(ctx, &mutation, variables)
}

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
