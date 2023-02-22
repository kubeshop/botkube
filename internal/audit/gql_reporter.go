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

func (r *GraphQLAuditReporter) ReportExecutorAuditEvent(ctx context.Context, e AuditEvent) error {
	r.log.Debugf("Reporting executor audit event for ID: %s", r.gql.DeploymentID)
	var mutation struct {
		CreateAuditEvent struct {
			ID graphql.ID
		} `graphql:"createAuditEvent(input: $input)"`
	}
	variables := map[string]interface{}{
		"input": AuditEventCreateInput{
			PlatformUser: &e.PlatformUser,
			CreatedAt:    e.CreatedAt,
			PluginName:   e.PluginName,
			Channel:      e.Channel,
			DeploymentID: r.gql.DeploymentID,
			Type:         AuditEventTypeCommandExecuted,
			CommandExecuted: &AuditEventCommandCreateInput{
				BotPlatform: e.BotPlatform,
				Command:     e.Command,
			},
		},
	}

	return r.gql.Cli.Mutate(ctx, &mutation, variables)
}

func (r *GraphQLAuditReporter) ReportSourceAuditEvent(ctx context.Context, e AuditEvent) error {
	r.log.Debugf("Reporting source audit event for ID: %s", r.gql.DeploymentID)
	var mutation struct {
		CreateAuditEvent struct {
			ID graphql.ID
		} `graphql:"createAuditEvent(input: $input)"`
	}
	variables := map[string]interface{}{
		"input": AuditEventCreateInput{
			PlatformUser: &e.PlatformUser,
			CreatedAt:    e.CreatedAt,
			PluginName:   e.PluginName,
			Channel:      e.Channel,
			DeploymentID: r.gql.DeploymentID,
			Type:         AuditEventTypeSourceEventEmitted,
			SourceEventEmitted: &AuditEventSourceCreateInput{
				Event: e.Event,
			},
		},
	}

	return r.gql.Cli.Mutate(ctx, &mutation, variables)
}

type AuditEventCreateInput struct {
	PlatformUser       *string                       `json:"platformUser"`
	Type               AuditEventType                `json:"type"`
	CreatedAt          string                        `json:"createdAt"`
	DeploymentID       string                        `json:"deploymentId"`
	Channel            string                        `json:"channel"`
	PluginName         string                        `json:"pluginName"`
	SourceEventEmitted *AuditEventSourceCreateInput  `json:"sourceEventEmitted"`
	CommandExecuted    *AuditEventCommandCreateInput `json:"commandExecuted"`
}

type AuditEventCommandCreateInput struct {
	BotPlatform BotPlatform `json:"botPlatform"`
	Command     string      `json:"command"`
}

type AuditEventSourceCreateInput struct {
	Event interface{} `json:"event"`
}

type BotPlatform string

const (
	BotPlatformSLACk      BotPlatform = "SLACK"
	BotPlatformDiscord    BotPlatform = "DISCORD"
	BotPlatformMattermost BotPlatform = "MATTERMOST"
	BotPlatformMsTeams    BotPlatform = "MS_TEAMS"
)

func NewBotPlatform(s string) (BotPlatform, error) {
	switch strings.ToUpper(s) {
	case "SLACK":
		fallthrough
	case "SOCKETSLACK":
		return BotPlatformSLACk, nil
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

type AuditEventType string

const (
	AuditEventTypeCommandExecuted    AuditEventType = "COMMAND_EXECUTED"
	AuditEventTypeSourceEventEmitted AuditEventType = "SOURCE_EVENT_EMITTED"
)
