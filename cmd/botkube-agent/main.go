package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v53/github"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	segment "github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/strings"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/internal/audit"
	"github.com/kubeshop/botkube/internal/command"
	intconfig "github.com/kubeshop/botkube/internal/config"
	"github.com/kubeshop/botkube/internal/config/reloader"
	"github.com/kubeshop/botkube/internal/config/remote"
	"github.com/kubeshop/botkube/internal/health"
	"github.com/kubeshop/botkube/internal/heartbeat"
	"github.com/kubeshop/botkube/internal/insights"
	"github.com/kubeshop/botkube/internal/kubex"
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/internal/source"
	"github.com/kubeshop/botkube/internal/status"
	"github.com/kubeshop/botkube/internal/storage"
	"github.com/kubeshop/botkube/pkg/action"
	"github.com/kubeshop/botkube/pkg/bot"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/controller"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/httpx"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/kubeshop/botkube/pkg/maputil"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/notifier"
	"github.com/kubeshop/botkube/pkg/sink"
	"github.com/kubeshop/botkube/pkg/version"
)

const (
	componentLogFieldKey      = "component"
	botLogFieldKey            = "bot"
	sinkLogFieldKey           = "sink"
	commGroupFieldKey         = "commGroup"
	printAPIKeyCharCount      = 3
	reportHeartbeatInterval   = 10
	reportHeartbeatMaxRetries = 30
)

func main() {
	// Set up context
	ctx := signals.SetupSignalHandler()
	ctx, cancelCtxFn := context.WithCancel(ctx)
	defer cancelCtxFn()

	err := run(ctx)
	if errors.Is(err, context.Canceled) {
		return
	}

	loggerx.ExitOnError(err, "while running application")
}

// run wraps the main logic of the app to be able to properly clean up resources via deferred calls.
func run(ctx context.Context) (err error) {
	// Load configuration
	intconfig.RegisterFlags(pflag.CommandLine)

	remoteCfg, remoteCfgEnabled := remote.GetConfig()
	var (
		gqlClient    *remote.Gql
		deployClient *remote.DeploymentClient
	)
	if remoteCfgEnabled {
		gqlClient = remote.NewDefaultGqlClient(remoteCfg)
		deployClient = remote.NewDeploymentClient(gqlClient)
	}

	statusReporter := status.GetReporter(remoteCfgEnabled, gqlClient, deployClient, nil)
	if err = statusReporter.ReportDeploymentConnectionInit(ctx, ""); err != nil {
		return fmt.Errorf("while reporting botkube connection initialization %w", err)
	}

	cfgProvider := intconfig.GetProvider(remoteCfgEnabled, deployClient)
	configs, cfgVersion, err := cfgProvider.Configs(ctx)
	if err != nil {
		return fmt.Errorf("while loading configuration files: %w", err)
	}

	conf, confDetails, err := config.LoadWithDefaults(configs)
	if err != nil {
		return fmt.Errorf("while merging app configuration: %w", err)
	}
	if conf == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	logger := loggerx.New(conf.Settings.Log)
	if confDetails.ValidateWarnings != nil {
		logger.Warnf("Configuration validation warnings: %v", confDetails.ValidateWarnings.Error())
	}
	// Set up analytics reporter
	analyticsReporter, err := getAnalyticsReporter(conf.Analytics.Disable, logger)
	if err != nil {
		return fmt.Errorf("while creating analytics reporter: %w", err)
	}
	defer func() {
		err := analyticsReporter.Close()
		if err != nil {
			logger.Errorf("while closing reporter: %s", err.Error())
		}
	}()
	// from now on recover from any panic, report it and close reader and app.
	// The reader must be not closed to report the panic properly.
	defer analytics.ReportPanicIfOccurs(logger, analyticsReporter)

	reportFatalError := reportFatalErrFn(logger, analyticsReporter, statusReporter)
	// Prepare K8s clients and mapper
	kubeConfig, err := kubex.BuildConfigFromFlags("", conf.Settings.Kubeconfig, conf.Settings.SACredentialsPathPrefix)
	if err != nil {
		return reportFatalError("while loading k8s config", err)
	}
	dynamicCli, discoveryCli, err := getK8sClients(kubeConfig)
	if err != nil {
		return reportFatalError("while getting K8s clients", err)
	}

	// Register current anonymous identity
	k8sCli, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return reportFatalError("while creating K8s clientset", err)
	}
	botkubeVersion, k8sVer, err := findVersions(k8sCli)
	if err = statusReporter.ReportDeploymentConnectionInit(ctx, k8sVer); err != nil {
		return reportFatalError("while reporting botkube connection initialization", err)
	}
	err = analyticsReporter.RegisterCurrentIdentity(ctx, k8sCli, remoteCfg.Identifier)
	if err != nil {
		return reportFatalError("while registering current identity", err)
	}

	statusReporter.SetLogger(logger)
	statusReporter.SetResourceVersion(cfgVersion)
	auditReporter := audit.GetReporter(remoteCfgEnabled, logger, gqlClient)

	ctx, cancel := context.WithCancel(ctx)
	errGroup, ctx := errgroup.WithContext(ctx)
	defer func() {
		// This is because, without cancellation, ctx is still alive and lots of other goroutines will not be
		// deferred and this wait will be stuck infinitely
		cancel()

		multiErr := multierror.New()

		errGroupErr := errGroup.Wait()

		if err != nil && !errors.Is(err, context.Canceled) {
			multiErr = multierror.Append(multiErr, err)
		}

		if errGroupErr != nil && !errors.Is(errGroupErr, context.Canceled) {
			multiErr = multierror.Append(multiErr, errGroupErr)
		}

		if multiErr.ErrorOrNil() == nil {
			return
		}

		err = reportFatalError("while waiting for goroutines to finish gracefully", multiErr.ErrorOrNil())
	}()

	errGroup.Go(func() error {
		err := analyticsReporter.Run(ctx)
		if err != nil {
			logger.Errorf("while closing reporter: %s", err.Error())
		}
		return err
	})

	schedulerChan := make(chan string)
	pluginHealthStats := plugin.NewHealthStats(conf.Plugins.RestartPolicy.Threshold)
	collector := plugin.NewCollector(logger)
	enabledPluginExecutors, enabledPluginSources := collector.GetAllEnabledAndUsedPlugins(conf)
	pluginManager := plugin.NewManager(logger, conf.Settings.Log, conf.Plugins, enabledPluginExecutors, enabledPluginSources, schedulerChan, pluginHealthStats)

	// Health endpoint
	healthChecker := health.NewChecker(ctx, conf, pluginHealthStats)
	healthSrv := healthChecker.NewServer(logger.WithField(componentLogFieldKey, "Health server"), conf.Settings.HealthPort)
	errGroup.Go(func() error {
		defer analytics.ReportPanicIfOccurs(logger, analyticsReporter)
		return healthSrv.Serve(ctx)
	})

	err = pluginManager.Start(ctx)
	if err != nil {
		return fmt.Errorf("while starting plugins manager: %w", err)
	}
	defer pluginManager.Shutdown()

	// Prometheus metrics
	metricsSrv := newMetricsServer(logger.WithField(componentLogFieldKey, "Metrics server"), conf.Settings.MetricsPort)
	errGroup.Go(func() error {
		defer analytics.ReportPanicIfOccurs(logger, analyticsReporter)
		return metricsSrv.Serve(ctx)
	})

	cmdGuard := command.NewCommandGuard(logger.WithField(componentLogFieldKey, "Command Guard"), discoveryCli)
	// Create executor factory
	cfgManager := config.NewManager(remoteCfgEnabled, logger.WithField(componentLogFieldKey, "Config manager"), conf.Settings.PersistentConfig, cfgVersion, k8sCli, gqlClient, deployClient)
	executorFactory, err := execute.NewExecutorFactory(
		execute.DefaultExecutorFactoryParams{
			Log:               logger.WithField(componentLogFieldKey, "Executor"),
			Cfg:               *conf,
			CfgManager:        cfgManager,
			AnalyticsReporter: analyticsReporter,
			CommandGuard:      cmdGuard,
			PluginManager:     pluginManager,
			BotKubeVersion:    botkubeVersion,
			RestCfg:           kubeConfig,
			AuditReporter:     auditReporter,
			PluginHealthStats: pluginHealthStats,
		},
	)
	if err != nil {
		return reportFatalError("while creating executor factory", err)
	}

	var (
		sinkNotifiers []notifier.Sink
		bots          = map[string]bot.Bot{}
	)

	// TODO: Current limitation: Communication platform config should be separate inside every group:
	//    For example, if in both communication groups there's a Slack configuration pointing to the same workspace,
	//	  when user executes `kubectl` command, one Bot instance will execute the command and return response,
	//	  and the second "Sorry, this channel is not authorized to execute kubectl command" error.
	commKeys := maputil.SortKeys(conf.Communications)
	for commGroupIdx, commGroupName := range commKeys {
		commGroupCfg := conf.Communications[commGroupName]

		commGroupLogger := logger.WithField(commGroupFieldKey, commGroupName)
		commGroupMeta := bot.CommGroupMetadata{
			Name:  commGroupName,
			Index: commGroupIdx + 1,
		}

		scheduleBotNotifier := func(in bot.Bot) {
			bots[fmt.Sprintf("%s-%s", commGroupName, in.IntegrationName())] = in
			errGroup.Go(func() error {
				defer analytics.ReportPanicIfOccurs(commGroupLogger, analyticsReporter)
				return in.Start(ctx)
			})
		}

		// Run bots
		if commGroupCfg.Slack.Enabled {
			sb, err := bot.NewSlack(commGroupLogger.WithField(botLogFieldKey, "Slack"), commGroupMeta, commGroupCfg.Slack, executorFactory, analyticsReporter)
			if err != nil {
				return reportFatalError("while creating Slack bot", err)
			}
			scheduleBotNotifier(sb)
		}

		if commGroupCfg.SocketSlack.Enabled {
			sb, err := bot.NewSocketSlack(commGroupLogger.WithField(botLogFieldKey, "SocketSlack"), commGroupMeta, commGroupCfg.SocketSlack, executorFactory, analyticsReporter)
			if err != nil {
				return reportFatalError("while creating SocketSlack bot", err)
			}
			scheduleBotNotifier(sb)
		}

		if commGroupCfg.CloudSlack.Enabled {
			sb, err := bot.NewCloudSlack(commGroupLogger.WithField(botLogFieldKey, "CloudSlack"), commGroupMeta, commGroupCfg.CloudSlack, conf.Settings.ClusterName, executorFactory, analyticsReporter)
			if err != nil {
				return reportFatalError("while creating CloudSlack bot", err)
			}
			scheduleBotNotifier(sb)
		}

		if commGroupCfg.Mattermost.Enabled {
			mb, err := bot.NewMattermost(ctx, commGroupLogger.WithField(botLogFieldKey, "Mattermost"), commGroupMeta, commGroupCfg.Mattermost, executorFactory, analyticsReporter)
			if err != nil {
				return reportFatalError("while creating Mattermost bot", err)
			}
			scheduleBotNotifier(mb)
		}

		if commGroupCfg.Teams.Enabled {
			tb, err := bot.NewTeams(commGroupLogger.WithField(botLogFieldKey, "MS Teams"), commGroupMeta, commGroupCfg.Teams, conf.Settings.ClusterName, executorFactory, analyticsReporter)
			if err != nil {
				return reportFatalError("while creating Teams bot", err)
			}
			scheduleBotNotifier(tb)
		}

		if commGroupCfg.CloudTeams.Enabled {
			ctb, err := bot.NewCloudTeams(commGroupLogger.WithField(botLogFieldKey, "CloudTeams"), commGroupMeta, commGroupCfg.CloudTeams, conf.Settings.ClusterName, executorFactory, analyticsReporter)
			if err != nil {
				return reportFatalError("while creating CloudSlack bot", err)
			}
			scheduleBotNotifier(ctb)
		}

		if commGroupCfg.Discord.Enabled {
			db, err := bot.NewDiscord(commGroupLogger.WithField(botLogFieldKey, "Discord"), commGroupMeta, commGroupCfg.Discord, executorFactory, analyticsReporter)
			if err != nil {
				return reportFatalError("while creating Discord bot", err)
			}
			scheduleBotNotifier(db)
		}

		// Run sinks
		if commGroupCfg.Elasticsearch.Enabled {
			es, err := sink.NewElasticsearch(commGroupLogger.WithField(sinkLogFieldKey, "Elasticsearch"), commGroupMeta.Index, commGroupCfg.Elasticsearch, analyticsReporter)
			if err != nil {
				return reportFatalError("while creating Elasticsearch sink", err)
			}
			sinkNotifiers = append(sinkNotifiers, es)
		}

		if commGroupCfg.Webhook.Enabled {
			wh, err := sink.NewWebhook(commGroupLogger.WithField(sinkLogFieldKey, "Webhook"), commGroupMeta.Index, commGroupCfg.Webhook, analyticsReporter)
			if err != nil {
				return reportFatalError("while creating Webhook sink", err)
			}

			sinkNotifiers = append(sinkNotifiers, wh)
		}
	}
	healthChecker.SetNotifiers(getHealthNotifiers(bots, sinkNotifiers))

	if conf.ConfigWatcher.Enabled {
		restarter := reloader.NewRestarter(
			logger.WithField(componentLogFieldKey, "Restarter"),
			k8sCli,
			conf.ConfigWatcher.Deployment,
			conf.Settings.ClusterName,
			func(msg string) error {
				return notifier.SendPlaintextMessage(ctx, bot.AsNotifiers(bots), msg)
			},
		)

		cfgReloader, err := reloader.Get(
			remoteCfgEnabled,
			logger.WithField(componentLogFieldKey, "Config Reloader"),
			deployClient,
			dynamicCli,
			restarter,
			analyticsReporter,
			*conf,
			cfgVersion,
			cfgManager,
		)
		if err != nil {
			return reportFatalError("while creating config reloader", err)
		}
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, analyticsReporter)
			return cfgReloader.Do(ctx)
		})
	}

	// Send help message
	helpDB := storage.NewForHelp(conf.Settings.SystemConfigMap.Namespace, conf.Settings.SystemConfigMap.Name, k8sCli)
	err = sendHelp(ctx, helpDB, conf.Settings.ClusterName, enabledPluginExecutors, bots)
	if err != nil {
		return fmt.Errorf("while sending initial help message: %w", err)
	}

	// Start upgrade checker
	ghCli := github.NewClient(&http.Client{
		Timeout: 1 * time.Minute,
	})
	if conf.Settings.UpgradeNotifier {
		upgradeChecker := controller.NewUpgradeChecker(
			logger.WithField(componentLogFieldKey, "Upgrade Checker"),
			bots,
			ghCli.Repositories,
		)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, analyticsReporter)
			return upgradeChecker.Run(ctx)
		})
	}

	actionProvider := action.NewProvider(logger.WithField(componentLogFieldKey, "Action Provider"), conf.Actions, executorFactory)

	sourcePluginDispatcher := source.NewDispatcher(logger, conf.Settings.ClusterName, bots, sinkNotifiers, pluginManager, actionProvider, analyticsReporter, auditReporter, kubeConfig)
	scheduler := source.NewScheduler(ctx, logger, conf, sourcePluginDispatcher, schedulerChan)
	err = scheduler.Start(ctx)
	if err != nil {
		return reportFatalError("while starting source plugin event dispatcher: %w", err)
	}

	if conf.Plugins.IncomingWebhook.Enabled {
		incomingWebhookSrv := source.NewIncomingWebhookServer(
			logger.WithField(componentLogFieldKey, "Incoming Webhook Server"),
			conf,
			sourcePluginDispatcher,
			scheduler.StartedSourcePlugins(),
		)

		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, analyticsReporter)
			return incomingWebhookSrv.Serve(ctx)
		})
	}

	// Create and start controller
	ctrl := controller.New(
		logger.WithField(componentLogFieldKey, "Controller"),
		conf,
		bots,
		statusReporter,
	)

	if err := statusReporter.ReportDeploymentStartup(ctx); err != nil {
		return reportFatalError("while reporting botkube startup", err)
	}

	errGroup.Go(func() error {
		if !remoteCfgEnabled {
			logger.Debug("Remote config is not enabled, skipping k8s insights collection...")
			return nil
		}
		defer func() {
			if err == nil {
				return
			}

			reportErr := reportFatalError("while starting k8s collector", err)
			if reportErr != nil {
				logger.Errorf("while reporting fatal error: %s", reportErr.Error())
			}
		}()
		heartbeatReporter := heartbeat.GetReporter(logger, gqlClient)
		k8sCollector := insights.NewK8sCollector(k8sCli, heartbeatReporter, logger, reportHeartbeatInterval, reportHeartbeatMaxRetries)
		return k8sCollector.Start(ctx)
	})

	healthChecker.MarkAsReady()
	err = ctrl.Start(ctx)
	if err != nil {
		return reportFatalError("while starting controller", err)
	}

	err = errGroup.Wait()
	if err != nil {
		// error from errGroup reported on defer, no need to do it twice
		return err
	}

	return nil
}

func newMetricsServer(log logrus.FieldLogger, metricsPort string) *httpx.Server {
	addr := fmt.Sprintf(":%s", metricsPort)
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	return httpx.NewServer(log, addr, router)
}

func getAnalyticsReporter(disableAnalytics bool, logger logrus.FieldLogger) (analytics.Reporter, error) {
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

func getK8sClients(cfg *rest.Config) (dynamic.Interface, discovery.DiscoveryInterface, error) {
	dynamicK8sCli, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("while creating dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("while creating discovery client: %w", err)
	}

	discoCacheClient := memory.NewMemCacheClient(discoveryClient)
	return dynamicK8sCli, discoCacheClient, nil
}

func reportFatalErrFn(logger logrus.FieldLogger, reporter analytics.Reporter, status status.StatusReporter) func(ctx string, err error) error {
	return func(ctx string, err error) error {
		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) {
			logger.Debugf("Context was cancelled. Skipping reporting error...")
			return nil
		}

		// use separate ctx as parent ctx might be cancelled already
		ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		wrappedErr := fmt.Errorf("%s: %w", ctx, err)

		if reportErr := reporter.ReportFatalError(err); reportErr != nil {
			logger.Errorf("while reporting fatal error: %s", err.Error())
		}

		if err := status.ReportDeploymentFailure(ctxTimeout, err.Error()); err != nil {
			logger.Errorf("while reporting deployment failure: %s", err.Error())
		}

		return wrappedErr
	}
}

// sendHelp sends the help message to all interactive bots.
func sendHelp(ctx context.Context, s *storage.Help, clusterName string, executors []string, notifiers map[string]bot.Bot) error {
	alreadySentHelp, err := s.GetSentHelpDetails(ctx)
	if err != nil {
		return fmt.Errorf("while getting the help data: %w", err)
	}

	var sent []string

	for key, notifier := range notifiers {
		if alreadySentHelp[key] {
			continue
		}

		help := interactive.NewHelpMessage(notifier.IntegrationName(), clusterName, executors).Build()
		err := notifier.SendMessageToAll(ctx, help)
		if err != nil {
			return fmt.Errorf("while sending help message for %s: %w", notifier.IntegrationName(), err)
		}
		sent = append(sent, key)
	}

	return s.MarkHelpAsSent(ctx, sent)
}

func findVersions(cli *kubernetes.Clientset) (string, string, error) {
	k8sVer, err := cli.ServerVersion()
	if err != nil {
		return "", "", fmt.Errorf("while getting server version: %w", err)
	}

	botkubeVersion := version.Short()
	if len(botkubeVersion) == 0 {
		botkubeVersion = "Unknown"
	}

	return fmt.Sprintf("K8s Server Version: %s\nBotkube version: %s", k8sVer.String(), botkubeVersion), k8sVer.String(), nil
}

func getHealthNotifiers(bots map[string]bot.Bot, sinks []notifier.Sink) map[string]health.Notifier {
	notifiers := make(map[string]health.Notifier)
	for key, botInstance := range bots {
		notifiers[key] = botInstance
	}
	for key, sinkInstance := range sinks {
		notifiers[fmt.Sprintf("%s-%d", sinkInstance.IntegrationName(), key)] = sinkInstance
	}

	return notifiers
}
