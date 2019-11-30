package main

import (
	"fmt"
	"os"

	"github.com/infracloudio/botkube/pkg/bot"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/controller"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/infracloudio/botkube/pkg/metrics"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
)

const (
	defaultMetricsPort = "2112"
)

func main() {
	log.Logger.Info("Starting controller")
	Config, err := config.New()
	if err != nil {
		log.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}

	if Config.Communications.Slack.Enabled {
		log.Logger.Info("Starting slack bot")
		sb := bot.NewSlackBot()
		go sb.Start()
	}

	if Config.Communications.Mattermost.Enabled {
		log.Logger.Info("Starting mattermost bot")
		mb := bot.NewMattermostBot()
		go mb.Start()
	}

	// Prometheus metrics
	metricsPort, exists := os.LookupEnv("METRICS_PORT")
	if !exists {
		metricsPort = defaultMetricsPort
	}
	go metrics.ServeMetrics(metricsPort)

	// List notifiers
	var notifiers []notify.Notifier
	if Config.Communications.Slack.Enabled {
		notifiers = append(notifiers, notify.NewSlack(Config))
	}
	if Config.Communications.Mattermost.Enabled {
		if notifier, err := notify.NewMattermost(Config); err == nil {
			notifiers = append(notifiers, notifier)
		}
	}
	if Config.Communications.ElasticSearch.Enabled {
		notifiers = append(notifiers, notify.NewElasticSearch(Config))
	}
	if Config.Communications.Webhook.Enabled {
		notifiers = append(notifiers, notify.NewWebhook(Config))
	}
	if Config.Settings.UpgradeNotifier {
		log.Logger.Info("Starting upgrade notifier")
		go controller.UpgradeNotifier(Config, notifiers)
	}

	// Init KubeClient, InformerMap and start controller
	utils.InitKubeClient()
	utils.InitInformerMap()
	controller.RegisterInformers(Config, notifiers)
}
