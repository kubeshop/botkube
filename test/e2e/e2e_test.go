package e2e

import (
	"log"
	"testing"
	"time"

	"github.com/infracloudio/botkube/pkg/bot"
	"github.com/infracloudio/botkube/pkg/controller"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/command"
	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/infracloudio/botkube/test/e2e/filters"
	"github.com/infracloudio/botkube/test/e2e/notifier/create"
	"github.com/infracloudio/botkube/test/e2e/welcome"
)

// TestRun run e2e integration tests
func TestRun(t *testing.T) {
	// New Environment to run integration tests
	testEnv := env.New()

	// Fake notifiers
	notifiers := []notify.Notifier{}

	if testEnv.Config.Communications.Slack.Enabled {
		fakeSlackNotifier := &notify.Slack{
			Token:     testEnv.Config.Communications.Slack.Token,
			Channel:   testEnv.Config.Communications.Slack.Channel,
			NotifType: testEnv.Config.Communications.Slack.NotifType,
			SlackURL:  testEnv.SlackServer.GetAPIURL(),
		}

		notifiers = append(notifiers, fakeSlackNotifier)
	}

	if testEnv.Config.Communications.Webhook.Enabled {
		fakeWebhookNotifier := &notify.Webhook{
			URL:         testEnv.WebhookServer.GetAPIURL(),
			ClusterName: testEnv.Config.Settings.ClusterName,
		}
		notifiers = append(notifiers, fakeWebhookNotifier)
	}

	utils.KubeClient = testEnv.K8sClient
	utils.InitInformerMap()

	// Start controller with fake notifiers
	go controller.RegisterInformers(testEnv.Config, notifiers)
	t.Run("Welcome", welcome.E2ETests(testEnv))

	if testEnv.Config.Communications.Slack.Enabled {
		// Start fake Slack bot
		StartFakeSlackBot(testEnv)
	}

	time.Sleep(time.Second)

	// Make test suite
	suite := map[string]env.E2ETest{
		"notifier": create.E2ETests(testEnv),
		"command":  command.E2ETests(testEnv),
		"filters":  filters.E2ETests(testEnv),
	}

	// Run test suite
	for name, test := range suite {
		t.Run(name, test.Run)
	}
}

// StartFakeSlackBot makes connection to mocked slack apiserver
func StartFakeSlackBot(testenv *env.TestEnv) {
	if testenv.Config.Communications.Slack.Enabled {
		log.Println("Starting fake Slack bot")

		// Fake bot
		sb := &bot.SlackBot{
			Token:          testenv.Config.Communications.Slack.Token,
			AllowKubectl:   testenv.Config.Settings.AllowKubectl,
			RestrictAccess: testenv.Config.Settings.RestrictAccess,
			ClusterName:    testenv.Config.Settings.ClusterName,
			ChannelName:    testenv.Config.Communications.Slack.Channel,
			SlackURL:       testenv.SlackServer.GetAPIURL(),
			BotID:          testenv.SlackServer.BotID,
		}
		go sb.Start()
	}
}
