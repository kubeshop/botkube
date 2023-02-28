package commander

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
)

func TestCommander_GetCommandsForEvent(t *testing.T) {
	// given
	allowedVerbs := []string{"describe", "get", "logs", "delete"}
	allowedResources := []string{"pods", "deployments", "nodes"}
	testCases := []struct {
		Name  string
		Event event.Event
		Guard CmdGuard

		ExpectedResult     []Command
		ExpectedErrMessage string
	}{
		{
			Name: "Skip delete event",
			Event: event.Event{
				Resource:  "apps/v1/deployments",
				Name:      "foo",
				Namespace: "default",
				Type:      config.DeleteEvent,
			},
			Guard: nil,

			ExpectedResult:     nil,
			ExpectedErrMessage: "",
		},
		{
			Name: "Resource not allowed",
			Event: event.Event{
				Resource:  "apps/v1/deployments",
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
			ExpectedResult:     nil,
			ExpectedErrMessage: "",
		},
		{
			Name: "Namespaced resource",
			Event: event.Event{
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
			ExpectedResult: []Command{
				{Name: "describe", Cmd: "describe pods foo --namespace default"},
				{Name: "get", Cmd: "get pods foo --namespace default"},
				{Name: "logs", Cmd: "logs pods/foo --namespace default"},
			},
			ExpectedErrMessage: "",
		},
		{
			Name: "Cluster-wide resource",
			Event: event.Event{
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
			ExpectedResult: []Command{
				{Name: "describe", Cmd: "describe nodes foo"},
				{Name: "get", Cmd: "get nodes foo"},
			},
			ExpectedErrMessage: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cmder := NewCommander(loggerx.NewNoop(), tc.Guard, allowedVerbs, allowedResources)

			// when
			result, err := cmder.GetCommandsForEvent(tc.Event)

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

type fakeGuard struct {
	resMap  map[string]metav1.APIResource
	verbMap map[string]map[string]Resource
}

func (f *fakeGuard) GetServerResourceMap() (map[string]metav1.APIResource, error) {
	return f.resMap, nil
}

func (f *fakeGuard) GetResourceDetailsFromMap(selectedVerb, resourceType string, _ map[string]metav1.APIResource) (Resource, error) {
	resources, ok := f.verbMap[selectedVerb]
	if !ok {
		return Resource{}, ErrVerbNotSupported
	}

	res, ok := resources[resourceType]
	if !ok {
		return Resource{}, ErrVerbNotSupported
	}

	return res, nil
}

func fixVerbMapForFakeGuard() map[string]map[string]Resource {
	return map[string]map[string]Resource{
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
		"delete": {
			"pods": {
				Name:                    "pods",
				SlashSeparatedInCommand: false,
				Namespaced:              true,
			},
		},
	}
}
