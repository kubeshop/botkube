package notifier

import (
	"github.com/kubeshop/botkube/internal/health"
	"github.com/kubeshop/botkube/pkg/config"
)

// Platform represents platform notifier
type Platform interface {
	GetStatus() health.PlatformStatus
	IntegrationName() config.CommPlatformIntegration
}
