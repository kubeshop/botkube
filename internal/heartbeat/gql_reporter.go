package heartbeat

import (
	"context"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/health"
)

var _ HeartbeatReporter = (*GraphQLHeartbeatReporter)(nil)

// GraphQLClient defines GraphQL client.
type GraphQLClient interface {
	Client() *graphql.Client
	DeploymentID() string
}

// GraphQLHeartbeatReporter reports heartbeat to GraphQL server.
type GraphQLHeartbeatReporter struct {
	log           logrus.FieldLogger
	gql           GraphQLClient
	healthChecker health.Checker
}

func newGraphQLHeartbeatReporter(logger logrus.FieldLogger, client GraphQLClient, healthChecker health.Checker) *GraphQLHeartbeatReporter {
	return &GraphQLHeartbeatReporter{
		log:           logger,
		gql:           client,
		healthChecker: healthChecker,
	}
}

func (r *GraphQLHeartbeatReporter) ReportHeartbeat(ctx context.Context, heartbeat ReportHeartbeat) error {
	logger := r.log.WithFields(logrus.Fields{
		"deploymentID": r.gql.DeploymentID(),
		"heartbeat":    heartbeat,
	})
	logger.Debug("Sending heartbeat...")
	var mutation struct {
		Success bool `graphql:"reportDeploymentHeartbeat(id: $id, in: $input)"`
	}
	status := r.healthChecker.GetStatus()
	var pluginsStatuses []DeploymentHeartbeatHealthPluginInput
	var platformsStatuses []DeploymentHeartbeatHealthPlatformInput
	for pluginKey, pluginStatus := range status.Plugins {
		pluginsStatuses = append(pluginsStatuses, DeploymentHeartbeatHealthPluginInput{
			Key:   pluginKey,
			Value: pluginStatus,
		})
	}
	for platformKey, platformStatus := range status.Platforms {
		platformsStatuses = append(platformsStatuses, DeploymentHeartbeatHealthPlatformInput{
			Key:   platformKey,
			Value: platformStatus,
		})
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.gql.DeploymentID()),
		"input": DeploymentHeartbeatInput{
			NodeCount: heartbeat.NodeCount,
			Health: &DeploymentHeartbeatHealthInput{
				Botkube:   status.Botkube,
				Plugins:   pluginsStatuses,
				Platforms: platformsStatuses,
			},
		},
	}
	err := r.gql.Client().Mutate(ctx, &mutation, variables)
	if err != nil {
		return errors.Wrap(err, "while sending heartbeat")
	}

	logger.Debug("Sending heartbeat successful.")
	return nil
}
