package status

import (
	"context"
	"os"

	"github.com/kubeshop/botkube/internal/graphql"
	"github.com/sirupsen/logrus"
)

type StatusReporter interface {
	ReportDeploymentStartup(ctx context.Context) (bool, error)
	ReportDeploymentShutdown(ctx context.Context) (bool, error)
	ReportDeploymentFailed(ctx context.Context) (bool, error)
}

func NewStatusReporter(logger logrus.FieldLogger, gql *graphql.Gql) StatusReporter {
	if _, provided := os.LookupEnv(graphql.GqlProviderEndpointEnvKey); provided {
		return newGraphQLStatusReporter(logger.WithField("component", "GraphQLStatusReporter"), gql)
	}
	return newNoopStatusReporter(logger.WithField("component", "NoopStatusReporter"))
}
