package main

import (
	"context"
	"encoding/json"
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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/strings"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/kubeshop/botkube/internal/analytics"
	intconfig "github.com/kubeshop/botkube/internal/config"
	"github.com/kubeshop/botkube/internal/lifecycle"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/internal/source"
	"github.com/kubeshop/botkube/internal/storage"
	"github.com/kubeshop/botkube/pkg/action"
	"github.com/kubeshop/botkube/pkg/bot"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/controller"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
	"github.com/kubeshop/botkube/pkg/filterengine"
	"github.com/kubeshop/botkube/pkg/httpsrv"
	"github.com/kubeshop/botkube/pkg/notifier"
	"github.com/kubeshop/botkube/pkg/recommendation"
	"github.com/kubeshop/botkube/pkg/sink"
	"github.com/kubeshop/botkube/pkg/version"
)

const (
	componentLogFieldKey = "component"
	botLogFieldKey       = "bot"
	sinkLogFieldKey      = "sink"
	commGroupFieldKey    = "commGroup"
	healthEndpointName   = "/healthz"
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
	var gqlClient intconfig.GqlClient = intconfig.NewGqlClient(intconfig.WithAPIURL(os.Getenv("CONFIG_PROVIDER_ENDPOINT")))
	configs, err := config.FromProvider(&gqlClient)
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

	collector := plugin.NewCollector(logger)
	enabledPluginExecutors, enabledPluginSources := collector.GetAllEnabledAndUsedPlugins(conf)
	pluginManager := plugin.NewManager(logger, conf.Plugins, enabledPluginExecutors, enabledPluginSources)

	err = pluginManager.Start(ctx)
	if err != nil {
		return fmt.Errorf("while starting plugins manager: %w", err)
	}
	defer pluginManager.Shutdown()

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
	err = reporter.RegisterCurrentIdentity(ctx, k8sCli)
	if err != nil {
		return reportFatalError("while registering current identity", err)
	}

	// Health endpoint
	healthSrv := newHealthServer(logger.WithField(componentLogFieldKey, "Health server"), conf.Settings.HealthPort)
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

	// Set up the filter engine
	filterEngine := filterengine.WithAllFilters(logger, dynamicCli, mapper, conf.Filters)

	// Kubectl config merger
	kcMerger := kubectl.NewMerger(conf.Executors)

	// Load resource variants name if needed
	var resourceNameNormalizerFunc kubectl.ResourceVariantsFunc
	if kcMerger.IsAtLeastOneEnabled() {
		resourceNameNormalizer, err := kubectl.NewResourceNormalizer(
			logger.WithField(componentLogFieldKey, "Resource Name Normalizer"),
			discoveryCli,
		)
		if err != nil {
			return reportFatalError("while creating resource name normalizer", err)
		}
		resourceNameNormalizerFunc = resourceNameNormalizer.Normalize
	}

	cmdGuard := kubectl.NewCommandGuard(logger.WithField(componentLogFieldKey, "Command Guard"), discoveryCli)
	commander := kubectl.NewCommander(logger.WithField(componentLogFieldKey, "Commander"), kcMerger, cmdGuard)

	runner := &execute.OSCommand{}
	k8sVersion, err := findK8sVersion(runner)
	if err != nil {
		return reportFatalError("while fetching kubernetes version", err)
	}
	botkubeVersion := findBotkubeVersion(k8sVersion)

	// Create executor factory
	cfgManager := config.NewManager(logger.WithField(componentLogFieldKey, "Config manager"), conf.Settings.PersistentConfig, k8sCli)
	executorFactory, err := execute.NewExecutorFactory(
		execute.DefaultExecutorFactoryParams{
			Log:               logger.WithField(componentLogFieldKey, "Executor"),
			CmdRunner:         runner,
			Cfg:               *conf,
			FilterEngine:      filterEngine,
			KcChecker:         kubectl.NewChecker(resourceNameNormalizerFunc),
			Merger:            kcMerger,
			CfgManager:        cfgManager,
			AnalyticsReporter: reporter,
			NamespaceLister:   k8sCli.CoreV1().Namespaces(),
			CommandGuard:      cmdGuard,
			PluginManager:     pluginManager,
			BotKubeVersion:    botkubeVersion,
		},
	)

	if err != nil {
		return reportFatalError("while creating executor factory", err)
	}

	router := source.NewRouter(mapper, dynamicCli, logger.WithField(componentLogFieldKey, "Router"))

	var (
		notifiers []notifier.Notifier
		bots      = map[string]bot.Bot{}
	)

	// TODO: Current limitation: Communication platform config should be separate inside every group:
	//    For example, if in both communication groups there's a Slack configuration pointing to the same workspace,
	//	  when user executes `kubectl` command, one Bot instance will execute the command and return response,
	//	  and the second "Sorry, this channel is not authorized to execute kubectl command" error.
	for commGroupName, commGroupCfg := range conf.Communications {
		commGroupLogger := logger.WithField(commGroupFieldKey, commGroupName)

		router.AddCommunicationsBindings(commGroupCfg)

		scheduleBot := func(in bot.Bot) {
			notifiers = append(notifiers, in)
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
			scheduleBot(sb)
		}

		if commGroupCfg.SocketSlack.Enabled {
			sb, err := bot.NewSocketSlack(commGroupLogger.WithField(botLogFieldKey, "SocketSlack"), commGroupName, commGroupCfg.SocketSlack, executorFactory, commander, reporter)
			if err != nil {
				return reportFatalError("while creating SocketSlack bot", err)
			}
			scheduleBot(sb)
		}

		if commGroupCfg.Mattermost.Enabled {
			mb, err := bot.NewMattermost(commGroupLogger.WithField(botLogFieldKey, "Mattermost"), commGroupName, commGroupCfg.Mattermost, executorFactory, reporter)
			if err != nil {
				return reportFatalError("while creating Mattermost bot", err)
			}
			scheduleBot(mb)
		}

		if commGroupCfg.Teams.Enabled {
			tb, err := bot.NewTeams(commGroupLogger.WithField(botLogFieldKey, "MS Teams"), commGroupName, commGroupCfg.Teams, conf.Settings.ClusterName, executorFactory, reporter)
			if err != nil {
				return reportFatalError("while creating Teams bot", err)
			}
			scheduleBot(tb)
		}

		if commGroupCfg.Discord.Enabled {
			db, err := bot.NewDiscord(commGroupLogger.WithField(botLogFieldKey, "Discord"), commGroupName, commGroupCfg.Discord, executorFactory, reporter)
			if err != nil {
				return reportFatalError("while creating Discord bot", err)
			}
			scheduleBot(db)
		}

		// Run sinks
		if commGroupCfg.Elasticsearch.Enabled {
			es, err := sink.NewElasticsearch(commGroupLogger.WithField(sinkLogFieldKey, "Elasticsearch"), commGroupCfg.Elasticsearch, reporter)
			if err != nil {
				return reportFatalError("while creating Elasticsearch sink", err)
			}
			notifiers = append(notifiers, es)
		}

		if commGroupCfg.Webhook.Enabled {
			wh, err := sink.NewWebhook(commGroupLogger.WithField(sinkLogFieldKey, "Webhook"), commGroupCfg.Webhook, reporter)
			if err != nil {
				return reportFatalError("while creating Webhook sink", err)
			}

			notifiers = append(notifiers, wh)
		}
	}

	// Lifecycle server
	if conf.Settings.LifecycleServer.Enabled {
		lifecycleSrv := lifecycle.NewServer(
			logger.WithField(componentLogFieldKey, "Lifecycle server"),
			k8sCli,
			conf.Settings.LifecycleServer,
			conf.Settings.ClusterName,
			func(msg string) error {
				return notifier.SendPlaintextMessage(ctx, notifiers, msg)
			},
		)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return lifecycleSrv.Serve(ctx)
		})
	}

	if conf.ConfigWatcher.Enabled {
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
			notifiers,
			ghCli.Repositories,
		)
		errGroup.Go(func() error {
			defer analytics.ReportPanicIfOccurs(logger, reporter)
			return upgradeChecker.Run(ctx)
		})
	}

	recommFactory := recommendation.NewFactory(logger.WithField(componentLogFieldKey, "Recommendations"), dynamicCli)

	actionProvider := action.NewProvider(logger.WithField(componentLogFieldKey, "Action Provider"), conf.Actions, executorFactory)
	router.AddEnabledActionBindings(conf.Actions)

	sourcePluginDispatcher := source.NewDispatcher(logger, notifiers, pluginManager)
	scheduler := source.NewScheduler(logger, conf, sourcePluginDispatcher)
	err = scheduler.Start(ctx)
	if err != nil {
		return fmt.Errorf("while starting source plugin event dispatcher: %w", err)
	}

	// Create and start controller
	ctrl := controller.New(
		logger.WithField(componentLogFieldKey, "Controller"),
		conf,
		notifiers,
		recommFactory,
		filterEngine,
		dynamicCli,
		mapper,
		conf.Settings.InformersResyncPeriod,
		router.BuildTable(conf),
		actionProvider,
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

func newMetricsServer(log logrus.FieldLogger, metricsPort string) *httpsrv.Server {
	addr := fmt.Sprintf(":%s", metricsPort)
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	return httpsrv.New(log, addr, router)
}

func newHealthServer(log logrus.FieldLogger, port string) *httpsrv.Server {
	addr := fmt.Sprintf(":%s", port)
	router := mux.NewRouter()
	router.Handle(healthEndpointName, healthChecker{})
	return httpsrv.New(log, addr, router)
}

type healthChecker struct{}

func (healthChecker) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	fmt.Fprint(resp, "ok")
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

	discoCacheClient := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoCacheClient)
	return dynamicK8sCli, discoCacheClient, mapper, nil
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

		help := interactive.NewHelpMessage(notifier.IntegrationName(), clusterName, notifier.BotName(), executors).Build()
		err := notifier.SendMessageToAll(ctx, help)
		if err != nil {
			return fmt.Errorf("while sending help message for %s: %w", notifier.IntegrationName(), err)
		}
		sent = append(sent, key)
	}

	return s.MarkHelpAsSent(ctx, sent)
}

func findK8sVersion(runner *execute.OSCommand) (string, error) {
	type kubectlVersionOutput struct {
		Server struct {
			GitVersion string `json:"gitVersion"`
		} `json:"serverVersion"`
	}

	args := []string{"-c", fmt.Sprintf("%s version --output=json", execute.KubectlBinary)}
	stdout, stderr, err := runner.RunSeparateOutput("sh", args)
	if err != nil {
		return "", fmt.Errorf("unable to execute kubectl version: %w [%q]", err, stderr)
	}

	var out kubectlVersionOutput
	err = json.Unmarshal([]byte(stdout), &out)
	if err != nil {
		return "", err
	}
	if out.Server.GitVersion == "" {
		return "", fmt.Errorf("unable to unmarshal server git version from %q", stdout)
	}

	ver := out.Server.GitVersion
	if stderr != "" {
		ver += "\n" + stderr
	}

	return ver, nil
}

func findBotkubeVersion(k8sVersion string) (versions string) {
	botkubeVersion := version.Short()
	if len(botkubeVersion) == 0 {
		botkubeVersion = "Unknown"
	}

	return fmt.Sprintf("K8s Server Version: %s\nBotkube version: %s", k8sVersion, botkubeVersion)
}
