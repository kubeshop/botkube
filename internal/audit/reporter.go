package audit

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/graphql"
)

type AuditReporter interface {
	ReportExecutorAuditEvent(ctx context.Context, e AuditEvent) error
	ReportSourceAuditEvent(ctx context.Context, e AuditEvent) error
}

type AuditEvent struct {
	PlatformUser string
	CreatedAt    string
	Channel      string
	PluginName   string
	BotPlatform  BotPlatform
	Command      string
	Event        string
}

func NewAuditReporter(logger logrus.FieldLogger, gql *graphql.Gql) AuditReporter {
	if _, provided := os.LookupEnv(graphql.GqlProviderIdentifierEnvKey); provided {
		return newGraphQLAuditReporter(logger.WithField("component", "GraphQLAuditReporter"), gql)
	}
	return newNoopAuditReporter(logger.WithField("component", "NoopAuditReporter"))
}
