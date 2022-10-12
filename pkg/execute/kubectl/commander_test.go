package kubectl_test

import (
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

func TestCommander_GetCommandsForEvent(t *testing.T) {
	// given
	executorBindings := []string{"foo", "bar"}
	testCases := []struct {
		Name           string
		Event          events.Event
		MergedKubectls kubectl.EnabledKubectl
		Guard          kubectl.CmdGuard

		ExpectedResult     []kubectl.Command
		ExpectedErrMessage string
	}{
		{
			Name: "Skip delete event",
			Event: events.Event{
				Resource:  "apps/v1/deployments",
				Name:      "foo",
				Namespace: "default",
				Type:      config.DeleteEvent,
			},
			Guard:              nil,
			MergedKubectls:     kubectl.EnabledKubectl{},
			ExpectedResult:     nil,
			ExpectedErrMessage: "",
		},
		{
			Name: "Resource not allowed",
			Event: events.Event{
				Resource:  "apps/v1/deployments",
				Name:      "foo",
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			Guard: nil,
			MergedKubectls: kubectl.EnabledKubectl{
				AllowedKubectlResource: map[string]struct{}{
					"services": {},
					"pods":     {},
				},
				AllowedKubectlVerb: map[string]struct{}{
					"get":      {},
					"describe": {},
				},
			},
			ExpectedResult:     nil,
			ExpectedErrMessage: "",
		},
		{
			Name: "Namespaced resource",
			Event: events.Event{
				Resource:  "v1/pods",
				Name:      "foo",
				Namespace: "default",
				Type:      config.CreateEvent,
			},
			Guard: &fakeGuard{
				resMap: map[string]metav1.APIResource{
					"pods":  {Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
					"nodes": {Name: "nodes", Namespaced: false, Kind: "Node", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"no"}},
				},
				verbMap: fixVerbMapForFakeGuard(),
			},
			MergedKubectls: kubectl.EnabledKubectl{
				AllowedKubectlResource: map[string]struct{}{
					"services":    {},
					"pods":        {},
					"deployments": {},
				},
				AllowedKubectlVerb: map[string]struct{}{
					"get":      {},
					"describe": {},
					"logs":     {},
				},
			},
			ExpectedResult: []kubectl.Command{
				{Name: "describe", Cmd: "describe pods foo --namespace default"},
				{Name: "get", Cmd: "get pods foo --namespace default"},
				{Name: "logs", Cmd: "logs pods/foo --namespace default"},
			},
			ExpectedErrMessage: "",
		},
		{
			Name: "Cluster-wide resource",
			Event: events.Event{
				Resource: "v1/nodes",
				Name:     "foo",
				Type:     config.UpdateEvent,
			},
			Guard: &fakeGuard{
				resMap: map[string]metav1.APIResource{
					"pods":  {Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
					"nodes": {Name: "nodes", Namespaced: false, Kind: "Node", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"no"}},
				},
				verbMap: fixVerbMapForFakeGuard(),
			},
			MergedKubectls: kubectl.EnabledKubectl{
				AllowedKubectlResource: map[string]struct{}{
					"services":    {},
					"pods":        {},
					"nodes":       {},
					"deployments": {},
				},
				AllowedKubectlVerb: map[string]struct{}{
					"get":      {},
					"describe": {},
					"logs":     {},
				},
			},
			ExpectedResult: []kubectl.Command{
				{Name: "describe", Cmd: "describe nodes foo"},
				{Name: "get", Cmd: "get nodes foo"},
			},
			ExpectedErrMessage: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			logger, _ := logtest.NewNullLogger()
			cmder := kubectl.NewCommander(logger, &fakeMerger{res: tc.MergedKubectls}, tc.Guard)

			// when
			result, err := cmder.GetCommandsForEvent(tc.Event, executorBindings)

			// then
			if tc.ExpectedErrMessage != "" {
				require.Error(t, err)
				assert.Equal(t, tc.ExpectedErrMessage, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}

type fakeMerger struct {
	res kubectl.EnabledKubectl
}

func (f *fakeMerger) MergeForNamespace(includeBindings []string, forNamespace string) kubectl.EnabledKubectl {
	return f.res
}

type fakeGuard struct {
	resMap  map[string]metav1.APIResource
	verbMap map[string]map[string]kubectl.Resource
}

func (f *fakeGuard) GetServerResourceMap() (map[string]metav1.APIResource, error) {
	return f.resMap, nil
}

func (f *fakeGuard) GetResourceDetailsFromMap(selectedVerb, resourceType string, resMap map[string]metav1.APIResource) (kubectl.Resource, error) {
	resources, ok := f.verbMap[selectedVerb]
	if !ok {
		return kubectl.Resource{}, kubectl.ErrVerbNotSupported
	}

	res, ok := resources[resourceType]
	if !ok {
		return kubectl.Resource{}, kubectl.ErrVerbNotSupported
	}

	return res, nil
}

func fixVerbMapForFakeGuard() map[string]map[string]kubectl.Resource {
	return map[string]map[string]kubectl.Resource{
		"get": {
			"pods": {
				Name:                    "pods",
				SlashSeparatedInCommand: false,
				Namespaced:              true,
			},
			"nodes": {
				Name:                    "nodes",
				Namespaced:              false,
				SlashSeparatedInCommand: false,
			},
		},
		"describe": {
			"pods": {
				Name:                    "pods",
				SlashSeparatedInCommand: false,
				Namespaced:              true,
			},
			"nodes": {
				Name:                    "nodes",
				Namespaced:              false,
				SlashSeparatedInCommand: false,
			},
		},
		"logs": {
			"pods": {
				Name:                    "pods",
				SlashSeparatedInCommand: true,
				Namespaced:              true,
			},
		},
	}
}
