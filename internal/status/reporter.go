package status

import (
	"context"

	"github.com/sirupsen/logrus"
)

type StatusReporter interface {
	ReportDeploymentStartup(ctx context.Context) (bool, error)
	ReportDeploymentShutdown(ctx context.Context) (bool, error)
	ReportDeploymentFailed(ctx context.Context) (bool, error)
}

func NewStatusReporter(logger logrus.FieldLogger, graphqlURL, deploymentID string) StatusReporter {
	if graphqlURL != "" {
		return newGraphQLStatusReporter(logger.WithField("component", "GraphQLStatusReporter"), graphqlURL, deploymentID)
	}
	return newNoopStatusReporter(logger.WithField("component", "NoopStatusReporter"))
}
