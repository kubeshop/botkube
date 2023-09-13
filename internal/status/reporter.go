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

func GetReporter(remoteCfgEnabled bool, gql GraphQLClient, resVerClient ResVerClient, logger *logrus.FieldLogger) StatusReporter {
	if remoteCfgEnabled {
		var log logrus.FieldLogger
		if logger == nil {
			log = loggerx.New(config.Logger{
				Level:         "info",
				DisableColors: false,
				Formatter:     "",
			})
		} else {
			log = *logger
		}

		return newGraphQLStatusReporter(
			log.WithField("component", "GraphQLStatusReporter"),
			gql,
			resVerClient,
		)
	}

	return newNoopStatusReporter()
}
