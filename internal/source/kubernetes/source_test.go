package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TODO: Refactor these tests as a part of https://github.com/kubeshop/botkube/issues/589
//  These tests were moved from old E2E package with fake K8s and Slack API
//  (deleted in https://github.com/kubeshop/botkube/pull/627) and adjusted to become unit tests.

func TestController_strToGVR(t *testing.T) {
	// test scenarios
	tests := []struct {
		Name               string
		Input              string
		Expected           schema.GroupVersionResource
		ExpectedErrMessage string
	}{
		{
			Name:  "Without group",
			Input: "v1/persistentvolumes",
			Expected: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "persistentvolumes",
			},
		},
		{
			Name:  "With group",
			Input: "apps/v1/daemonsets",
			Expected: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "daemonsets",
			},
		},
		{
			Name:  "With more complex group",
			Input: "rbac.authorization.k8s.io/v1/clusterroles",
			Expected: schema.GroupVersionResource{
				Group:    "rbac.authorization.k8s.io",
				Version:  "v1",
				Resource: "clusterroles",
			},
		},
		{
			Name:               "Error",
			Input:              "foo/bar/baz/qux",
			ExpectedErrMessage: "invalid string: expected 2 or 3 parts when split by \"/\"",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			res, err := strToGVR(testCase.Input)

			if testCase.ExpectedErrMessage != "" {
				require.Error(t, err)
				assert.EqualError(t, err, testCase.ExpectedErrMessage)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.Expected, res)
		})
	}
}
