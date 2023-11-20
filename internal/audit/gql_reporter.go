package audit

import (
	"context"

	"github.com/hasura/go-graphql-client"
	"github.com/sirupsen/logrus"

	gqlmodel "github.com/kubeshop/botkube/internal/remote/graphql"
)

var _ AuditReporter = (*GraphQLAuditReporter)(nil)

// GraphQLClient defines GraphQL client.
type GraphQLClient interface {
	Client() *graphql.Client
	DeploymentID() string
}

// GraphQLAuditReporter is the graphql audit reporter
type GraphQLAuditReporter struct {
	log logrus.FieldLogger
	gql GraphQLClient
}

func newGraphQLAuditReporter(logger logrus.FieldLogger, client GraphQLClient) *GraphQLAuditReporter {
	return &GraphQLAuditReporter{
		log: logger,
		gql: client,
	}
}

// ReportExecutorAuditEvent reports executor audit event using graphql interface
func (r *GraphQLAuditReporter) ReportExecutorAuditEvent(ctx context.Context, e ExecutorAuditEvent) error {
	r.log.Debugf("Reporting executor audit event for ID %q", r.gql.DeploymentID())
	var mutation struct {
		CreateAuditEvent struct {
			ID graphql.ID
		} `graphql:"createAuditEvent(input: $input)"`
	}
	variables := map[string]interface{}{
		"input": gqlmodel.AuditEventCreateInput{
			CreatedAt:    e.CreatedAt,
			PluginName:   e.PluginName,
			DeploymentID: r.gql.DeploymentID(),
			Type:         gqlmodel.AuditEventTypeCommandExecuted,
			CommandExecuted: &gqlmodel.AuditEventCommandCreateInput{
				PlatformUser:            e.PlatformUser,
				BotPlatform:             e.BotPlatform,
				Command:                 e.Command,
				Channel:                 e.Channel,
				AdditionalCreateContext: e.AdditionalCreateContext,
			},
		},
	}

	return r.gql.Client().Mutate(ctx, &mutation, variables)
}

// ReportSourceAuditEvent reports source audit event using graphql interface
func (r *GraphQLAuditReporter) ReportSourceAuditEvent(ctx context.Context, e SourceAuditEvent) error {
	r.log.Debugf("Reporting source audit event for ID %q", r.gql.DeploymentID())
	var mutation struct {
		CreateAuditEvent struct {
			ID graphql.ID
		} `graphql:"createAuditEvent(input: $input)"`
	}
	variables := map[string]interface{}{
		"input": gqlmodel.AuditEventCreateInput{
			CreatedAt:    e.CreatedAt,
			PluginName:   e.PluginName,
			DeploymentID: r.gql.DeploymentID(),
			Type:         gqlmodel.AuditEventTypeSourceEventEmitted,
			SourceEventEmitted: &gqlmodel.AuditEventSourceCreateInput{
				Event: e.Event,
				Source: &gqlmodel.AuditEventSourceDetailsInput{
					Name:        e.Source.Name,
					DisplayName: e.Source.DisplayName,
				},
			},
		},
	}

	return r.gql.Client().Mutate(ctx, &mutation, variables)
}
