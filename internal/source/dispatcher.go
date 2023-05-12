package source

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/internal/audit"
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/action"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/bot"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/notifier"
)

// Dispatcher provides functionality to starts a given plugin, watches for incoming events and calling all notifiers to dispatch received event.
type Dispatcher struct {
	log                  logrus.FieldLogger
	manager              *plugin.Manager
	actionProvider       ActionProvider
	reporter             AnalyticsReporter
	auditReporter        audit.AuditReporter
	markdownNotifiers    []notifier.Bot
	interactiveNotifiers []notifier.Bot
	sinkNotifiers        []notifier.Sink
	restCfg              *rest.Config
}

// ActionProvider defines a provider that is responsible for automated actions.
type ActionProvider interface {
	RenderedActions(data any, sourceBindings []string) ([]action.Action, error)
	ExecuteAction(ctx context.Context, action action.Action) interactive.CoreMessage
}

// AnalyticsReporter defines a reporter that collects analytics data.
type AnalyticsReporter interface {
	// ReportHandledEventSuccess reports a successfully handled event using a given integration type, communication platform, and plugin.
	ReportHandledEventSuccess(event analytics.ReportEvent) error

	// ReportHandledEventError reports a failure while handling event using a given integration type, communication platform, and plugin.
	ReportHandledEventError(event analytics.ReportEvent, err error) error

	// ReportFatalError reports a fatal app error.
	ReportFatalError(err error) error

	// Close cleans up the reporter resources.
	Close() error
}

// NewDispatcher create a new Dispatcher instance.
func NewDispatcher(log logrus.FieldLogger, notifiers map[string]bot.Bot, sinkNotifiers []notifier.Sink, manager *plugin.Manager, actionProvider ActionProvider, reporter AnalyticsReporter, auditReporter audit.AuditReporter, restCfg *rest.Config) *Dispatcher {
	var (
		interactiveNotifiers []notifier.Bot
		markdownNotifiers    []notifier.Bot
	)
	for _, n := range notifiers {
		if n.IntegrationName().IsInteractive() {
			interactiveNotifiers = append(interactiveNotifiers, n)
			continue
		}

		markdownNotifiers = append(markdownNotifiers, n)
	}

	return &Dispatcher{
		log:                  log,
		manager:              manager,
		actionProvider:       actionProvider,
		reporter:             reporter,
		auditReporter:        auditReporter,
		interactiveNotifiers: interactiveNotifiers,
		markdownNotifiers:    markdownNotifiers,
		sinkNotifiers:        sinkNotifiers,
		restCfg:              restCfg,
	}
}

// Dispatch starts a given plugin, watches for incoming events and calling all notifiers to dispatch received event.
// Once we will have the gRPC contract established with proper Cloud Event schema, we should move also this logic here:
// https://github.com/kubeshop/botkube/blob/525c737956ff820a09321879284037da8bf5d647/pkg/controller/controller.go#L200-L253
func (d *Dispatcher) Dispatch(dispatch PluginDispatch) error {
	log := d.log.WithFields(logrus.Fields{
		"pluginName": dispatch.pluginName,
		"sourceName": dispatch.sourceName,
	})

	log.Info("Start source streaming...")

	sourceClient, err := d.manager.GetSource(dispatch.pluginName)
	if err != nil {
		return fmt.Errorf("while getting source client for %s: %w", dispatch.pluginName, err)
	}

	kubeconfig, err := plugin.GenerateKubeConfig(d.restCfg, dispatch.pluginContext, plugin.KubeConfigInput{})
	if err != nil {
		return fmt.Errorf("while generating kube config for %s: %w", dispatch.pluginName, err)
	}

	ctx := dispatch.ctx
	out, err := sourceClient.Stream(ctx, source.StreamInput{
		Configs: dispatch.pluginConfigs,
		Context: source.StreamInputContext{
			IsInteractivitySupported: dispatch.isInteractivitySupported,
			ClusterName:              dispatch.cfg.Settings.ClusterName,
			KubeConfig:               kubeconfig,
		},
	})
	if err != nil {
		return fmt.Errorf(`while opening stream for "%s.%s" source: %w`, dispatch.sourceName, dispatch.pluginName, err)
	}

	go func() {
		for {
			select {
			case event, ok := <-out.Output:
				if !ok {
					return
				}
				log.WithField("event", string(event)).Debug("Dispatching received event...")
				d.dispatch(ctx, event, dispatch)
			case msg, ok := <-out.Event:
				if !ok {
					return
				}
				log.WithField("message", msg).Debug("Dispatching received message...")
				d.dispatchMsg(ctx, msg, dispatch)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (d *Dispatcher) getBotNotifiers(dispatch PluginDispatch) []notifier.Bot {
	if dispatch.isInteractivitySupported {
		return d.interactiveNotifiers
	}
	return d.markdownNotifiers
}

func (d *Dispatcher) dispatchMsg(ctx context.Context, event source.Event, dispatch PluginDispatch) {
	var (
		pluginName = dispatch.pluginName
		sources    = []string{dispatch.sourceName}
	)

	for _, n := range d.getBotNotifiers(dispatch) {
		go func(n notifier.Bot) {
			defer analytics.ReportPanicIfOccurs(d.log, d.reporter)
			msg := interactive.CoreMessage{
				Message: event.Message,
			}
			err := n.SendMessage(ctx, msg, sources)
			if err != nil {
				reportErr := d.reportError(err, n, pluginName, event)
				if reportErr != nil {
					err = multierror.Append(err, fmt.Errorf("while reporting error: %w", reportErr))
				}

				d.log.Errorf("while sending bot message: %s", err.Error())
				return
			}

			reportErr := d.reportSuccess(n, pluginName, event)
			if reportErr != nil {
				d.log.Error(err)
			}
		}(n)
	}

	for _, n := range d.sinkNotifiers {
		go func(n notifier.Sink) {
			defer analytics.ReportPanicIfOccurs(d.log, d.reporter)
			err := n.SendEvent(ctx, event.RawObject, sources)
			if err != nil {
				reportErr := d.reportError(err, n, pluginName, event)
				if reportErr != nil {
					err = multierror.Append(err, fmt.Errorf("while reporting error: %w", reportErr))
				}

				d.log.Errorf("while sending sink message: %s", err.Error())
				return
			}

			reportErr := d.reportSuccess(n, pluginName, event)
			if reportErr != nil {
				d.log.Error(err)
			}
		}(n)
	}

	if err := d.reportAuditEvent(ctx, pluginName, event.RawObject, dispatch.sourceName, dispatch.sourceDisplayName); err != nil {
		d.log.Errorf("while reporting audit event for source %q: %s", dispatch.sourceName, err.Error())
	}

	// execute actions
	actions, err := d.actionProvider.RenderedActions(event.RawObject, sources)
	if err != nil {
		d.log.Errorf("while rendering automated actions: %s", err.Error())
		return
	}
	for _, act := range actions {
		log := d.log.WithFields(logrus.Fields{
			"name":    act.DisplayName,
			"command": act.Command,
		})
		log.Infof("Executing automated action...")
		genericMsg := d.actionProvider.ExecuteAction(ctx, act)
		log.WithField("message", fmt.Sprintf("%+v", genericMsg)).Debug("Automated action executed. Printing output message...")

		for _, n := range d.getBotNotifiers(dispatch) {
			go func(n notifier.Bot) {
				defer analytics.ReportPanicIfOccurs(d.log, d.reporter)
				err := n.SendMessage(ctx, genericMsg, sources)
				if err != nil {
					d.log.Errorf("while sending action result message: %s", err.Error())
				}
			}(n)
		}
	}
}

func (d *Dispatcher) dispatch(ctx context.Context, event []byte, dispatch PluginDispatch) {
	if event == nil {
		return
	}

	d.dispatchMsg(ctx, source.Event{
		Message: api.Message{
			BaseBody: api.Body{
				Plaintext: string(event),
			},
		},
	}, dispatch)
}

func (d *Dispatcher) reportAuditEvent(ctx context.Context, pluginName string, event any, sourceName, sourceDisplayName string) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("while marshaling audit event: %w", err)
	}

	e := audit.SourceAuditEvent{
		CreatedAt:  time.Now().Format(time.RFC3339),
		PluginName: pluginName,
		Event:      string(eventBytes),
		Source: audit.SourceDetails{
			Name:        sourceName,
			DisplayName: sourceDisplayName,
		},
	}
	return d.auditReporter.ReportSourceAuditEvent(ctx, e)
}

type genericNotifier interface {
	IntegrationName() config.CommPlatformIntegration
	Type() config.IntegrationType
}

func (d *Dispatcher) reportSuccess(n genericNotifier, pluginName string, event source.Event) error {
	errs := multierror.New()
	reportErr := d.reporter.ReportHandledEventSuccess(analytics.ReportEvent{
		IntegrationType:       n.Type(),
		Platform:              n.IntegrationName(),
		PluginName:            pluginName,
		AnonymizedEventFields: event.AnalyticsLabels,
	})
	if reportErr != nil {
		errs = multierror.Append(errs, fmt.Errorf("while reporting %s analytics: %w", n.Type(), reportErr))
	}
	return errs.ErrorOrNil()
}

func (d *Dispatcher) reportError(err error, n genericNotifier, pluginName string, event source.Event) error {
	errs := multierror.New()
	reportErr := d.reporter.ReportHandledEventError(analytics.ReportEvent{
		IntegrationType:       n.Type(),
		Platform:              n.IntegrationName(),
		PluginName:            pluginName,
		AnonymizedEventFields: event.AnalyticsLabels,
	}, err)
	if reportErr != nil {
		errs = multierror.Append(errs, fmt.Errorf("while reporting %s analytics: %w", n.Type(), reportErr))
	}

	return errs.ErrorOrNil()
}
