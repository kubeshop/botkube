package heartbeat

import (
	"context"
	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ HeartbeatReporter = (*GraphQLHeartbeatReporter)(nil)

// GraphQLClient defines GraphQL client.
type GraphQLClient interface {
	Client() *graphql.Client
	DeploymentID() string
}

// GraphQLHeartbeatReporter reports heartbeat to GraphQL server.
type GraphQLHeartbeatReporter struct {
	log logrus.FieldLogger
	gql GraphQLClient
}

func newGraphQLHeartbeatReporter(logger logrus.FieldLogger, client GraphQLClient) *GraphQLHeartbeatReporter {
	return &GraphQLHeartbeatReporter{
		log: logger,
		gql: client,
	}
}

func (r *GraphQLHeartbeatReporter) ReportHeartbeat(ctx context.Context, heartbeat DeploymentHeartbeatInput) error {
	logger := r.log.WithFields(logrus.Fields{
		"deploymentID": r.gql.DeploymentID,
		"heartbeat":    heartbeat,
	})
	logger.Debug("Sending heartbeat...")
	var mutation struct {
		Success bool `graphql:"reportDeploymentHeartbeat(id: $id, in: $heartbeat)"`
	}
	variables := map[string]interface{}{
		"id":        graphql.ID(r.gql.DeploymentID()),
		"heartbeat": heartbeat,
	}
	err := r.gql.Client().Mutate(ctx, &mutation, variables)
	if err != nil {
		return errors.Wrap(err, "while sending heartbeat")
	}

	logger.Debug("Sending heartbeat successful.")
	return nil
}
