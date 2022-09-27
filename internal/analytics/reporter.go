package analytics

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/botkube/pkg/config"
)

// Reporter defines an analytics reporter implementation.
type Reporter interface {
	// RegisterCurrentIdentity loads the current anonymous identity and registers it.
	RegisterCurrentIdentity(ctx context.Context, k8sCli kubernetes.Interface) error

	// ReportCommand reports a new executed command. The command should be anonymized before using this method.
	ReportCommand(platform config.CommPlatformIntegration, command string, isInteractiveOrigin bool) error

	// ReportBotEnabled reports an enabled bot.
	ReportBotEnabled(platform config.CommPlatformIntegration) error

	// ReportSinkEnabled reports an enabled sink.
	ReportSinkEnabled(platform config.CommPlatformIntegration) error

	// ReportHandledEventSuccess reports a successfully handled event using a given communication platform.
	ReportHandledEventSuccess(integrationType config.IntegrationType, platform config.CommPlatformIntegration, eventDetails EventDetails) error

	// ReportHandledEventError reports a failure while handling event using a given communication platform.
	ReportHandledEventError(integrationType config.IntegrationType, platform config.CommPlatformIntegration, eventDetails EventDetails, err error) error

	// ReportFatalError reports a fatal app error.
	ReportFatalError(err error) error

	// Close cleans up the reporter resources.
	Close() error
}

// EventDetails contains anonymous details of a given event
type EventDetails struct {
	Type       config.EventType `json:"type"`
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
}
