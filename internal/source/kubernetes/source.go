package kubernetes

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/internal/command"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/source/kubernetes/commander"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/filterengine"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	pkgConfig "github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

var _ source.Source = (*Source)(nil)

var (
	// configJSONSchema JSON schema with duplications,
	// as some UI components (e.g. https://github.com/rjsf-team/react-jsonschema-form) don't support nested defaults for definitions.
	//go:embed config-jsonschema.json
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
	pluginVersion            string
	config                   config.Config
	logger                   logrus.FieldLogger
	eventCh                  chan source.Event
	startTime                time.Time
	recommFactory            RecommendationFactory
	commandGuard             *command.CommandGuard
	filterEngine             filterengine.FilterEngine
	clusterName              string
	kubeConfig               []byte
	messageBuilder           *MessageBuilder
	isInteractivitySupported bool

	source.HandleExternalRequestUnimplemented
}

// NewSource returns a new instance of Source.
func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
	}
}

// Stream streams Kubernetes events
func (*Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	if err := pluginx.ValidateKubeConfigProvided(PluginName, input.Context.KubeConfig); err != nil {
		return source.StreamOutput{}, err
	}

	cfg, err := config.MergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	s := Source{
		startTime: time.Now(),
		eventCh:   make(chan source.Event),
		config:    cfg,
		logger: loggerx.New(pkgConfig.Logger{
			Level: cfg.Log.Level,
		}),
		clusterName:              input.Context.ClusterName,
		kubeConfig:               input.Context.KubeConfig,
		isInteractivitySupported: input.Context.IsInteractivitySupported,
	}

	go consumeEvents(ctx, s)
	return source.StreamOutput{
		Event: s.eventCh,
	}, nil
}

// Metadata returns metadata of Kubernetes configuration
func (s *Source) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:          s.pluginVersion,
		Description:      description,
		DocumentationURL: "https://docs.botkube.io/configuration/source/kubernetes",
		JSONSchema: api.JSONSchema{
			Value: configJSONSchema,
		},
	}, nil
}

func consumeEvents(ctx context.Context, s Source) {
	client, err := NewClient(s.kubeConfig)
	exitOnError(err, s.logger)

	dynamicKubeInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(client.dynamicCli, s.config.InformerResyncPeriod)
	router := NewRouter(client.mapper, client.dynamicCli, s.logger)
	router.BuildTable(&s.config)
	s.recommFactory = recommendation.NewFactory(s.logger.WithField("component", "Recommendations"), client.dynamicCli)
	s.commandGuard = command.NewCommandGuard(s.logger.WithField(componentLogFieldKey, "Command Guard"), client.discoveryCli)
	cmdr := commander.NewCommander(s.logger.WithField(componentLogFieldKey, "Commander"), s.commandGuard, s.config.Commands)
	s.messageBuilder = NewMessageBuilder(s.isInteractivitySupported, s.logger.WithField(componentLogFieldKey, "Message Builder"), cmdr)
	s.filterEngine = filterengine.WithAllFilters(s.logger, client.dynamicCli, client.mapper, s.config.Filters)

	err = router.RegisterInformers([]config.EventType{
		config.CreateEvent,
		config.UpdateEvent,
		config.DeleteEvent,
	}, func(resource string) (cache.SharedIndexInformer, error) {
		gvr, err := parseResourceArg(resource, client.mapper)
		if err != nil {
			s.logger.Infof("Unable to parse resource: %s to register with informer\n", resource)
			return nil, err
		}
		return dynamicKubeInformerFactory.ForResource(gvr).Informer(), nil
	})
	if err != nil {
		exitOnError(err, s.logger.WithFields(logrus.Fields{
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
				s.logger.Infof("Unable to parse resource: %s to register with informer\n", resource)
				return nil, err
			}
			return dynamicKubeInformerFactory.ForResource(gvr).Informer(), nil
		})
	if err != nil {
		exitOnError(err, s.logger.WithFields(logrus.Fields{
			"srcEvent": config.ErrorEvent,
			"dstEvent": config.WarningEvent,
			"error":    err.Error(),
		}))
	}

	eventTypes := []config.EventType{
		config.CreateEvent,
		config.DeleteEvent,
		config.UpdateEvent,
	}
	for _, eventType := range eventTypes {
		router.RegisterEventHandler(
			ctx,
			s,
			eventType,
			handleEvent,
		)
	}

	router.HandleMappedEvent(
		ctx,
		s,
		config.ErrorEvent,
		handleEvent,
	)

	stopCh := ctx.Done()
	dynamicKubeInformerFactory.Start(stopCh)
}

func handleEvent(ctx context.Context, s Source, e event.Event, updateDiffs []string) {
	s.logger.Debugf("Processing %s to %s/%v in %s namespace", e.Type, e.Resource, e.Name, e.Namespace)
	enrichEventWithAdditionalMetadata(s, &e)

	// Skip older events
	if !e.TimeStamp.IsZero() && e.TimeStamp.Before(s.startTime) {
		s.logger.Debug("Skipping older events")
		return
	}

	// Check for significant Update Events in objects
	if e.Type == config.UpdateEvent && len(updateDiffs) > 0 {
		e.Messages = append(e.Messages, updateDiffs...)
	}

	// Filter events
	e = s.filterEngine.Run(ctx, e)
	if e.Skip {
		s.logger.Debugf("Skipping event: %#v", e)
		return
	}

	if len(e.Kind) <= 0 {
		s.logger.Warn("sendEvent received e with Kind nil. Hence skipping.")
		return
	}

	recRunner, recCfg := s.recommFactory.New(s.config)
	err := recRunner.Do(ctx, &e)
	if err != nil {
		s.logger.Errorf("while running recommendations: %w", err)
		return
	}

	if recommendation.ShouldIgnoreEvent(&recCfg, e) {
		s.logger.Debugf("Skipping event as it is related to recommendation informers and doesn't have any recommendations: %#v", e)
		return
	}

	msg, err := s.messageBuilder.FromEvent(e, s.config.ExtraButtons)
	if err != nil {
		s.logger.Errorf("while rendering message from event: %w", err)
		return
	}

	message := source.Event{
		Message:         msg,
		RawObject:       e,
		AnalyticsLabels: event.AnonymizedEventDetailsFrom(e),
	}
	s.eventCh <- message
}

func enrichEventWithAdditionalMetadata(s Source, event *event.Event) {
	event.Cluster = s.clusterName
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
