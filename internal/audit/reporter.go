package audit

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/remote/graphql"
)

// AuditReporter defines interface for reporting audit events
type AuditReporter interface {
	ReportExecutorAuditEvent(ctx context.Context, e ExecutorAuditEvent) error
	ReportSourceAuditEvent(ctx context.Context, e SourceAuditEvent) error
}

// ExecutorAuditEvent contains audit event data
type ExecutorAuditEvent struct {
	CreatedAt               string
	PluginName              string
	PlatformUser            string
	BotPlatform             *graphql.BotPlatform
	Command                 string
	Channel                 string
	AdditionalCreateContext map[string]interface{}
}

// SourceAuditEvent contains audit event data
type SourceAuditEvent struct {
	CreatedAt  string
	PluginName string
	Event      string
	Source     SourceDetails
}

type SourceDetails struct {
	Name        string
	DisplayName string
}

// GetReporter creates new AuditReporter
func GetReporter(remoteCfgSyncEnabled bool, logger logrus.FieldLogger, gql GraphQLClient) AuditReporter {
	if remoteCfgSyncEnabled {
		return newGraphQLAuditReporter(logger.WithField("component", "GraphQLAuditReporter"), gql)
	}
	return newNoopAuditReporter(nil)
}
