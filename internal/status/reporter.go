package status

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/graphql"
)

type StatusReporter interface {
	ReportDeploymentStartup(ctx context.Context) error
	ReportDeploymentShutdown(ctx context.Context) error
	ReportDeploymentFailed(ctx context.Context) error
	SetResourceVersion(resourceVersion int)
}

func NewStatusReporter(remoteCfgEnabled bool, logger logrus.FieldLogger, gql *graphql.Gql, resVerClient ResVerClient, cfgVersion int) StatusReporter {
	if remoteCfgEnabled {
		return newGraphQLStatusReporter(
			logger.WithField("component", "GraphQLStatusReporter"),
			gql,
			resVerClient,
			cfgVersion,
		)
	}

	return newNoopStatusReporter()
}
