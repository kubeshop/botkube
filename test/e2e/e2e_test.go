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

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	"golang.org/x/sync/errgroup"

	"github.com/infracloudio/botkube/pkg/bot"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/controller"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/test/e2e/command"
	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/infracloudio/botkube/test/e2e/filters"
	"github.com/infracloudio/botkube/test/e2e/notifier/create"
	"github.com/infracloudio/botkube/test/e2e/notifier/delete"
	"github.com/infracloudio/botkube/test/e2e/notifier/update"
	"github.com/infracloudio/botkube/test/e2e/welcome"
)

const componentLogFieldKey = "component"
const botLogFieldKey = "bot"

// Config contains the test configuration parameters.
type Config struct {
	ConfigPath            string        `envconfig:"optional"`
	InformersResyncPeriod time.Duration `envconfig:"default=30m"`
	KubeconfigPath        string        `envconfig:"optional,KUBECONFIG"`
}

// TestRun run e2e integration tests
func TestRun(t *testing.T) {
	var appCfg Config
	err := envconfig.Init(&appCfg)
	require.NoError(t, err, "while loading app configuration")

	// New Environment to run integration tests
	testEnv, err := env.New(appCfg.KubeconfigPath)
	require.NoError(t, err)

	ctx, cancelCtxFn := context.WithCancel(context.Background())
	defer cancelCtxFn()

	logger, _ := logtest.NewNullLogger()
	errGroup := new(errgroup.Group)

	// Filter engine
	filterEngine := filterengine.WithAllFilters(logger, testEnv.DynamicCli, testEnv.Mapper, testEnv.Config)

	// Fake notifiers
	var notifiers []notify.Notifier

	if testEnv.Config.Communications.Slack.Enabled {
		t.Log("Starting test Slack server")
		testEnv.SetupFakeSlack()

		fakeSlackNotifier := notify.NewSlack(logger, testEnv.Config.Communications.Slack)
		fakeSlackNotifier.Client = slack.New(testEnv.Config.Communications.Slack.Token, slack.OptionAPIURL(testEnv.SlackServer.GetAPIURL()))
		notifiers = append(notifiers, fakeSlackNotifier)
	}

	if testEnv.Config.Communications.Webhook.Enabled {
		t.Log("Starting fake webhook server")
		testEnv.SetupFakeWebhook()

		fakeWebhookNotifier := notify.NewWebhook(logger, config.CommunicationsConfig{
			Webhook: config.Webhook{
				URL: testEnv.WebhookServer.GetAPIURL(),
			},
		})
		notifiers = append(notifiers, fakeWebhookNotifier)
	}

	resMapping, err := execute.LoadResourceMappingIfShould(
		logger.WithField(componentLogFieldKey, "Resource Mapping Loader"),
		testEnv.Config,
		testEnv.DiscoveryCli,
	)
	require.NoError(t, err)

	executorFactory := execute.NewExecutorFactory(
		logger.WithField(componentLogFieldKey, "Executor"),
		command.FakeCommandRunnerFunc,
		*testEnv.Config,
		filterEngine,
		resMapping,
	)

	// Controller

	ctrl := controller.New(
		logger.WithField(componentLogFieldKey, "Controller"),
		testEnv.Config,
		notifiers,
		filterEngine,
		appCfg.ConfigPath,
		testEnv.DynamicCli,
		testEnv.Mapper,
		appCfg.InformersResyncPeriod,
	)

	testEnv.Ctrl = ctrl

	// Start test components

	errGroup.Go(func() error {
		t.Log("Starting controller")
		return ctrl.Start(ctx)
	})

	if testEnv.Config.Communications.Slack.Enabled {
		t.Log("Starting fake Slack bot")
		sb := NewFakeSlackBot(logger.WithField(botLogFieldKey, "Slack"), executorFactory, testEnv)
		errGroup.Go(func() error {
			return sb.Start(ctx)
		})
	}

	// Start controller with fake notifiers
	t.Run("Welcome", welcome.E2ETests(testEnv))

	time.Sleep(time.Second)

	// Make test suite
	suite := map[string]env.E2ETest{
		"notifier": create.E2ETests(testEnv),
		"command":  command.E2ETests(testEnv),
		"filters":  filters.E2ETests(testEnv),
		"update":   update.E2ETests(testEnv),
		"delete":   delete.E2ETests(testEnv),
	}

	// Run test suite
	for name, test := range suite {
		t.Run(name, test.Run)
	}

	t.Log("Cancelling context")
	cancelCtxFn()
	err = errGroup.Wait()
	require.NoError(t, err)
}

// NewFakeSlackBot creates new mocked Slack bot
func NewFakeSlackBot(log logrus.FieldLogger, executorFactory bot.ExecutorFactory, testenv *env.TestEnv) *bot.SlackBot {
	slackBot := bot.NewSlackBot(log, testenv.Config, executorFactory)
	slackBot.SlackURL = testenv.SlackServer.GetAPIURL()
	slackBot.BotID = testenv.SlackServer.BotID

	return slackBot
}
