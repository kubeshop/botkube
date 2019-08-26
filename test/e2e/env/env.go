package env

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slacktest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// TestEnv to store objects required for e2e testing
// K8sClient    : Fake K8s client to mock resource creation
// SlackServer  : Fake Slack server
// SlackMessages: Channel to store incoming Slack messages from BotKube
// Config	: BotKube config provided with config.yaml
type TestEnv struct {
	K8sClient     kubernetes.Interface
	SlackServer   *slacktest.Server
	SlackMessages chan (*slack.MessageEvent)
	Config        *config.Config
}

// E2ETest interface to run tests
type E2ETest interface {
	Run(*testing.T)
}

// New creates TestEnv and populate required objects
func New() *TestEnv {
	testEnv := &TestEnv{}

	conf, err := config.New()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}
	testEnv.Config = conf

	// Set Slack Api token
	testEnv.Config.Communications.Slack.Enabled = true
	testEnv.Config.Communications.Slack.Token = "ABCDEFG"
	testEnv.Config.Communications.Slack.Channel = "cloud-alerts"

	// Add settings
	testEnv.Config.Settings.ClusterName = "test-cluster-1"
	testEnv.Config.Settings.AllowKubectl = true

	// Set fake BotKube version
	os.Setenv("BOTKUBE_VERSION", "v9.99.9")

	testEnv.K8sClient = fake.NewSimpleClientset()
	testEnv.SlackMessages = make(chan (*slack.MessageEvent), 1)
	testEnv.SetupFakeSlack()

	return testEnv
}

// SetupFakeSlack create fake Slack server to mock Slack
func (e *TestEnv) SetupFakeSlack() {
	s := slacktest.NewTestServer()
	s.SetBotName("BotKube")
	go s.Start()

	e.SlackServer = s
}

// GetLastSeenSlackMessage return last message received by fake slack server
func (e TestEnv) GetLastSeenSlackMessage() string {
	allSeenMessages := e.SlackServer.GetSeenOutboundMessages()
	if len(allSeenMessages) != 0 {
		return allSeenMessages[len(allSeenMessages)-1]
	}
	return ""
}
