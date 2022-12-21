package config

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
				gotRepo, gotPlugin, gotVer, err := DecomposePluginKey(tc.givenKey)

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
				expErr:   `plugin key "kubectl@v1.0.0" doesn't follow the required {repo_name}/{plugin_name} syntax`,
			},
			{
				name:     "should report error about missing plugin name",
				givenKey: "test/@v1.0.0",
				expErr: heredoc.Doc(`
					doesn't follow the required {repo_name}/{plugin_name} syntax: 1 error occurred:
						* plugin name is required`),
			},
			{
				name:     "should report error about missing repo name",
				givenKey: "/kubectl@v1.0.0",
				expErr: heredoc.Doc(`
					doesn't follow the required {repo_name}/{plugin_name} syntax: 1 error occurred:
						* repository name is required`),
			},
			{
				name:     "should report error about missing repo and plugin names",
				givenKey: "/@v1.0.0",
				expErr: heredoc.Doc(`
					doesn't follow the required {repo_name}/{plugin_name} syntax: 2 errors occurred:
						* repository name is required
						* plugin name is required`),
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// when
				gotRepo, gotPlugin, gotVer, err := DecomposePluginKey(tc.givenKey)

				// then
				assert.Empty(t, gotRepo)
				assert.Empty(t, gotPlugin)
				assert.Empty(t, gotVer)

				assert.EqualError(t, err, tc.expErr)
			})
		}
	})
}
