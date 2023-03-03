package audit

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/graphql"
)

// AuditReporter defines interface for reporting audit events
type AuditReporter interface {
	ReportExecutorAuditEvent(ctx context.Context, e ExecutorAuditEvent) error
	ReportSourceAuditEvent(ctx context.Context, e SourceAuditEvent) error
}

// ExecutorAuditEvent contains audit event data
type ExecutorAuditEvent struct {
	CreatedAt    string
	PluginName   string
	PlatformUser string
	BotPlatform  BotPlatform
	Command      string
	Channel      string
}

// SourceAuditEvent contains audit event data
type SourceAuditEvent struct {
	CreatedAt  string
	PluginName string
	Event      string
	Bindings   []string
}

// NewAuditReporter creates new AuditReporter
func NewAuditReporter(remoteCfgSyncEnabled bool, logger logrus.FieldLogger, gql *graphql.Gql) AuditReporter {
	if remoteCfgSyncEnabled {
		return newGraphQLAuditReporter(logger.WithField("component", "GraphQLAuditReporter"), gql)
	}
	return newNoopAuditReporter(nil)
}
