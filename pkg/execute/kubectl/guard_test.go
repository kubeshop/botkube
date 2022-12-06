package kubectl_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

func TestCommandGuard_GetAllowedResourcesForVerb(t *testing.T) {
	// given
	fakeDiscoClient := &fakeDisco{
		list: []*v1.APIResourceList{
			{
				GroupVersion: "v1",
				APIResources: []v1.APIResource{
					{Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
					{Name: "services", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
					{Name: "nodes", Namespaced: false, Kind: "Node", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"no"}},
				},
			},
			{
				GroupVersion: "authentication.k8s.io/v1",
				APIResources: []v1.APIResource{
					{Name: "tokenreviews", Namespaced: false, Kind: "TokenReview", Verbs: []string{"create"}},
				},
			},
		},
	}
	testCases := []struct {
		Name                   string
		SelectedVerb           string
		AllConfiguredResources []string
		FakeDiscoClient        *fakeDisco

		ExpectedResult     []kubectl.Resource
		ExpectedErrMessage string
	}{
		{
			Name:                   "Resourceless verb",
			SelectedVerb:           "api-resources",
			AllConfiguredResources: []string{},
			FakeDiscoClient:        fakeDiscoClient,
			ExpectedResult:         nil,
			ExpectedErrMessage:     "",
		},
		{
			Name:                   "Discovery API Error",
			SelectedVerb:           "get",
			AllConfiguredResources: []string{"pods", "services", "nodes"},
			FakeDiscoClient:        &fakeDisco{err: errors.New("test")},
			ExpectedResult:         nil,
			ExpectedErrMessage:     "while getting resource list from K8s cluster: test",
		},
		{
			Name:                   "Discovery API resource list ignored error",
			SelectedVerb:           "get",
			AllConfiguredResources: []string{"pods"},
			FakeDiscoClient: &fakeDisco{
				list: []*v1.APIResourceList{
					{
						GroupVersion: "v1",
						APIResources: []v1.APIResource{
							{Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
						},
					},
				},
				err: &discovery.ErrGroupDiscoveryFailed{
					Groups: map[schema.GroupVersion]error{
						{Group: "", Version: "external.metrics.k8s.io/v1beta1"}: errors.New("Got empty response for: external.metrics.k8s.io/v1beta1"),
					},
				}},
			ExpectedResult: []kubectl.Resource{
				{Name: "pods", Namespaced: true, SlashSeparatedInCommand: false},
			},
			ExpectedErrMessage: "",
		},
		{
			Name:                   "Verb not supported",
			SelectedVerb:           "create",
			AllConfiguredResources: []string{"pods", "services", "nodes"},
			FakeDiscoClient:        fakeDiscoClient,
			ExpectedErrMessage:     "verb not supported",
		},
		{
			Name:                   "Filter out resources that don't support the verb",
			SelectedVerb:           "get",
			AllConfiguredResources: []string{"pods", "services", "nodes", "tokenreviews"},
			FakeDiscoClient:        fakeDiscoClient,
			ExpectedResult: []kubectl.Resource{
				{Name: "pods", Namespaced: true, SlashSeparatedInCommand: false},
				{Name: "services", Namespaced: true, SlashSeparatedInCommand: false},
				{Name: "nodes", Namespaced: false, SlashSeparatedInCommand: false},
			},
			ExpectedErrMessage: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cmdGuard := kubectl.NewCommandGuard(loggerx.NewNoop(), tc.FakeDiscoClient)

			// when
			result, err := cmdGuard.GetAllowedResourcesForVerb(tc.SelectedVerb, tc.AllConfiguredResources)

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

func TestCommandGuard_GetResourceDetails_HappyPath(t *testing.T) {
	testCases := []struct {
		Name         string
		SelectedVerb string
		ResourceType string
		ResourceMap  map[string]v1.APIResource

		ExpectedResult     kubectl.Resource
		ExpectedErrMessage string
	}{
		{
			Name:         "Namespaced",
			SelectedVerb: "get",
			ResourceType: "pods",
			ExpectedResult: kubectl.Resource{
				Name:                    "pods",
				Namespaced:              true,
				SlashSeparatedInCommand: false,
			},
		},
		{
			Name:           "Verb is resourceless",
			SelectedVerb:   "api-versions",
			ResourceType:   "",
			ExpectedResult: kubectl.Resource{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fakeDisco := &fakeDisco{
				list: []*v1.APIResourceList{
					{GroupVersion: "v1", APIResources: []v1.APIResource{
						{Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
					}},
				},
			}
			cmdGuard := kubectl.NewCommandGuard(loggerx.NewNoop(), fakeDisco)

			// when
			result, err := cmdGuard.GetResourceDetails(tc.SelectedVerb, tc.ResourceType)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}

func TestCommandGuard_GetServerResourceMap_HappyPath(t *testing.T) {
	// given
	expectedResMap := map[string]v1.APIResource{
		"pods":         {Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
		"nodes":        {Name: "nodes", Namespaced: false, Kind: "Node", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"no"}},
		"services":     {Name: "services", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
		"tokenreviews": {Name: "tokenreviews", Namespaced: false, Kind: "TokenReview", Verbs: []string{"create"}},
	}
	fakeDisco := &fakeDisco{
		list: []*v1.APIResourceList{
			{
				GroupVersion: "v1",
				APIResources: []v1.APIResource{
					{Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
					{Name: "services", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
					{Name: "nodes", Namespaced: false, Kind: "Node", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"no"}},
				},
			},
			{
				GroupVersion: "authentication.k8s.io/v1",
				APIResources: []v1.APIResource{
					{Name: "tokenreviews", Namespaced: false, Kind: "TokenReview", Verbs: []string{"create"}},
				},
			},
			{
				GroupVersion: "metrics.k8s.io/v1beta1",
				APIResources: []v1.APIResource{
					{Name: "pods", Namespaced: false, Kind: "PodMetrics", Verbs: []string{"get", "list"}},
				},
			},
		},
	}
	cmdGuard := kubectl.NewCommandGuard(loggerx.NewNoop(), fakeDisco)

	// when

	resMap, err := cmdGuard.GetServerResourceMap()

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedResMap, resMap)
}

func TestCommandGuard_GetResourceDetailsFromMap(t *testing.T) {
	// given
	resMap := map[string]v1.APIResource{
		"pods":  {Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
		"nodes": {Name: "nodes", Namespaced: false, Kind: "Node", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"no"}},
	}
	testCases := []struct {
		Name         string
		SelectedVerb string
		ResourceType string
		ResourceMap  map[string]v1.APIResource

		ExpectedResult     kubectl.Resource
		ExpectedErrMessage string
	}{
		{
			Name:         "Namespaced",
			SelectedVerb: "get",
			ResourceType: "pods",
			ResourceMap:  resMap,
			ExpectedResult: kubectl.Resource{
				Name:                    "pods",
				Namespaced:              true,
				SlashSeparatedInCommand: false,
			},
		},
		{
			Name:         "Slash-separated command",
			SelectedVerb: "logs",
			ResourceType: "pods",
			ResourceMap:  resMap,
			ExpectedResult: kubectl.Resource{
				Name:                    "pods",
				Namespaced:              true,
				SlashSeparatedInCommand: true,
			},
		},
		{
			Name:         "Cluster-wide",
			SelectedVerb: "get",
			ResourceType: "nodes",
			ResourceMap:  resMap,
			ExpectedResult: kubectl.Resource{
				Name:                    "nodes",
				Namespaced:              false,
				SlashSeparatedInCommand: false,
			},
		},
		{
			Name:         "Additional top verb",
			SelectedVerb: "top",
			ResourceType: "nodes",
			ResourceMap:  resMap,
			ExpectedResult: kubectl.Resource{
				Name:                    "nodes",
				Namespaced:              false,
				SlashSeparatedInCommand: false,
			},
		},
		{
			Name:               "Resource doesn't exist",
			SelectedVerb:       "get",
			ResourceType:       "drillbinding",
			ResourceMap:        resMap,
			ExpectedResult:     kubectl.Resource{},
			ExpectedErrMessage: "resource not found",
		},
		{
			Name:               "Unsupported verb",
			SelectedVerb:       "check",
			ResourceType:       "pods",
			ResourceMap:        resMap,
			ExpectedResult:     kubectl.Resource{},
			ExpectedErrMessage: "verb not supported",
		},
		{
			Name:               "Unsupported verb, but was returned by K8s API",
			SelectedVerb:       "patch",
			ResourceType:       "pods",
			ResourceMap:        resMap,
			ExpectedResult:     kubectl.Resource{},
			ExpectedErrMessage: "verb not supported",
		},
		{
			Name:         "Additional verb",
			SelectedVerb: "describe",
			ResourceType: "nodes",
			ResourceMap:  resMap,
			ExpectedResult: kubectl.Resource{
				Name:                    "nodes",
				Namespaced:              false,
				SlashSeparatedInCommand: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cmdGuard := kubectl.NewCommandGuard(loggerx.NewNoop(), nil)

			// when
			result, err := cmdGuard.GetResourceDetailsFromMap(tc.SelectedVerb, tc.ResourceType, tc.ResourceMap)

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

type fakeDisco struct {
	list []*v1.APIResourceList
	err  error
}

func (f *fakeDisco) ServerPreferredResources() ([]*v1.APIResourceList, error) {
	if f.err != nil {
		return f.list, f.err
	}

	return f.list, nil
}
