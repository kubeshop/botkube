package kubernetes

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/internal/command"
	"github.com/kubeshop/botkube/internal/source/kubernetes/commander"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/filterengine"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	pkgConfig "github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/plugin"
)

var _ source.Source = (*Source)(nil)

var (
	// configJSONSchema contains duplications,
	// as some UI components (e.g. https://github.com/rjsf-team/react-jsonschema-form) don't support nested defaults for definitions.
	//go:embed config_schema.json
	configJSONSchema string
)

const (
	// PluginName is the name of the Kubernetes Botkube plugin.
	PluginName = "kubernetes"

	description = "Consume Kubernetes events and get notifications with additional warnings and recommendations."

	componentLogFieldKey = "component"
)

type RecommendationFactory interface {
	New(cfg config.Config) (recommendation.AggregatedRunner, config.Recommendations)
}

// Source Kubernetes source plugin data structure
type Source struct {
	bgProcessor   *backgroundProcessor
	pluginVersion string
	configStore   *configurationStore

	mu sync.Mutex

	source.HandleExternalRequestUnimplemented
}

type SourceConfig struct {
	name                     string
	eventCh                  chan source.Event
	cfg                      config.Config
	isInteractivitySupported bool
	clusterName              string
	kubeConfig               []byte

	*ActiveSourceConfig
}

type ActiveSourceConfig struct {
	logger         logrus.FieldLogger
	messageBuilder *MessageBuilder
	filterEngine   *filterengine.DefaultFilterEngine
	recommFactory  *recommendation.Factory
}

// NewSource returns a new instance of Source.
func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
		configStore:   newConfigurations(),
		bgProcessor:   newBackgroundProcessor(),
	}
}

// Metadata returns metadata of Kubernetes configuration.
func (s *Source) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:          s.pluginVersion,
		Description:      description,
		DocumentationURL: "https://docs.botkube.io/configuration/source/kubernetes",
		JSONSchema: api.JSONSchema{
			Value: configJSONSchema,
		},
		Recommended: true,
	}, nil
}

// Stream streams Kubernetes events.
// WARNING: This method has to be thread-safe.
func (s *Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	kubeConfig := input.Context.KubeConfig
	if err := plugin.ValidateKubeConfigProvided(PluginName, kubeConfig); err != nil {
		return source.StreamOutput{}, err
	}

	cfg, err := config.MergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}

	srcName := input.Context.SourceName
	eventCh := make(chan source.Event)
	s.configStore.Store(srcName, SourceConfig{
		name:                     srcName,
		eventCh:                  eventCh,
		cfg:                      cfg,
		isInteractivitySupported: input.Context.IsInteractivitySupported,
		clusterName:              input.Context.ClusterName,
		kubeConfig:               kubeConfig,
	})

	globalSrcCfg, err := s.getGlobalSrcCfg()
	if err != nil {
		exitOnError(err, loggerx.New(pkgConfig.Logger{}).WithField("error", err.Error()))
	}

	id := s.configStore.Len()
	globalLogger := loggerx.NewStderr(pkgConfig.Logger{
		Level: globalSrcCfg.cfg.Log.Level,
	}).WithField("id", id)

	err = s.bgProcessor.StopAndWait(globalLogger)
	if err != nil {
		globalLogger.WithField("error", err.Error()).Error("While stopping background processor")
	}

	cfgsByKubeConfig := s.configStore.CloneByKubeconfig()
	globalLogger.Infof("Reconfiguring background process with %d different Kubeconfig(s)...", len(cfgsByKubeConfig))

	var fns []func(context.Context)
	for kubeConfig, srcCfgs := range cfgsByKubeConfig {
		fn := s.genFnForKubeconfig(id, []byte(kubeConfig), globalLogger, globalSrcCfg.cfg.InformerResyncPeriod, srcCfgs)
		fns = append(fns, fn)
	}

	s.bgProcessor.Run(ctx, fns)

	return source.StreamOutput{
		Event: eventCh,
	}, nil
}

func (s *Source) configureProcessForSources(ctx context.Context, id int, kubeConfig []byte, globalLogger logrus.FieldLogger, informerResyncPeriod time.Duration, srcCfgs map[string]SourceConfig) error {
	client, err := NewClient(kubeConfig)
	if err != nil {
		return fmt.Errorf("while creating Kubernetes client: %w", err)
	}

	for _, srcCfg := range srcCfgs {
		cfg := srcCfg.cfg
		logger := loggerx.NewStderr(pkgConfig.Logger{
			Level: cfg.Log.Level,
		}).WithField("id", id)

		commandGuard := command.NewCommandGuard(logger.WithField(componentLogFieldKey, "Command Guard"), client.discoveryCli)
		cmdr := commander.NewCommander(logger.WithField(componentLogFieldKey, "Commander"), commandGuard, cfg.Commands)

		recommFactory := recommendation.NewFactory(logger.WithField("component", "Recommendations"), client.dynamicCli)
		filterEngine := filterengine.WithAllFilters(logger, client.dynamicCli, client.mapper, cfg.Filters)
		messageBuilder := NewMessageBuilder(srcCfg.isInteractivitySupported, logger.WithField(componentLogFieldKey, "Message Builder"), cmdr)

		srcCfg.ActiveSourceConfig = &ActiveSourceConfig{
			logger:         logger,
			recommFactory:  recommFactory,
			filterEngine:   filterEngine,
			messageBuilder: messageBuilder,
		}

		s.configStore.Store(srcCfg.name, srcCfg)
	}

	router := NewRouter(client.mapper, client.dynamicCli, globalLogger)
	router.BuildTable(srcCfgs)

	globalLogger.Info("Registering informers...")
	dynamicKubeInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(client.dynamicCli, informerResyncPeriod)

	err = router.RegisterInformers([]config.EventType{
		config.CreateEvent,
		config.UpdateEvent,
		config.DeleteEvent,
	}, func(resource string) (cache.SharedIndexInformer, error) {
		gvr, err := parseResourceArg(resource, client.mapper)
		if err != nil {
			globalLogger.Errorf("Unable to parse resource: %s to register with informer\n", resource)
			return nil, err
		}
		return dynamicKubeInformerFactory.ForResource(gvr).Informer(), nil
	})
	if err != nil {
		exitOnError(err, globalLogger.WithFields(logrus.Fields{
			"events": []config.EventType{
				config.CreateEvent,
				config.UpdateEvent,
				config.DeleteEvent,
			},
			"error": err.Error(),
		}))
	}

	err = router.MapWithEventsInformer(
		config.ErrorEvent,
		config.WarningEvent,
		func(resource string) (cache.SharedIndexInformer, error) {
			gvr, err := parseResourceArg(resource, client.mapper)
			if err != nil {
				globalLogger.Infof("Unable to parse resource: %s to register with informer\n", resource)
				return nil, err
			}
			return dynamicKubeInformerFactory.ForResource(gvr).Informer(), nil
		})
	if err != nil {
		return fmt.Errorf("while mapping with events informer: %w", err)
	}

	eventTypes := []config.EventType{
		config.CreateEvent,
		config.DeleteEvent,
		config.UpdateEvent,
	}
	for _, eventType := range eventTypes {
		router.RegisterEventHandler(
			ctx,
			eventType,
			s.handleEventFn(globalLogger),
		)
	}

	router.HandleMappedEvent(
		ctx,
		config.ErrorEvent,
		s.handleEventFn(globalLogger),
	)

	globalLogger.Info("Starting background process...")
	stopCh := ctx.Done()
	dynamicKubeInformerFactory.Start(stopCh)
	<-stopCh
	dynamicKubeInformerFactory.Shutdown()
	globalLogger.Info("Stopped background process...")
	return nil
}

func (s *Source) handleEventFn(log logrus.FieldLogger) func(ctx context.Context, e event.Event, sources, updateDiffs []string) {
	globalLogger := log

	return func(ctx context.Context, e event.Event, sources, updateDiffs []string) {
		globalLogger.Debugf("Processing %s to %s/%v in %s namespace", e.Type, e.Resource, e.Name, e.Namespace)

		// Skip older events
		if !e.TimeStamp.IsZero() && e.TimeStamp.Before(s.bgProcessor.StartTime()) {
			globalLogger.Debug("Skipping older event...")
			return
		}

		// Check for significant Update Events in objects
		if e.Type == config.UpdateEvent && len(updateDiffs) > 0 {
			e.Messages = append(e.Messages, updateDiffs...)
		}

		errs := multierror.New()
		for _, sourceKey := range sources {
			eventCopy := e

			srcCfg, ok := s.configStore.Get(sourceKey)
			if !ok {
				errs = multierror.Append(errs, fmt.Errorf("source with key %q not found", sourceKey))
			}

			if srcCfg.ActiveSourceConfig == nil {
				globalLogger.Errorf("ActiveSourceConfig not found for source %s", srcCfg.name)
				continue
			}

			setClusterName(srcCfg.clusterName, &eventCopy)

			// Filter events
			e = srcCfg.filterEngine.Run(ctx, eventCopy)
			if e.Skip {
				srcCfg.logger.Debugf("Skipping event: %#v", eventCopy)
				continue
			}

			if len(e.Kind) <= 0 {
				srcCfg.logger.Warn("sendEvent received e with Kind nil. Hence skipping.")
				continue
			}

			recRunner, recCfg := srcCfg.recommFactory.New(srcCfg.cfg)
			err := recRunner.Do(ctx, &eventCopy)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("while running recommendation: %w", err))
				continue
			}

			if recommendation.ShouldIgnoreEvent(&recCfg, eventCopy) {
				srcCfg.logger.Debugf("Skipping event as it is related to recommendation informers and doesn't have any recommendations: %#v", e)
				continue
			}

			msg, err := srcCfg.messageBuilder.FromEvent(eventCopy, srcCfg.cfg.ExtraButtons)
			if err != nil {
				srcCfg.logger.Errorf("while rendering message from event: %w", err)
				continue
			}

			message := source.Event{
				Message:         msg,
				RawObject:       eventCopy,
				AnalyticsLabels: event.AnonymizedEventDetailsFrom(eventCopy),
			}

			srcCfg.eventCh <- message
		}

		if errs.ErrorOrNil() != nil {
			globalLogger.Errorf("while sending event: %w", errs)
		}
	}
}

func (s *Source) genFnForKubeconfig(id int, kubeConfig []byte, globalLogger logrus.FieldLogger, informerResyncPeriod time.Duration, srcCfgs map[string]SourceConfig) func(ctx context.Context) {
	return func(ctx context.Context) {
		err := s.configureProcessForSources(ctx, id, kubeConfig, globalLogger, informerResyncPeriod, srcCfgs)
		if err != nil {
			exitOnError(fmt.Errorf("while configuring process for sources: %w", err), globalLogger.WithFields(logrus.Fields{
				"srcEvent": config.ErrorEvent,
				"dstEvent": config.WarningEvent,
				"error":    err.Error(),
			}))
		}
	}
}

func setClusterName(clusterName string, event *event.Event) {
	event.Cluster = clusterName
}

func parseResourceArg(arg string, mapper meta.RESTMapper) (schema.GroupVersionResource, error) {
	gvr, err := strToGVR(arg)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("while converting string to GroupVersionReference: %w", err)
	}

	// Validate the GVR provided
	if _, err := mapper.ResourcesFor(gvr); err != nil {
		return schema.GroupVersionResource{}, err
	}
	return gvr, nil
}

func strToGVR(arg string) (schema.GroupVersionResource, error) {
	const separator = "/"
	gvrStrParts := strings.Split(arg, separator)
	switch len(gvrStrParts) {
	case 2:
		return schema.GroupVersionResource{Group: "", Version: gvrStrParts[0], Resource: gvrStrParts[1]}, nil
	case 3:
		return schema.GroupVersionResource{Group: gvrStrParts[0], Version: gvrStrParts[1], Resource: gvrStrParts[2]}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("invalid string: expected 2 or 3 parts when split by %q", separator)
	}
}

func exitOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Error(err)
		// Error message is not propagated to Botkube core without this wait.
		time.Sleep(time.Second * 2)
		os.Exit(1)
	}
}

func (s *Source) getGlobalSrcCfg() (SourceConfig, error) {
	if s.configStore.Len() == 0 {
		return SourceConfig{}, fmt.Errorf("no source configurations found")
	}

	globalSrcCfg, ok := s.configStore.GetGlobal()
	if !ok {
		return SourceConfig{}, fmt.Errorf("global source configuration not found")
	}

	return globalSrcCfg, nil
}
