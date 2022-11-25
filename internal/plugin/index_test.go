package plugin_test

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/plugin"
)

func TestBuildPluginKey(t *testing.T) {
	type given struct {
		repoName   string
		pluginName string
		version    string
	}
	t.Run("Success", func(t *testing.T) {
		tests := []struct {
			name   string
			given  given
			expKey string
		}{
			{
				name: "build key with all properties",
				given: given{
					repoName:   "hakuna",
					pluginName: "kubectl",
					version:    "v1.0.0",
				},
				expKey: "hakuna/kubectl@v1.0.0",
			},
			{
				name: "build key without version",
				given: given{
					repoName:   "hakuna",
					pluginName: "kubectl",
				},
				expKey: "hakuna/kubectl",
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// when
				gotKey, err := plugin.BuildPluginKey(tc.given.repoName, tc.given.pluginName, tc.given.version)

				// then
				require.NoError(t, err)
				assert.Equal(t, tc.expKey, gotKey)
			})
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			name   string
			given  given
			expErr string
		}{
			{
				name: "should report error when all properties are empty",
				given: given{
					repoName:   "",
					pluginName: "",
				},
				expErr: heredoc.Doc(`
					2 errors occurred:
						* repository name is required
						* plugin name is required`),
			},
			{
				name: "should report error repo name is empty",
				given: given{
					repoName:   "",
					pluginName: "kubectl",
				},
				expErr: heredoc.Doc(`
					1 error occurred:
						* repository name is required`),
			},
			{
				name: "should report error plugin name is empty",
				given: given{
					repoName:   "repository",
					pluginName: "",
				},
				expErr: heredoc.Doc(`
					1 error occurred:
						* plugin name is required`),
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// when
				gotKey, err := plugin.BuildPluginKey(tc.given.repoName, tc.given.pluginName, tc.given.version)

				// then
				assert.Empty(t, gotKey)
				assert.EqualError(t, err, tc.expErr)
			})
		}
	})
}

func TestDecomposePluginKey(t *testing.T) {
	type expected struct {
		repoName   string
		pluginName string
		version    string
	}

	t.Run("Success", func(t *testing.T) {
		tests := []struct {
			name     string
			givenKey string
			exp      expected
		}{
			{
				name:     "should decompose full key name",
				givenKey: "hakuna/kubectl@v1.0.0",
				exp: expected{
					repoName:   "hakuna",
					pluginName: "kubectl",
					version:    "v1.0.0",
				},
			},
			{
				name:     "should decompose key without version",
				givenKey: "hakuna/kubectl",
				exp: expected{
					repoName:   "hakuna",
					pluginName: "kubectl",
					version:    "",
				},
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// when
				gotRepo, gotPlugin, gotVer, err := plugin.DecomposePluginKey(tc.givenKey)

				// then
				require.NoError(t, err)

				assert.Equal(t, tc.exp.repoName, gotRepo)
				assert.Equal(t, tc.exp.pluginName, gotPlugin)
				assert.Equal(t, tc.exp.version, gotVer)
			})
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			name     string
			givenKey string
			expErr   string
		}{
			{
				name:     "should report error of wrong pattern",
				givenKey: "kubectl@v1.0.0",
				expErr:   `plugin key "kubectl@v1.0.0" doesn't follow required {repo_name}/{plugin_name} syntax`,
			},
			{
				name:     "should report error about missing plugin name",
				givenKey: "test/@v1.0.0",
				expErr: heredoc.Doc(`
					1 error occurred:
						* plugin name is required`),
			},
			{
				name:     "should report error about missing repo name",
				givenKey: "/kubectl@v1.0.0",
				expErr: heredoc.Doc(`
					1 error occurred:
						* repository name is required`),
			},
			{
				name:     "should report error about missing repo and plugin names",
				givenKey: "/@v1.0.0",
				expErr: heredoc.Doc(`
					2 errors occurred:
						* repository name is required
						* plugin name is required`),
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// when
				gotRepo, gotPlugin, gotVer, err := plugin.DecomposePluginKey(tc.givenKey)

				// then
				assert.Empty(t, gotRepo)
				assert.Empty(t, gotPlugin)
				assert.Empty(t, gotVer)

				assert.EqualError(t, err, tc.expErr)
			})
		}
	})
}

// TestBuildAndDecomposePluginKey test that Decompose and Build respect the same contract.
func TestBuildAndDecomposePluginKey(t *testing.T) {
	// given
	const key = "botkube/kubectl@v1.0.0"

	// when
	repo, pluginName, ver, err := plugin.DecomposePluginKey(key)
	// then
	require.NoError(t, err)

	// when
	gotKey, err := plugin.BuildPluginKey(repo, pluginName, ver)

	//then
	require.NoError(t, err)
	assert.Equal(t, key, gotKey)
}
