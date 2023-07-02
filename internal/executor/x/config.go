package x

import (
	"github.com/kubeshop/botkube/internal/executor/x/getter"
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
)

// Config holds x plugin configuration.
type Config struct {
	Templates []getter.Source `yaml:"templates"`
	TmpDir    plugin.TmpDir   `yaml:"tmpDir"`
	Logger    config.Logger
}

// GetPluginDependencies returns x plugin dependencies.
func GetPluginDependencies() map[string]api.Dependency {
	return map[string]api.Dependency{
		"eget": {
			URLs: map[string]string{
				"windows/amd64": "https://github.com/zyedidia/eget/releases/download/v1.3.3/eget-1.3.3-windows_amd64.zip//eget-1.3.3-windows_amd64",
				"darwin/amd64":  "https://github.com/zyedidia/eget/releases/download/v1.3.3/eget-1.3.3-darwin_amd64.tar.gz//eget-1.3.3-darwin_amd64",
				"darwin/arm64":  "https://github.com/zyedidia/eget/releases/download/v1.3.3/eget-1.3.3-darwin_arm64.tar.gz//eget-1.3.3-darwin_arm64",
				"linux/amd64":   "https://github.com/zyedidia/eget/releases/download/v1.3.3/eget-1.3.3-linux_amd64.tar.gz//eget-1.3.3-linux_amd64",
				"linux/arm64":   "https://github.com/zyedidia/eget/releases/download/v1.3.3/eget-1.3.3-linux_arm64.tar.gz//eget-1.3.3-linux_arm64",
				"linux/386":     "https://github.com/zyedidia/eget/releases/download/v1.3.3/eget-1.3.3-linux_386.tar.gz//eget-1.3.3-linux_386",
			},
		},
	}
}
