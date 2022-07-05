package notify

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

const notifierLogFieldKey = "notifier"

// Notifier to send event notification on the communication channels
type Notifier interface {
	SendEvent(context.Context, events.Event) error
	SendMessage(context.Context, string) error
}

// LoadNotifiers returns list of configured notifiers
func LoadNotifiers(log *logrus.Logger, conf config.CommunicationsConfig) ([]Notifier, error) {
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
		notifiers = append(notifiers, NewDiscord(log.WithField(notifierLogFieldKey, "Discord"), conf.Discord))
	}

	if conf.ElasticSearch.Enabled {
		esNotifier, err := NewElasticSearch(log.WithField(notifierLogFieldKey, "ElasticSearch"), conf.ElasticSearch)
		if err != nil {
			return nil, fmt.Errorf("while creating ElasticSearch client: %w", err)
		}

		notifiers = append(notifiers, esNotifier)
	}

	if conf.Webhook.Enabled {
		notifiers = append(notifiers, NewWebhook(log.WithField(notifierLogFieldKey, "Webhook"), conf))
	}

	return notifiers, nil
}
