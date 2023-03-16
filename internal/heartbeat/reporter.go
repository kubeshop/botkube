package heartbeat

import (
	"context"

	"github.com/sirupsen/logrus"
)

type DeploymentHeartbeatInput struct {
	NodeCount int `json:"nodeCount"`
}

type HeartbeatReporter interface {
	ReportHeartbeat(ctx context.Context, heartBeat DeploymentHeartbeatInput) error
}

func GetReporter(remoteCfgEnabled bool, logger logrus.FieldLogger, gql GraphQLClient) HeartbeatReporter {
	if remoteCfgEnabled {
		return newGraphQLStatusReporter(
			logger.WithField("component", "GraphQLHeartbeatReporter"),
			gql,
		)
	}

	return newNoopHeartbeatReporter()
}
