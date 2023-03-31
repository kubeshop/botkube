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

func GetReporter(logger logrus.FieldLogger, gql GraphQLClient) HeartbeatReporter {
	return newGraphQLHeartbeatReporter(
		logger.WithField("component", "GraphQLHeartbeatReporter"),
		gql,
	)
}
