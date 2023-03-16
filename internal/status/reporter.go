package status

import (
	"context"

	"github.com/sirupsen/logrus"
)

type DeploymentHeartbeatInput struct {
	NodeCount int `json:"nodeCount"`
}

type StatusReporter interface {
	ReportDeploymentStartup(ctx context.Context) error
	ReportDeploymentShutdown(ctx context.Context) error
	ReportDeploymentFailed(ctx context.Context) error
	SetResourceVersion(resourceVersion int)
	ReportHeartbeat(ctx context.Context, heartBeat DeploymentHeartbeatInput) error
}

func GetReporter(remoteCfgEnabled bool, logger logrus.FieldLogger, gql GraphQLClient, resVerClient ResVerClient, cfgVersion int) StatusReporter {
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
