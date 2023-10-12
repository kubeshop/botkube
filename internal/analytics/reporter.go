package analytics

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

// Reporter defines an analytics reporter implementation.
type Reporter interface {
	// RegisterCurrentIdentity loads the current anonymous identity and registers it.
	RegisterCurrentIdentity(ctx context.Context, k8sCli kubernetes.Interface, deployID string) error

	// ReportCommandInput reports a new executed command. The command should be anonymized before using this method.
	ReportCommand(in ReportCommandInput) error

	// ReportBotEnabled reports an enabled bot.
	ReportBotEnabled(platform config.CommPlatformIntegration, commGroupIdx int) error

	// ReportSinkEnabled reports an enabled sink.
	ReportSinkEnabled(platform config.CommPlatformIntegration, commGroupIdx int) error

	// ReportHandledEventSuccess reports a successfully handled event using a given integration type, communication platform, and plugin.
	ReportHandledEventSuccess(event ReportEventInput) error

	// ReportHandledEventError reports a failure while handling event using a given integration type, communication platform, and plugin.
	ReportHandledEventError(event ReportEventInput, err error) error

	// ReportFatalError reports a fatal app error.
	ReportFatalError(err error) error

	// Close cleans up the reporter resources.
	Close() error
}

type ReportEventInput struct {
	IntegrationType       config.IntegrationType
	Platform              config.CommPlatformIntegration
	PluginName            string
	AnonymizedEventFields map[string]any
}

type ReportCommandInput struct {
	Platform   config.CommPlatformIntegration
	PluginName string
	Command    string
	Origin     command.Origin
	WithFilter bool
}
