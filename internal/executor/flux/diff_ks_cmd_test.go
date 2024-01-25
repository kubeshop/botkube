package flux

import (
	"github.com/kubeshop/botkube/pkg/loggerx"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewKustomizeDiffCmd(t *testing.T) {
	log := loggerx.NewNoop()

	t.Run("The command should be handled", func(t *testing.T) {
		cmd := "flux diff ks podinfo  --github-ref https://github.com/mszostok/podinfo/pull/2 --path ./kustomize --ignore-paths abc --ignore-paths dd"
		diffCmd, can := NewKustomizeDiffCmdService(nil, log).ShouldHandle(cmd)

		assert.True(t, can)
		assert.NotNil(t, diffCmd)

		diff := diffCmd.Get()
		assert.Equal(t, "podinfo", diff.AppName)
		assert.Equal(t, "https://github.com/mszostok/podinfo/pull/2", diff.GitHubRef)
		assert.Equal(t, "./kustomize", diff.Path)
		assert.Equal(t, []string{"abc", "dd"}, diff.IgnorePaths)

		assert.Equal(t, "flux diff ks podinfo --path ./kustomize --ignore-paths abc,dd --progress-bar=false", diff.ToCmdString())
	})

	t.Run("The command should not be handled", func(t *testing.T) {
		unsupportedCmd := "flux diff unsupported --some-flag value"
		unsupportedDiffCmd, can := NewKustomizeDiffCmdService(nil, log).ShouldHandle(unsupportedCmd)

		assert.False(t, can)
		assert.Empty(t, unsupportedDiffCmd)
	})
}
func TestNewKustomizeGitHubCommentCmd(t *testing.T) {
	log := loggerx.NewNoop()
	cmd := "flux diff gh comment --url https://github.com/mszostok/podinfo/pull/2 --cache-id d720520fc1bc3c07657130a0fa270d33"
	diffCmd, can := NewKustomizeDiffCmdService(nil, log).ShouldHandle(cmd)

	assert.True(t, can)
	assert.NotNil(t, diffCmd)

	comment := diffCmd.GitHub.Comment
	assert.Equal(t, "d720520fc1bc3c07657130a0fa270d33", comment.ArtifactID)
	assert.Equal(t, "https://github.com/mszostok/podinfo/pull/2", comment.URL)
}

func TestKustomizationDiffCommandFlags_ToString(t *testing.T) {
	// given
	flags := KustomizationDiffCommandFlags{
		IgnorePaths:       []string{"abc", "dd"},
		KustomizationFile: "ks",
		Path:              "./kustomize",
		ProgressBar:       false,
		GitHubRef:         "https://github.com/mszostok/podinfo/pull/2",
	}

	expected := " --kustomization-file ks --path ./kustomize --ignore-paths abc,dd --progress-bar=false"

	// when
	gotStringFlags := flags.ToString()

	// then
	assert.Equal(t, expected, gotStringFlags)
}

func TestGlobalCommandFlags_ToString(t *testing.T) {
	// given
	flags := GlobalCommandFlags{
		CacheDir:              "/Users/mszostok/.kube/cache",
		DisableCompression:    true,
		InsecureSkipTLSVerify: true,
		KubeAPIBurst:          300,
		KubeAPIQPS:            50,
		Namespace:             "flux-system",
		Timeout:               5 * time.Minute,
		Token:                 "YOUR_BEARER_TOKEN",
		Verbose:               true,
	}

	expected := " --cache-dir /Users/mszostok/.kube/cache --disable-compression  --insecure-skip-tls-verify  --kube-api-burst 300 --kube-api-qps 50 -n flux-system --timeout 5m0s --token YOUR_BEARER_TOKEN --verbose "

	// when
	gotStringFlags := flags.ToString()

	// then
	assert.Equal(t, expected, gotStringFlags)
}
