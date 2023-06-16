package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/v44/github"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	segment "github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
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
	"github.com/kubeshop/botkube/internal/heartbeat"
	"github.com/kubeshop/botkube/internal/insights"
	"github.com/kubeshop/botkube/internal/kubex"
	"github.com/kubeshop/botkube/internal/lifecycle"
	"github.com/kubeshop/botkube/internal/loggerx"
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
	"github.com/kubeshop/botkube/pkg/httpsrv"
	"github.com/kubeshop/botkube/pkg/notifier"
	"github.com/kubeshop/botkube/pkg/sink"
	"github.com/kubeshop/botkube/pkg/version"
)

const (
	componentLogFieldKey      = "component"
	botLogFieldKey            = "bot"
	sinkLogFieldKey           = "sink"
	commGroupFieldKey         = "commGroup"
	healthEndpointName        = "/healthz"
	printAPIKeyCharCount      = 3
	reportHeartbeatInterval   = 10
	reportHeartbeatMaxRetries = 30
)

func main() {
	// Set up context
	ctx := signals.SetupSignalHandler()
	ctx, cancelCtxFn := context.WithCancel(ctx)
	defer cancelCtxFn()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

// run wraps the main logic of the app to be able to properly clean up resources via deferred calls.
func run(ctx context.Context) error {
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

	cfgProvider := intconfig.GetProvider(remoteCfgEnabled, deployClient)
	configs, cfgVersion, err := cfgProvider.Configs(ctx)
	if err != nil {
		return fmt.Errorf("while loading configuration files: %w", err)
	}

	conf, confDetails, err := config.LoadWithDefaults(configs)
	if err != nil {
		return fmt.Errorf("while merging app configuration: %w", err)
	}

	logger := loggerx.New(conf.Settings.Log)
	if confDetails.ValidateWarnings != nil {
		logger.Warnf("Configuration validation warnings: %v", confDetails.ValidateWarnings.Error())
	}

	statusReporter := status.GetReporter(remoteCfgEnabled, logger, gqlClient, deployClient, cfgVersion)
	auditReporter := audit.GetReporter(remoteCfgEnabled, logger, gqlClient)

	// Set up analytics reporter
	reporter, err := getAnalyticsReporter(conf.Analytics.Disable, logger)
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

	reportFatalError := reportFatalErrFn(logger, reporter, statusReporter)
	errGroup, ctx := errgroup.WithContext(ctx)
	defer func() {
		err := errGroup.Wait()
		wrappedErr := reportFatalError("while waiting for goroutines to finish gracefully", err)
		if wrappedErr != nil {
			logger.Error(wrappedErr.Error())
		}
	}()

	collector := plugin.NewCollector(logger)
	enabledPluginExecutors, enabledPluginSources := collector.GetAllEnabledAndUsedPlugins(conf)
	pluginManager := plugin.NewManager(logger, conf.Plugins, enabledPluginExecutors, enabledPluginSources)

	err = pluginManager.Start(ctx)
	if err != nil {
		return fmt.Errorf("while starting plugins manager: %w", err)
	}
	defer pluginManager.Shutdown()

	// Prepare K8s clients and mapper
	kubeConfig, err := kubex.BuildConfigFromFlags("", conf.Settings.Kubeconfig, conf.Settings.SACredentialsPathPrefix)
	if err != nil {
		return reportFatalError("while loading k8s config", err)
	}
	discoveryCli, err := getK8sClients(kubeConfig)
	if err != nil {
		return reportFatalError("while getting K8s clients", err)
	}

	// Register current anonymous identity
	k8sCli, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return reportFatalError("while creating K8s clientset", err)
	}
	err = reporter.RegisterCurrentIdentity(ctx, k8sCli, remoteCfg.Identifier)
	if err != nil {
		return reportFatalError("while registering current identity", err)
	}

	// Health endpoint
	healthChecker := healthChecker{applicationStarted: false}
	healthSrv := newHealthServer(logger.WithField(componentLogFieldKey, "Health server"), conf.Settings.HealthPort, &healthChecker)
	errGroup.Go(func() error {
		defer analytics.ReportPanicIfOccurs(logger, reporter)
		return healthSrv.Serve(ctx)
	})

	// Prometheus metrics
	metricsSrv := newMetricsServer(logger.WithField(componentLogFieldKey, "Metrics server"), conf.Settings.MetricsPort)
	errGroup.Go(func() error {
		defer analytics.ReportPanicIfOccurs(logger, reporter)
		return metricsSrv.Serve(ctx)
	})

	cmdGuard := command.NewCommandGuard(logger.WithField(componentLogFieldKey, "Command Guard"), discoveryCli)
	botkubeVersion, err := findVersions(k8sCli)
	if err != nil {
		return reportFatalError("while fetching versions", err)
	}
	// Create executor factory
	cfgManager := config.NewManager(remoteCfgEnabled, logger.WithField(componentLogFieldKey, "Config manager"), conf.Settings.PersistentConfig, cfgVersion, k8sCli, gqlClient, deployClient)
	executorFactory, err := execute.NewExecutorFactory(
		execute.DefaultExecutorFactoryParams{
			Log:               logger.WithField(componentLogFieldKey, "Executor"),
			Cfg:               *conf,
			CfgManager:        cfgManager,
			AnalyticsReporter: reporter,
			CommandGuard:      cmdGuard,
			PluginManager:     pluginManager,
			BotKubeVersion:    botkubeVersion,
			RestCfg:           kubeConfig,
			AuditReporter:     auditReporter,
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
	for commGroupName, commGroupCfg := range conf.Communications {
		commGroupLogger := logger.WithField(commGroupFieldKey, commGroupName)

		scheduleBotNotifier := func(in bot.Bot) {
			bots[fmt.Sprintf("%s-%s", commGroupName, in.IntegrationName())] = in
			errGroup.Go(func() error {
				defer analytics.ReportPanicIfOccurs(commGroupLogger, reporter)
				return in.Start(ctx)
			})
		}

		// Run bots
		if commGroupCfg.Slack.Enabled {
			sb, err := bot.NewSlack(commGroupLogger.WithField(botLogFieldKey, "Slack"), commGroupName, commGroupCfg.Slack, executorFactory, reporter)
			if err != nil {
				return reportFatalError("while creating Slack bot", err)
			}
			scheduleBotNotifier(sb)
		}

		if commGroupCfg.SocketSlack.Enabled {
			sb, err := bot.NewSocketSlack(commGroupLogger.WithField(botLogFieldKey, "SocketSlack"), commGroupName, commGroupCfg.SocketSlack, executorFactory, reporter)
			if err != nil {
				return reportFatalError("while creating SocketSlack bot", err)
			}
			scheduleBotNotifier(sb)
		}

		if commGroupCfg.CloudSlack.Enabled {
			sb, err := bot.NewCloudSlack(commGroupLogger.WithField(botLogFieldKey, "CloudSlack"), commGroupName, commGroupCfg.CloudSlack, conf.Settings.ClusterName, executorFactory, reporter)
			if err != nil {
				return reportFatalError("while creating CloudSlack bot", err)
			}
			scheduleBotNotifier(sb)
		}

		if commGroupCfg.Mattermost.Enabled {
			mb, err := bot.NewMattermost(commGroupLogger.WithField(botLogFieldKey, "Mattermost"), commGroupName, commGroupCfg.Mattermost, executorFactory, reporter)
			if err != nil {
				return reportFatalError("while creating Mattermost bot", err)
			}
			scheduleBotNotifier(mb)
		}

		if commGroupCfg.Teams.Enabled {
			tb, err := bot.NewTeams(commGroupLogger.WithField(botLogFieldKey, "MS Teams"), commGroupName, commGroupCfg.Teams, conf.Settings.ClusterName, executorFactory, reporter)
			if err != nil {
				return reportFatalError("while creating Teams bot", err)
			}
			scheduleBotNotifier(tb)
		}

		if commGroupCfg.Discord.Enabled {
			db, err := bot.NewDiscord(commGroupLogger.WithField(botLogFieldKey, "Discord"), commGroupName, commGroupCfg.Discord, executorFactory, reporter)
			if err != nil {
				return reportFatalError("while creating Discord bot", err)
			}
			scheduleBotNotifier(db)
		}

		// Run sinks
		if commGroupCfg.Elasticsearch.Enabled {
			es, err := sink.NewElasticsearch(commGroupLogger.WithField(sinkLogFieldKey, "Elasticsearch"), commGroupCfg.Elasticsearch, reporter)
			if err != nil {
				return reportFatalError("while creating Elasticsearch sink", err)
			}
			sinkNotifiers = append(sinkNotifiers, es)
		}

		if commGroupCfg.Webhook.Enabled {
			wh, err := sink.NewWebhook(commGroupLogger.WithField(sinkLogFieldKey, "Webhook"), commGroupCfg.Webhook, reporter)
			if err != nil {
				return reportFatalError("while creating Webhook sink", err)
			}

			sinkNotifiers = append(sinkNotifiers, wh)
		}
	}

	// TODO(https://github.com/kubeshop/botkube/issues/1011): Move restarter under `if conf.ConfigWatcher.Enabled {`
	restarter := reloader.NewRestarter(
		logger.WithField(componentLogFieldKey, "Restarter"),
		k8sCli,
		conf.ConfigWatcher.Deployment,
		conf.Settings.ClusterName,
		func(msg string) error {
			return notifier.SendPlaintextMessage(ctx, bot.AsNotifiers(bots), msg)
		},
	)
	if conf.ConfigWatcher.Enabled {
		cfgReloader := reloader.Get(
			remoteCfgEnabled,
			logger.WithField(componentLogFieldKey, "Config Updater"),
			deployClient,
			restarter,
			*conf,
			cfgVersion,
			statusReporter,
			cfgManager,
		)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return cfgReloader.Do(ctx)
		})

		// TODO(https://github.com/kubeshop/botkube/issues/1011): Remove once we migrate to ConfigMap-based config reloader
		err := config.WaitForWatcherSync(
			ctx,
			logger.WithField(componentLogFieldKey, "Config Watcher Sync"),
			conf.ConfigWatcher,
		)
		if err != nil {
			if err != wait.ErrWaitTimeout {
				return reportFatalError("while waiting for Config Watcher sync", err)
			}

			// non-blocking error, move forward
			logger.Warn("Config Watcher is still not synchronized. Read the logs of the sidecar container to see the cause. Continuing running Botkube...")
		}
	}

	// TODO(https://github.com/kubeshop/botkube/issues/1011): Remove once we migrate to ConfigMap-based config reloader
	// Lifecycle server
	if conf.Settings.LifecycleServer.Enabled {
		lifecycleSrv := lifecycle.NewServer(
			logger.WithField(componentLogFieldKey, "Lifecycle server"),
			conf.Settings.LifecycleServer,
			restarter,
		)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return lifecycleSrv.Serve(ctx)
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
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return upgradeChecker.Run(ctx)
		})
	}

	actionProvider := action.NewProvider(logger.WithField(componentLogFieldKey, "Action Provider"), conf.Actions, executorFactory)

	sourcePluginDispatcher := source.NewDispatcher(logger, bots, sinkNotifiers, pluginManager, actionProvider, reporter, auditReporter, kubeConfig)
	scheduler := source.NewScheduler(logger, conf, sourcePluginDispatcher)
	err = scheduler.Start(ctx)
	if err != nil {
		return fmt.Errorf("while starting source plugin event dispatcher: %w", err)
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

func newMetricsServer(log logrus.FieldLogger, metricsPort string) *httpsrv.Server {
	addr := fmt.Sprintf(":%s", metricsPort)
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	return httpsrv.New(log, addr, router)
}

func newHealthServer(log logrus.FieldLogger, port string, healthChecker *healthChecker) *httpsrv.Server {
	addr := fmt.Sprintf(":%s", port)
	router := mux.NewRouter()
	router.Handle(healthEndpointName, healthChecker)
	return httpsrv.New(log, addr, router)
}

type healthChecker struct {
	applicationStarted bool
}

func (h *healthChecker) MarkAsReady() {
	h.applicationStarted = true
}

func (h *healthChecker) IsReady() bool {
	return h.applicationStarted
}

func (h *healthChecker) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if h.IsReady() {
		resp.WriteHeader(http.StatusOK)
		fmt.Fprint(resp, "ok")
	} else {
		resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(resp, "unavailable")
	}
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

func getK8sClients(cfg *rest.Config) (discovery.DiscoveryInterface, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("while creating discovery client: %w", err)
	}

	discoCacheClient := memory.NewMemCacheClient(discoveryClient)
	return discoCacheClient, nil
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

func findVersions(cli *kubernetes.Clientset) (string, error) {
	k8sVer, err := cli.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("while getting server version: %w", err)
	}

	botkubeVersion := version.Short()
	if len(botkubeVersion) == 0 {
		botkubeVersion = "Unknown"
	}

	return fmt.Sprintf("K8s Server Version: %s\nBotkube version: %s", k8sVer.String(), botkubeVersion), nil
}
