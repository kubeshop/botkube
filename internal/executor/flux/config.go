package flux

import (
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/config"
)

// Config holds Flux executor configuration.
type Config struct {
	Logger config.Logger `yaml:"logger"`
	TmpDir plugin.TmpDir `yaml:"tmpDir"`
	GitHub struct {
		Auth struct {
			// GitHub access token.
			// Instructions for token creation: https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/#creating-a-token.
			// Lack of token may limit functionality, e.g., adding comments to pull requests or approving them.
			AccessToken string `yaml:"accessToken"`
		} `yaml:"auth"`
	} `yaml:"github"`
}
