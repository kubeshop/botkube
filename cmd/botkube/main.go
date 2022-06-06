// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/infracloudio/botkube/pkg/bot"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/controller"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/httpsrv"
	"github.com/infracloudio/botkube/pkg/kube"
	"github.com/infracloudio/botkube/pkg/notify"
)

// Config contains the app configuration parameters.
type Config struct {
	MetricsPort           string        `envconfig:"default=2112"`
	LogLevel              string        `envconfig:"default=error"`
	ConfigPath            string        `envconfig:"optional"`
	InformersResyncPeriod time.Duration `envconfig:"default=30m"`
	KubeconfigPath        string        `envconfig:"optional,KUBECONFIG"`
	LogForceColors        bool          `envconfig:"optional"`
}

const (
	componentLogFieldKey = "component"
	botLogFieldKey       = "bot"
)

func main() {
	var appCfg Config
	err := envconfig.Init(&appCfg)
	exitOnError(err, "while loading app configuration")

	logger := newLogger(appCfg.LogLevel, appCfg.LogForceColors)
	ctx := signals.SetupSignalHandler()
	ctx, cancelCtxFn := context.WithCancel(ctx)
	defer cancelCtxFn()

	errGroup, ctx := errgroup.WithContext(ctx)

	// Prometheus metrics
	metricsSrv := newMetricsServer(logger.WithField(componentLogFieldKey, "Metrics server"), appCfg.MetricsPort)
	errGroup.Go(func() error {
		return metricsSrv.Serve(ctx)
	})

	conf, err := config.Load(appCfg.ConfigPath)
	exitOnError(err, "while loading configuration")
	if conf == nil {
		log.Fatal("while loading configuration: config cannot be nil")
	}

	// Prepare K8s clients and mapper
	dynamicCli, discoveryCli, mapper, err := kube.SetupK8sClients(appCfg.KubeconfigPath)
	exitOnError(err, "while initializing K8s clients")

	// Set up the filter engine
	filterEngine := filterengine.WithAllFilters(logger, dynamicCli, mapper, conf)

	// List notifiers
	notifiers, err := notify.LoadNotifiers(logger, conf.Communications)
	exitOnError(err, "while loading notifiers")

	// Create Executor Factory
	resMapping, err := execute.LoadResourceMappingIfShould(
		logger.WithField(componentLogFieldKey, "Resource Mapping Loader"),
		conf,
		discoveryCli,
	)
	exitOnError(err, "while loading resource mapping")

	executorFactory := execute.NewExecutorFactory(
		logger.WithField(componentLogFieldKey, "Executor"),
		execute.DefaultCommandRunnerFunc,
		*conf,
		filterEngine,
		resMapping,
	)

	// Run bots
	if conf.Communications.Slack.Enabled {
		sb := bot.NewSlackBot(logger.WithField(botLogFieldKey, "Slack"), conf, executorFactory)
		errGroup.Go(func() error {
			return sb.Start(ctx)
		})
	}

	if conf.Communications.Mattermost.Enabled {
		mb := bot.NewMattermostBot(logger.WithField(botLogFieldKey, "Mattermost"), conf, executorFactory)
		errGroup.Go(func() error {
			return mb.Start(ctx)
		})
	}

	if conf.Communications.Teams.Enabled {
		tb := bot.NewTeamsBot(logger.WithField(botLogFieldKey, "MS Teams"), conf, executorFactory)
		// TODO: Unify that with other notifiers: Split this into two structs or merge other bots and notifiers into single structs
		notifiers = append(notifiers, tb)
		errGroup.Go(func() error {
			return tb.Start(ctx)
		})
	}

	if conf.Communications.Discord.Enabled {
		db := bot.NewDiscordBot(logger.WithField(botLogFieldKey, "Discord"), conf, executorFactory)
		errGroup.Go(func() error {
			return db.Start(ctx)
		})
	}

	if conf.Communications.Lark.Enabled {
		lb := bot.NewLarkBot(logger.WithField(botLogFieldKey, "Lark"), logger.GetLevel(), conf, executorFactory)
		errGroup.Go(func() error {
			return lb.Start(ctx)
		})
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
			return upgradeChecker.Run(ctx)
		})
	}

	// Start Config Watcher
	if conf.Settings.ConfigWatcher {
		cfgWatcher := controller.NewConfigWatcher(
			logger.WithField(componentLogFieldKey, "Config Watcher"),
			appCfg.ConfigPath,
			conf.Settings.ClusterName,
			notifiers,
		)
		errGroup.Go(func() error {
			return cfgWatcher.Do(ctx, cancelCtxFn)
		})
	}

	// Start controller

	ctrl := controller.New(
		logger.WithField(componentLogFieldKey, "Controller"),
		conf,
		notifiers,
		filterEngine,
		appCfg.ConfigPath,
		dynamicCli,
		mapper,
		appCfg.InformersResyncPeriod,
	)

	err = ctrl.Start(ctx)
	exitOnError(err, "while starting controller")

	err = errGroup.Wait()
	exitOnError(err, "while waiting for goroutines to finish gracefully")
}

func newLogger(logLevelStr string, logForceColors bool) *logrus.Logger {
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
	logger.Formatter = &logrus.TextFormatter{ForceColors: logForceColors, FullTimestamp: true}

	return logger
}

func newMetricsServer(log logrus.FieldLogger, metricsPort string) *httpsrv.Server {
	addr := fmt.Sprintf(":%s", metricsPort)
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	return httpsrv.New(log, addr, router)
}

func exitOnError(err error, context string) {
	if err != nil {
		log.Fatalf("%s: %v", context, err)
	}
}
