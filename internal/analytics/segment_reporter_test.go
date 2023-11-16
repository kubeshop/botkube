// The analytics.SegmentReporter tests in the `analytics_test` package is based on golden file pattern.
// If the `-test.update-golden` flag is set then the actual content is written
// to the golden file.
//
// To update golden files, run:
//
//	go test ./internal/analytics -test.update-golden
package analytics_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	segment "github.com/segmentio/analytics-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sVersion "k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/botkube/internal/analytics"
	batched "github.com/kubeshop/botkube/internal/analytics/batched"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/ptr"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/version"
)

func TestSegmentReporter_RegisterCurrentIdentity(t *testing.T) {
	// given
	kubeSystemNs := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
			UID:  "ff68560b-44e8-4b0d-880b-e114f5d15933",
		},
	}
	cpNode1 := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cp1",
			Labels: map[string]string{
				"node-role.kubernetes.io/control-plane": "true",
			},
		},
	}
	cpNode2 := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cp2",
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "true",
			},
		},
	}
	wrkNode1 := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker1",
		},
	}
	wrkNode2 := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker2",
		},
	}
	wrkNode3 := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker3",
		},
	}

	fakeIdentity := fixIdentity()

	k8sCli := fake.NewSimpleClientset(&kubeSystemNs, &cpNode1, &cpNode2, &wrkNode1, &wrkNode2, &wrkNode3)
	fakeDisco, ok := k8sCli.Discovery().(*fakediscovery.FakeDiscovery)
	require.True(t, ok)

	fakeDisco.FakedServerVersion = &fakeIdentity.KubernetesVersion

	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(nil)

	// when
	err := segmentReporter.RegisterCurrentIdentity(context.Background(), k8sCli, "")
	require.NoError(t, err)
	err = segmentReporter.RegisterCurrentIdentity(context.Background(), k8sCli, "remote-deploy-id")
	require.NoError(t, err)

	// then
	identity := segmentReporter.Identity()
	assert.Equal(t, string(kubeSystemNs.UID), identity.AnonymousID)
	assert.Equal(t, "remote-deploy-id", identity.DeploymentID)

	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func TestSegmentReporter_ReportCommand(t *testing.T) {
	// given
	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)

	// when
	err := segmentReporter.ReportCommand(analytics.ReportCommandInput{
		Platform: config.DiscordCommPlatformIntegration,
		Command:  "enable notifications",
		Origin:   command.TypedOrigin,
	})
	require.NoError(t, err)

	err = segmentReporter.ReportCommand(analytics.ReportCommandInput{
		Platform:   config.SlackCommPlatformIntegration,
		PluginName: "botkube/kubectl",
		Command:    "get",
		Origin:     command.ButtonClickOrigin,
		WithFilter: true,
	})
	require.NoError(t, err)

	err = segmentReporter.ReportCommand(analytics.ReportCommandInput{
		Platform: config.TeamsCommPlatformIntegration,
		Command:  "disable notifications",
		Origin:   command.SelectValueChangeOrigin,
	})
	require.NoError(t, err)

	// then
	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func TestSegmentReporter_ReportBotEnabled(t *testing.T) {
	// given
	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)

	// when
	err := segmentReporter.ReportBotEnabled(config.SlackCommPlatformIntegration, 1)
	require.NoError(t, err)

	// when
	err = segmentReporter.ReportBotEnabled(config.DiscordCommPlatformIntegration, 2)
	require.NoError(t, err)

	// when
	err = segmentReporter.ReportBotEnabled(config.TeamsCommPlatformIntegration, 1)
	require.NoError(t, err)

	// then
	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func TestSegmentReporter_ReportSinkEnabled(t *testing.T) {
	// given
	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)

	// when
	err := segmentReporter.ReportSinkEnabled(config.WebhookCommPlatformIntegration, 1)
	require.NoError(t, err)

	// when
	err = segmentReporter.ReportSinkEnabled(config.ElasticsearchCommPlatformIntegration, 2)
	require.NoError(t, err)

	// then
	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

// ReportHandledEventSuccess and ReportHandledEventError are tested together as a part of TestSegmentReporter_Run.

func TestSegmentReporter_Run(t *testing.T) {
	// given
	tick := 50 * time.Millisecond
	timeout := 5 * time.Second
	sampleErr := errors.New("sample error")

	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)
	segmentReporter.SetTickDuration(tick)

	eventDetails := map[string]interface{}{
		"type":       "create",
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
	}

	// when
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()

	wg := sync.WaitGroup{}
	var runErr error
	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		runErr = segmentReporter.Run(ctx)
	}(ctx)

	err := segmentReporter.ReportHandledEventSuccess(analytics.ReportEventInput{
		IntegrationType:       config.BotIntegrationType,
		Platform:              config.SlackCommPlatformIntegration,
		PluginName:            "botkube/kubernetes",
		AnonymizedEventFields: eventDetails,
	})
	require.NoError(t, err)

	err = segmentReporter.ReportHandledEventError(analytics.ReportEventInput{
		IntegrationType:       config.SinkIntegrationType,
		Platform:              config.ElasticsearchCommPlatformIntegration,
		PluginName:            "botkube/kubernetes",
		AnonymizedEventFields: eventDetails,
	}, sampleErr)
	require.NoError(t, err)

	time.Sleep(tick + 5*time.Millisecond)

	err = segmentReporter.ReportHandledEventSuccess(analytics.ReportEventInput{
		IntegrationType:       config.BotIntegrationType,
		Platform:              config.TeamsCommPlatformIntegration,
		PluginName:            "botkube/argocd",
		AnonymizedEventFields: eventDetails,
	})
	require.NoError(t, err)
	err = segmentReporter.ReportHandledEventSuccess(analytics.ReportEventInput{
		IntegrationType:       config.BotIntegrationType,
		Platform:              config.SlackCommPlatformIntegration,
		PluginName:            "botkube/kubernetes",
		AnonymizedEventFields: eventDetails,
	})
	require.NoError(t, err)

	cancelFn()
	wg.Wait()
	require.NoError(t, runErr)

	// then
	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func TestSegmentReporter_ReportFatalError(t *testing.T) {
	//given
	fatalErr := errors.New("fatal error")
	testCases := []struct {
		Name          string
		InputIdentity *analytics.Identity
	}{
		{
			Name:          "Identity",
			InputIdentity: fixIdentity(),
		},
		{
			Name:          "No identity",
			InputIdentity: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(tc.InputIdentity)

			// when
			err := segmentReporter.ReportFatalError(fatalErr)
			require.NoError(t, err)

			// then
			compareMessagesAgainstGoldenFile(t, segmentCli.messages)
		})
	}
}

func TestSegmentReporter_ReportHeartbeatEvent(t *testing.T) {
	//given
	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)
	batchedData := fakeBatchedData{
		props: fixHeartbeatProperties(),
	}
	segmentReporter.SetBatchedData(batchedData)

	// when
	err := segmentReporter.ReportHeartbeatEvent()
	require.NoError(t, err)

	// then
	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func fakeSegmentReporterWithIdentity(identity *analytics.Identity) (*analytics.SegmentReporter, *fakeSegmentCli) {
	segmentCli := &fakeSegmentCli{}
	segmentReporter := analytics.NewSegmentReporter(loggerx.NewNoop(), segmentCli)
	segmentReporter.SetIdentity(identity)

	return segmentReporter, segmentCli
}

func compareMessagesAgainstGoldenFile(t *testing.T, actualMessages []segment.Message) {
	filename := fmt.Sprintf("%s.json", t.Name())
	bytes, err := json.MarshalIndent(actualMessages, "", "\t")
	require.NoError(t, err)
	golden.Assert(t, string(bytes), filename)
}

func fixIdentity() *analytics.Identity {
	return &analytics.Identity{
		AnonymousID: "cluster-id",
		KubernetesVersion: k8sVersion.Info{
			Major:        "k8s-major",
			Minor:        "k8s-minor",
			GitVersion:   "k8s-git-version",
			GitCommit:    "k8s-git-commit",
			GitTreeState: "k8s-git-tree-state",
			BuildDate:    "k8s-build-date",
			GoVersion:    "k8s-go-version",
			Compiler:     "k8s-compiler",
			Platform:     "k8s-platform",
		},
		BotkubeVersion: version.Details{
			Version:     "botkube-version",
			GitCommitID: "botkube-git-commit-id",
			BuildDate:   "botkube-build-date",
		},
		WorkerNodeCount:       0,
		ControlPlaneNodeCount: 0,
	}
}

func fixHeartbeatProperties() batched.HeartbeatProperties {
	return batched.HeartbeatProperties{
		TimeWindowInHours: 1,
		EventsCount:       3,
		Sources: map[string]batched.SourceProperties{
			"botkube/argocd": {
				EventsCount: 1,
				Events: []batched.SourceEvent{
					{
						IntegrationType: config.SinkIntegrationType,
						Platform:        config.ElasticsearchCommPlatformIntegration,
						PluginName:      "botkube/argocd",
						AnonymizedEventFields: map[string]any{
							"foo": "bar",
							"baz": 1,
						},
						Success: true,
						Error:   nil,
					},
				},
			},
			"botkube/kubernetes": {
				EventsCount: 2,
				Events: []batched.SourceEvent{
					{
						IntegrationType:       config.BotIntegrationType,
						Platform:              config.CloudSlackCommPlatformIntegration,
						PluginName:            "botkube/kubernetes",
						AnonymizedEventFields: nil,
						Success:               false,
						Error:                 ptr.FromType("sample error"),
					},
					{
						IntegrationType: config.BotIntegrationType,
						Platform:        config.DiscordCommPlatformIntegration,
						PluginName:      "botkube/kubernetes",
						AnonymizedEventFields: map[string]any{
							"foo": "bar",
						},
						Success: true,
					},
				},
			},
		},
	}
}

type fakeBatchedData struct {
	props batched.HeartbeatProperties
}

func (f fakeBatchedData) AddSourceEvent(event batched.SourceEvent) {}

func (f fakeBatchedData) HeartbeatProperties() batched.HeartbeatProperties {
	return f.props
}

func (f fakeBatchedData) IncrementTimeWindowInHours() {}

func (f fakeBatchedData) Reset() {}
