// The analytics.SegmentReporter tests in the `analytics_test` package is based on golden file pattern.
// If the `-test.update-golden` flag is set then the actual content is written
// to the golden file.
//
// To update golden files, run:
//
//	go test ./internal/analytics/... -test.update-golden
package analytics_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	segment "github.com/segmentio/analytics-go"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sVersion "k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/botkube/internal/analytics"
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
	err := segmentReporter.RegisterCurrentIdentity(context.Background(), k8sCli)
	require.NoError(t, err)

	// then
	identity := segmentReporter.Identity()
	assert.Equal(t, string(kubeSystemNs.UID), identity.ID)

	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func TestSegmentReporter_ReportCommand(t *testing.T) {
	// given
	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)

	// when
	err := segmentReporter.ReportCommand(config.DiscordCommPlatformIntegration, "notifications stop", command.TypedOrigin, false)
	require.NoError(t, err)

	err = segmentReporter.ReportCommand(config.SlackCommPlatformIntegration, "get", command.ButtonClickOrigin, false)
	require.NoError(t, err)

	err = segmentReporter.ReportCommand(config.TeamsCommPlatformIntegration, "notifications start", command.SelectValueChangeOrigin, false)
	require.NoError(t, err)

	// then
	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func TestSegmentReporter_ReportBotEnabled(t *testing.T) {
	// given
	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)

	// when
	err := segmentReporter.ReportBotEnabled(config.SlackCommPlatformIntegration)
	require.NoError(t, err)

	// when
	err = segmentReporter.ReportBotEnabled(config.DiscordCommPlatformIntegration)
	require.NoError(t, err)

	// when
	err = segmentReporter.ReportBotEnabled(config.TeamsCommPlatformIntegration)
	require.NoError(t, err)

	// then
	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func TestSegmentReporter_ReportSinkEnabled(t *testing.T) {
	// given
	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)

	// when
	err := segmentReporter.ReportSinkEnabled(config.WebhookCommPlatformIntegration)
	require.NoError(t, err)

	// when
	err = segmentReporter.ReportSinkEnabled(config.ElasticsearchCommPlatformIntegration)
	require.NoError(t, err)

	// then
	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func TestSegmentReporter_ReportHandledEventSuccess(t *testing.T) {
	// given
	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)
	eventDetails := analytics.EventDetails{
		Type:       "create",
		APIVersion: "apps/v1",
		Kind:       "Deployment",
	}

	// when
	err := segmentReporter.ReportHandledEventSuccess(config.BotIntegrationType, config.SlackCommPlatformIntegration, eventDetails)
	require.NoError(t, err)

	err = segmentReporter.ReportHandledEventSuccess(config.SinkIntegrationType, config.ElasticsearchCommPlatformIntegration, eventDetails)
	require.NoError(t, err)

	// then
	compareMessagesAgainstGoldenFile(t, segmentCli.messages)
}

func TestSegmentReporter_ReportHandledEventError(t *testing.T) {
	// given
	identity := fixIdentity()
	segmentReporter, segmentCli := fakeSegmentReporterWithIdentity(identity)
	eventDetails := analytics.EventDetails{
		Type:       "create",
		APIVersion: "apps/v1",
		Kind:       "Deployment",
	}
	sampleErr := errors.New("sample error")

	// when
	err := segmentReporter.ReportHandledEventError(config.BotIntegrationType, config.SlackCommPlatformIntegration, eventDetails, sampleErr)
	require.NoError(t, err)

	err = segmentReporter.ReportHandledEventError(config.SinkIntegrationType, config.ElasticsearchCommPlatformIntegration, eventDetails, sampleErr)
	require.NoError(t, err)

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

func fakeSegmentReporterWithIdentity(identity *analytics.Identity) (*analytics.SegmentReporter, *fakeSegmentCli) {
	logger, _ := logtest.NewNullLogger()
	segmentCli := &fakeSegmentCli{}
	segmentReporter := analytics.NewSegmentReporter(logger, segmentCli)
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
		ID: "cluster-id",
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
