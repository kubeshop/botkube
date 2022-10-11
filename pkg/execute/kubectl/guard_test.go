package kubectl_test

import (
	"errors"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
			ExpectedErrMessage:     "while getting server resources: test",
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
			logger, _ := logtest.NewNullLogger()
			cmdGuard := kubectl.NewCommandGuard(logger, tc.FakeDiscoClient)

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
	// given
	expectedRes := kubectl.Resource{
		Name:                    "pods",
		Namespaced:              true,
		SlashSeparatedInCommand: false,
	}
	fakeDisco := &fakeDisco{
		list: []*v1.APIResourceList{
			{GroupVersion: "v1", APIResources: []v1.APIResource{
				{Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}},
			}},
		},
	}
	logger, _ := logtest.NewNullLogger()

	cmdGuard := kubectl.NewCommandGuard(logger, fakeDisco)

	// when

	res, err := cmdGuard.GetResourceDetails("get", "pods")

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedRes, res)
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
	logger, _ := logtest.NewNullLogger()

	cmdGuard := kubectl.NewCommandGuard(logger, fakeDisco)

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
			logger, _ := logtest.NewNullLogger()
			cmdGuard := kubectl.NewCommandGuard(logger, nil)

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
		return nil, f.err
	}

	return f.list, nil
}
