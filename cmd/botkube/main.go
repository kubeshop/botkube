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
	"os"

	"golang.org/x/sync/errgroup"

	"github.com/infracloudio/botkube/pkg/bot"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/controller"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/metrics"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
)

const (
	defaultMetricsPort = "2112"
)

// TODO:
// 	- Use context to make sure all goroutines shutdowns gracefully: https://github.com/infracloudio/botkube/issues/220
//  - Make the code testable (shorten methods and functions, and reduce level of cyclomatic complexity): https://github.com/infracloudio/botkube/issues/589

func main() {
	// Prometheus metrics
	metricsPort, exists := os.LookupEnv("METRICS_PORT")
	if !exists {
		metricsPort = defaultMetricsPort
	}

	// Set up global logger and filter engine
	log.SetupGlobal()
	filterengine.SetupGlobal()

	errGroup := new(errgroup.Group)

	errGroup.Go(func() error {
		return metrics.ServeMetrics(metricsPort)
	})

	log.Info("Starting controller")

	conf, err := config.New()
	exitOnError(err, "while loading configuration")

	// List notifiers
	notifiers := notify.ListNotifiers(conf.Communications)

	if conf.Communications.Slack.Enabled {
		log.Info("Starting slack bot")
		sb := bot.NewSlackBot(conf)
		errGroup.Go(func() error {
			return sb.Start()
		})
	}

	if conf.Communications.Mattermost.Enabled {
		log.Info("Starting mattermost bot")
		mb := bot.NewMattermostBot(conf)
		errGroup.Go(func() error {
			return mb.Start()
		})
	}

	if conf.Communications.Teams.Enabled {
		log.Info("Starting MS Teams bot")
		tb := bot.NewTeamsBot(conf)
		notifiers = append(notifiers, tb)
		errGroup.Go(func() error {
			return tb.Start()
		})
	}

	if conf.Communications.Discord.Enabled {
		log.Info("Starting discord bot")
		db := bot.NewDiscordBot(conf)
		errGroup.Go(func() error {
			return db.Start()
		})
	}

	if conf.Communications.Lark.Enabled {
		log.Info("Starting lark bot")
		lb := bot.NewLarkBot(conf)
		errGroup.Go(func() error {
			return lb.Start()
		})
	}

	// Start upgrade notifier
	if conf.Settings.UpgradeNotifier {
		log.Info("Starting upgrade notifier")
		errGroup.Go(func() error {
			controller.UpgradeNotifier(notifiers)
			return nil
		})
	}

	// Init KubeClient, InformerMap and start controller
	utils.InitKubeClient()
	utils.InitInformerMap(conf)
	utils.InitResourceMap(conf)
	controller.RegisterInformers(conf, notifiers)

	err = errGroup.Wait()
	exitOnError(err, "while waiting for goroutines to finish gracefully")
}

func exitOnError(err error, context string) {
	if err != nil {
		log.Fatalf("%s: %v", context, err)
	}
}
