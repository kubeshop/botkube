package argocd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/loggerx"
)

func TestNormalize(t *testing.T) {
	testCases := []struct {
		Name           string
		Input          string
		MaxSize        int
		ExpectedOutput string
	}{
		{
			Name:           "Invalid long string",
			Input:          "bk-botkube/argocd_gLhts.,-app-de.leted",
			MaxSize:        128,
			ExpectedOutput: "bk-botkube-argocd-glhts-app-de-leted",
		},
		{
			Name:           "Valid shorter string",
			Input:          "argocd-botkube",
			MaxSize:        20,
			ExpectedOutput: "argocd-botkube",
		},
		{
			Name:           "Too long invalid string",
			Input:          "bk-botkube/argocd_gLhts-app-deleted",
			MaxSize:        10,
			ExpectedOutput: "b-a5efd156",
		},
		{
			Name:           "Too long invalid string with same prefix",
			Input:          "bk-botkube/argocd_gLhts-app-del",
			MaxSize:        10,
			ExpectedOutput: "b-80c99c2d",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			out := normalize(loggerx.NewNoop(), tc.Input, tc.MaxSize)
			assert.Equal(t, tc.ExpectedOutput, out)
		})
	}
}
