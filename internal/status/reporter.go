package status

import (
	"context"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"

	"github.com/sirupsen/logrus"
)

type StatusReporter interface {
	ReportDeploymentConnectionInit(ctx context.Context, k8sVer string) error
	ReportDeploymentStartup(ctx context.Context) error
	ReportDeploymentShutdown(ctx context.Context) error
	ReportDeploymentFailure(ctx context.Context, errMsg string) error
	SetResourceVersion(resourceVersion int)
	SetLogger(logger logrus.FieldLogger)
}

func GetReporter(remoteCfgEnabled bool, gql GraphQLClient, resVerClient ResVerClient, log logrus.FieldLogger) StatusReporter {
	if remoteCfgEnabled {
		log = withDefaultLogger(log)
		return newGraphQLStatusReporter(
			log.WithField("component", "GraphQLStatusReporter"),
			gql,
			resVerClient,
		)
	}

	return newNoopStatusReporter()
}

func withDefaultLogger(log logrus.FieldLogger) logrus.FieldLogger {
	if log != nil {
		return log
	}
	return loggerx.New(config.Logger{
		Level: "info",
	})
}
