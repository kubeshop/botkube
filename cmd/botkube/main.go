package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v44/github"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	segment "github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/strings"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/bot"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/controller"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/filterengine"
	"github.com/kubeshop/botkube/pkg/httpsrv"
	"github.com/kubeshop/botkube/pkg/sink"
)

const (
	componentLogFieldKey = "component"
	botLogFieldKey       = "bot"
	sinkLogFieldKey      = "sink"
	printAPIKeyCharCount = 3
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run wraps the main logic of the app to be able to properly clean up resources via deferred calls.
func run() error {
	// Load configuration
	config.RegisterFlags(pflag.CommandLine)
	conf, loadedCfgFiles, err := config.LoadWithDefaults(config.FromEnvOrFlag)
	if err != nil {
		return fmt.Errorf("while loading app configuration: %w", err)
	}

	logger := newLogger(conf.Settings.Log.Level, conf.Settings.Log.DisableColors)

	// Set up analytics reporter
	reporter, err := newAnalyticsReporter(conf.Analytics.Disable, logger)
	if err != nil {
		return fmt.Errorf("while creating analytics reporter: %w", err)
	}
	defer func() {
		err := reporter.Close()
		if err != nil {
			logger.Errorf("while closing reporter: %s", err.Error())
		}
	}()
	// from now on recover from any panic, report it and close reader and app.
	// The reader must be not closed to report the panic properly.
	defer analytics.ReportPanicIfOccurs(logger, reporter)

	reportFatalError := reportFatalErrFn(logger, reporter)

	// Set up context
	ctx := signals.SetupSignalHandler()
	ctx, cancelCtxFn := context.WithCancel(ctx)
	defer cancelCtxFn()

	errGroup, ctx := errgroup.WithContext(ctx)

	// Prepare K8s clients and mapper
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", conf.Settings.Kubeconfig)
	if err != nil {
		return reportFatalError("while loading k8s config", err)
	}
	dynamicCli, discoveryCli, mapper, err := getK8sClients(kubeConfig)
	if err != nil {
		return reportFatalError("while getting K8s clients", err)
	}

	// Register current anonymous identity
	k8sCli, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return reportFatalError("while creating K8s clientset", err)
	}
	err = reporter.RegisterCurrentIdentity(ctx, k8sCli, conf.Analytics.InstallationID)
	if err != nil {
		return reportFatalError("while registering current identity", err)
	}

	// Prometheus metrics
	metricsSrv := newMetricsServer(logger.WithField(componentLogFieldKey, "Metrics server"), conf.Settings.MetricsPort)
	errGroup.Go(func() error {
		defer analytics.ReportPanicIfOccurs(logger, reporter)
		return metricsSrv.Serve(ctx)
	})

	// Set up the filter engine
	filterEngine := filterengine.WithAllFilters(logger, dynamicCli, mapper, conf)

	// Create Executor Factory
	resMapping, err := execute.LoadResourceMappingIfShould(
		logger.WithField(componentLogFieldKey, "Resource Mapping Loader"),
		conf,
		discoveryCli,
	)
	if err != nil {
		return reportFatalError("while loading resource mapping", err)
	}

	executorFactory := execute.NewExecutorFactory(
		logger.WithField(componentLogFieldKey, "Executor"),
		execute.DefaultCommandRunnerFunc,
		*conf,
		filterEngine,
		resMapping,
		reporter,
	)

	commCfg := conf.Communications.GetFirst()
	var notifiers []controller.Notifier

	// Run bots
	if commCfg.Slack.Enabled {
		sb := bot.NewSlackBot(logger.WithField(botLogFieldKey, "Slack"), conf, executorFactory, reporter)
		notifiers = append(notifiers, sb)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return sb.Start(ctx)
		})
	}

	if commCfg.Mattermost.Enabled {
		mb := bot.NewMattermostBot(logger.WithField(botLogFieldKey, "Mattermost"), conf, executorFactory, reporter)
		notifiers = append(notifiers, mb)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return mb.Start(ctx)
		})
	}

	if commCfg.Teams.Enabled {
		tb := bot.NewTeamsBot(logger.WithField(botLogFieldKey, "MS Teams"), conf, executorFactory, reporter)
		notifiers = append(notifiers, tb)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return tb.Start(ctx)
		})
	}

	if commCfg.Discord.Enabled {
		db := bot.NewDiscordBot(logger.WithField(botLogFieldKey, "Discord"), conf, executorFactory, reporter)
		notifiers = append(notifiers, db)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return db.Start(ctx)
		})
	}

	// Run sinks
	if commCfg.Elasticsearch.Enabled {
		es, err := sink.NewElasticSearch(logger.WithField(sinkLogFieldKey, "Elasticsearch"), commCfg.Elasticsearch, reporter)
		if err != nil {
			return reportFatalError("while creating Elasticsearch sink", err)
		}
		notifiers = append(notifiers, es)
	}

	if commCfg.Webhook.Enabled {
		wh, err := sink.NewWebhook(logger.WithField(sinkLogFieldKey, "Webhook"), commCfg.Webhook, reporter)
		if err != nil {
			return reportFatalError("while creating Webhook sink", err)
		}

		notifiers = append(notifiers, wh)
	}

	// Start upgrade checker
	ghCli := github.NewClient(&http.Client{
		Timeout: 1 * time.Minute,
	})
	if conf.Settings.UpgradeNotifier {
		upgradeChecker := controller.NewUpgradeChecker(
			logger.WithField(componentLogFieldKey, "Upgrade Checker"),
			notifiers,
			ghCli.Repositories,
		)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return upgradeChecker.Run(ctx)
		})
	}

	// Start Config Watcher
	if conf.Settings.ConfigWatcher {
		cfgWatcher := controller.NewConfigWatcher(
			logger.WithField(componentLogFieldKey, "Config Watcher"),
			loadedCfgFiles,
			conf.Settings.ClusterName,
			notifiers,
		)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return cfgWatcher.Do(ctx, cancelCtxFn)
		})
	}

	// Create and start controller
	ctrl := controller.New(
		logger.WithField(componentLogFieldKey, "Controller"),
		conf,
		notifiers,
		filterEngine,
		dynamicCli,
		mapper,
		conf.Settings.InformersResyncPeriod,
		reporter,
	)

	err = ctrl.Start(ctx)
	if err != nil {
		return reportFatalError("while starting controller", err)
	}

	err = errGroup.Wait()
	if err != nil {
		return reportFatalError("while waiting for goroutines to finish gracefully", err)
	}

	return nil
}

func newLogger(logLevelStr string, logDisableColors bool) *logrus.Logger {
	logger := logrus.New()
	// Output to stdout instead of the default stderr
	logger.SetOutput(os.Stdout)

	// Only logger the warning severity or above.
	logLevel, err := logrus.ParseLevel(logLevelStr)
	if err != nil {
		// Set Info level as a default
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true, DisableColors: logDisableColors}

	return logger
}

func newMetricsServer(log logrus.FieldLogger, metricsPort string) *httpsrv.Server {
	addr := fmt.Sprintf(":%s", metricsPort)
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	return httpsrv.New(log, addr, router)
}

func newAnalyticsReporter(disableAnalytics bool, logger logrus.FieldLogger) (analytics.Reporter, error) {
	if disableAnalytics {
		logger.Info("Analytics disabled via configuration settings.")
		return analytics.NewNoopReporter(), nil
	}

	if analytics.APIKey == "" {
		logger.Info("Analytics disabled as the API key is missing.")
		return analytics.NewNoopReporter(), nil
	}

	wrappedLogger := logger.WithField(componentLogFieldKey, "Analytics reporter")
	wrappedLogger.Infof("Using API Key starting with %q...", strings.ShortenString(analytics.APIKey, printAPIKeyCharCount))
	segmentCli, err := segment.NewWithConfig(analytics.APIKey, segment.Config{
		Logger:  analytics.NewSegmentLoggerAdapter(wrappedLogger),
		Verbose: false,
	})
	if err != nil {
		return nil, fmt.Errorf("while creating new Analytics Client: %w", err)
	}

	analyticsReporter := analytics.NewSegmentReporter(wrappedLogger, segmentCli)
	if err != nil {
		return nil, err
	}

	return analyticsReporter, nil
}

func getK8sClients(cfg *rest.Config) (dynamic.Interface, discovery.DiscoveryInterface, meta.RESTMapper, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("while creating discovery client: %w", err)
	}

	dynamicK8sCli, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("while creating dynamic K8s client: %w", err)
	}

	discoCacheClient := cacheddiscovery.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoCacheClient)
	return dynamicK8sCli, discoveryClient, mapper, nil
}

func reportFatalErrFn(logger logrus.FieldLogger, reporter analytics.Reporter) func(ctx string, err error) error {
	return func(ctx string, err error) error {
		wrappedErr := fmt.Errorf("%s: %w", ctx, err)

		if reportErr := reporter.ReportFatalError(err); reportErr != nil {
			logger.Errorf("while reporting fatal error: %s", err.Error())
		}

		return wrappedErr
	}
}
