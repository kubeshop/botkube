package status

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/graphql"
)

type StatusReporter interface {
	ReportDeploymentStartup(ctx context.Context) (bool, error)
	ReportDeploymentShutdown(ctx context.Context) (bool, error)
	ReportDeploymentFailed(ctx context.Context) (bool, error)
	SetResourceVersion(resourceVersion int)
}

func NewStatusReporter(logger logrus.FieldLogger, gql *graphql.Gql) StatusReporter {
	if _, provided := os.LookupEnv(graphql.GqlProviderIdentifierEnvKey); provided {
		return newGraphQLStatusReporter(logger.WithField("component", "GraphQLStatusReporter"), gql)
	}
	return newNoopStatusReporter(logger.WithField("component", "NoopStatusReporter"))
}
