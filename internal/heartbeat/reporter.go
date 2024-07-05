package heartbeat

import (
	"context"

	"github.com/kubeshop/botkube/internal/health"

	"github.com/sirupsen/logrus"
)

type DeploymentHeartbeatInput struct {
	NodeCount int                             `json:"nodeCount"`
	Health    *DeploymentHeartbeatHealthInput `json:"health,omitempty"`
}

type DeploymentHeartbeatHealthPluginInput struct {
	Key   string              `json:"key"`
	Value health.PluginStatus `json:"value"`
}
type DeploymentHeartbeatHealthPlatformInput struct {
	Key   string                `json:"key"`
	Value health.PlatformStatus `json:"value"`
}
type DeploymentHeartbeatHealthInput struct {
	Botkube   health.BotStatus                         `json:"botkube"`
	Plugins   []DeploymentHeartbeatHealthPluginInput   `json:"plugins,omitempty"`
	Platforms []DeploymentHeartbeatHealthPlatformInput `json:"platforms,omitempty"`
}

type ReportHeartBeat struct {
	NodeCount int `json:"nodeCount"`
}

type HeartbeatReporter interface {
	ReportHeartbeat(ctx context.Context, heartBeat ReportHeartBeat) error
}

func GetReporter(logger logrus.FieldLogger, gql GraphQLClient, healthChecker health.Checker) HeartbeatReporter {
	return newGraphQLHeartbeatReporter(
		logger.WithField("component", "GraphQLHeartbeatReporter"),
		gql,
		healthChecker,
	)
}
