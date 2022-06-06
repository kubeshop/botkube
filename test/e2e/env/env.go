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

package env

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slacktest"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	kubeFake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/restmapper"
	samplev1alpha1 "k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/controller"
	"github.com/infracloudio/botkube/test/e2e/utils"
	"github.com/infracloudio/botkube/test/webhook"
)

// TestEnv to store objects required for e2e testing
type TestEnv struct {

	// Ctrl is a pointer to the BotKube controller
	Ctrl *controller.Controller

	// DiscoveryCli is a fake Discovery client
	DiscoveryCli discovery.DiscoveryInterface

	// DynamicCli is a fake K8s client to mock resource creation
	DynamicCli dynamic.Interface

	// Config is provided with config.yaml
	Config *config.Config

	// SlackServer is a fake Slack server
	SlackServer *slacktest.Server

	// SlackMessages is a channel to store incoming Slack messages from BotKube
	SlackMessages chan *slack.MessageEvent

	// WebhookServer is a fake Webhook server
	WebhookServer *webhook.Server

	// Mapper is a K8s resources mapper that uses DiscoveryCli
	Mapper meta.RESTMapper
}

// E2ETest interface to run tests
type E2ETest interface {
	Run(*testing.T)
}

// New creates TestEnv and populate required objects
func New(configPath string) (*TestEnv, error) {
	// Loads test configuration for Integration Testing
	conf, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("while loading configuration: %w", err)
	}

	if conf == nil {
		return nil, errors.New("while loading configuration: config cannot be nil")
	}

	s := runtime.NewScheme()

	err = samplev1alpha1.AddToScheme(s)
	if err != nil {
		return nil, err
	}
	err = corev1.AddToScheme(s)
	if err != nil {
		return nil, err
	}
	err = networkingv1.AddToScheme(s)
	if err != nil {
		return nil, err
	}

	return &TestEnv{
		Config:       conf,
		DynamicCli:   fake.NewSimpleDynamicClient(s),
		DiscoveryCli: kubeFake.NewSimpleClientset().Discovery(),
		Mapper: restmapper.NewDeferredDiscoveryRESTMapper(
			cacheddiscovery.NewMemCacheClient(FakeCachedDiscoveryInterface()),
		),
	}, nil
}

// SetupFakeSlack create fake Slack server to mock Slack
func (e *TestEnv) SetupFakeSlack() {
	e.SlackMessages = make(chan *slack.MessageEvent, 1)

	s := slacktest.NewTestServer()
	s.SetBotName("BotKube")
	go s.Start()

	e.SlackServer = s
}

// GetLastSeenSlackMessage return last message received by fake slack server
func (e TestEnv) GetLastSeenSlackMessage() *string {
	time.Sleep(5 * time.Second)

	allSeenMessages := e.SlackServer.GetSeenOutboundMessages()
	if len(allSeenMessages) != 0 {
		return &allSeenMessages[len(allSeenMessages)-1]
	}
	return nil
}

// SetupFakeWebhook create fake Slack server to mock Slack
func (e *TestEnv) SetupFakeWebhook() {
	s := webhook.NewTestServer()
	go s.Start()

	e.WebhookServer = s
}

// GetLastReceivedPayload return last message received by fake webhook server
func (e TestEnv) GetLastReceivedPayload() *utils.WebhookPayload {
	time.Sleep(5 * time.Second)

	allSeenMessages := e.WebhookServer.GetReceivedPayloads()
	if len(allSeenMessages) != 0 {
		return &allSeenMessages[len(allSeenMessages)-1]
	}
	return nil
}
