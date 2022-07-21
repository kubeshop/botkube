package notifier

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

const notifierLogFieldKey = "notifier"

// Notifier to send event notification on the communication channels
type Notifier interface {
	SendEvent(context.Context, events.Event) error
	SendMessage(context.Context, string) error
	IntegrationName() config.CommPlatformIntegration
	Type() config.IntegrationType
}

// SinkAnalyticsReporter defines a reporter that collects analytics data.
type SinkAnalyticsReporter interface {
	// ReportSinkEnabled reports an enabled sink.
	ReportSinkEnabled(platform config.CommPlatformIntegration) error
}

// LoadNotifiers returns list of configured notifiers
func LoadNotifiers(log logrus.FieldLogger, conf config.CommunicationsConfig, reporter analytics.Reporter) ([]Notifier, error) {
	var notifiers []Notifier
	if conf.Slack.Enabled {
		notifiers = append(notifiers, NewSlack(log.WithField(notifierLogFieldKey, "Slack"), conf.Slack))
	}

	if conf.Mattermost.Enabled {
		mmNotifier, err := NewMattermost(log.WithField(notifierLogFieldKey, "Mattermost"), conf.Mattermost)
		if err != nil {
			return nil, fmt.Errorf("while creating Mattermost client: %w", err)
		}

		notifiers = append(notifiers, mmNotifier)
	}

	if conf.Discord.Enabled {
		dNotifier, err := NewDiscord(log.WithField(notifierLogFieldKey, "Discord"), conf.Discord)
		if err != nil {
			return nil, fmt.Errorf("while creating Discord notifier: %w", err)
		}

		notifiers = append(notifiers, dNotifier)
	}

	if conf.Elasticsearch.Enabled {
		esNotifier, err := NewElasticSearch(log.WithField(notifierLogFieldKey, "ElasticSearch"), conf.Elasticsearch, reporter)
		if err != nil {
			return nil, fmt.Errorf("while creating Elasticsearch notifier: %w", err)
		}

		notifiers = append(notifiers, esNotifier)
	}

	if conf.Webhook.Enabled {
		whNotifier, err := NewWebhook(log.WithField(notifierLogFieldKey, "Webhook"), conf, reporter)
		if err != nil {
			return nil, fmt.Errorf("while creating Webhook notifier: %w", err)
		}

		notifiers = append(notifiers, whNotifier)
	}

	return notifiers, nil
}
