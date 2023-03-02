package status

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/graphql"
)

type StatusReporter interface {
	ReportDeploymentStartup(ctx context.Context) (bool, error)
	ReportDeploymentShutdown(ctx context.Context) (bool, error)
	ReportDeploymentFailed(ctx context.Context) (bool, error)
	SetResourceVersion(resourceVersion int)
}

func NewStatusReporter(remoteCfgEnabled bool, logger logrus.FieldLogger, gql *graphql.Gql, cfgVersion int) StatusReporter {
	if remoteCfgEnabled {
		return newGraphQLStatusReporter(logger.WithField("component", "GraphQLStatusReporter"), gql, cfgVersion)
	}

	return newNoopStatusReporter()
}
