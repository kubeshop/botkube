package config

import (
	"github.com/kubeshop/botkube/pkg/config"
	"os"
)

// GetProvider resolves and returns paths for config files.
// It reads them the 'BOTKUBE_CONFIG_PATHS' env variable. If not found, then it uses '--config' flag.
func GetProvider(remoteCfgSyncEnabled bool, deployClient DeploymentClient) config.Provider {
	if remoteCfgSyncEnabled {
		return NewGqlProvider(deployClient)
	}

	if os.Getenv(EnvProviderConfigPathsEnvKey) != "" {
		return NewEnvProvider()
	}

	return NewFileSystemProvider(configPathsFlag)
}

