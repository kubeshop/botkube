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

package notify

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
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

	if conf.Lark.Enabled {
		notifiers = append(notifiers, NewLark(log.WithField(notifierLogFieldKey, "Lark"), log.GetLevel(), conf))
	}

	return notifiers, nil
}
