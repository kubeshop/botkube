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
	"testing"
	"time"

	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"

	"github.com/stretchr/testify/require"

	"github.com/slack-go/slack"

	"github.com/infracloudio/botkube/pkg/bot"
	"github.com/infracloudio/botkube/pkg/controller"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/command"
	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/infracloudio/botkube/test/e2e/filters"
	"github.com/infracloudio/botkube/test/e2e/notifier/create"
	"github.com/infracloudio/botkube/test/e2e/notifier/delete"
	notifierErr "github.com/infracloudio/botkube/test/e2e/notifier/error"
	"github.com/infracloudio/botkube/test/e2e/notifier/update"
	"github.com/infracloudio/botkube/test/e2e/welcome"
)

// TestRun run e2e integration tests
func TestRun(t *testing.T) {
	// New Environment to run integration tests
	testEnv, err := env.New()
	require.NoError(t, err)

	// Set up global logger and filter engine
	log.SetupGlobal()
	filterengine.SetupGlobal()

	// Fake notifiers
	var notifiers []notify.Notifier

	if testEnv.Config.Communications.Slack.Enabled {
		fakeSlackNotifier := &notify.Slack{
			Channel:   testEnv.Config.Communications.Slack.Channel,
			NotifType: testEnv.Config.Communications.Slack.NotifType,
			Client:    slack.New(testEnv.Config.Communications.Slack.Token, slack.OptionAPIURL(testEnv.SlackServer.GetAPIURL())),
		}

		notifiers = append(notifiers, fakeSlackNotifier)
	}

	if testEnv.Config.Communications.Webhook.Enabled {
		fakeWebhookNotifier := &notify.Webhook{
			URL: testEnv.WebhookServer.GetAPIURL(),
		}
		notifiers = append(notifiers, fakeWebhookNotifier)
	}

	utils.DynamicKubeClient = testEnv.K8sClient
	utils.DiscoveryClient = testEnv.DiscoFake
	utils.Mapper = testEnv.Mapper
	utils.InitInformerMap(testEnv.Config)
	utils.InitResourceMap(testEnv.Config)

	// Start controller with fake notifiers
	go controller.RegisterInformers(testEnv.Config, notifiers)
	t.Run("Welcome", welcome.E2ETests(testEnv))

	if testEnv.Config.Communications.Slack.Enabled {
		// Start fake Slack bot
		sb := NewFakeSlackBot(testEnv)
		go func(t *testing.T) {
			err := sb.Start()
			require.NoError(t, err)
		}(t)
	}

	time.Sleep(time.Second)

	// Make test suite
	suite := map[string]env.E2ETest{
		"notifier": create.E2ETests(testEnv),
		"command":  command.E2ETests(testEnv),
		"filters":  filters.E2ETests(testEnv),
		"update":   update.E2ETests(testEnv),
		"delete":   delete.E2ETests(testEnv),
		"error":    notifierErr.E2ETests(testEnv),
	}

	// Run test suite
	for name, test := range suite {
		t.Run(name, test.Run)
	}
}

// NewFakeSlackBot creates new mocked Slack bot
func NewFakeSlackBot(testenv *env.TestEnv) *bot.SlackBot {
	log.Info("Starting fake Slack bot")

	return &bot.SlackBot{
		Token:            testenv.Config.Communications.Slack.Token,
		AllowKubectl:     testenv.Config.Settings.Kubectl.Enabled,
		RestrictAccess:   testenv.Config.Settings.Kubectl.RestrictAccess,
		ClusterName:      testenv.Config.Settings.ClusterName,
		ChannelName:      testenv.Config.Communications.Slack.Channel,
		SlackURL:         testenv.SlackServer.GetAPIURL(),
		BotID:            testenv.SlackServer.BotID,
		DefaultNamespace: testenv.Config.Settings.Kubectl.DefaultNamespace,
		NewExecutorFn: func(msg string, allowKubectl, restrictAccess bool, defaultNamespace, clusterName string, platform config.BotPlatform, channelName string, isAuthChannel bool) execute.Executor {
			return execute.NewExecutorWithCustomCommandRunner(msg, allowKubectl, restrictAccess, defaultNamespace, clusterName, platform, channelName, isAuthChannel, command.FakeCommandRunnerFunc)
		},
	}
}
