package config

import (
	"github.com/kubeshop/botkube/internal/graphql"
	"github.com/kubeshop/botkube/pkg/config"
	"os"
)

// GetProvider resolves and returns paths for config files.
// It reads them the 'BOTKUBE_CONFIG_PATHS' env variable. If not found, then it uses '--config' flag.
func GetProvider(gql *graphql.Gql) config.Provider {
	if _, provided := os.LookupEnv(graphql.GqlProviderIdentifierEnvKey); provided {
		dc := NewDeploymentClient(gql)
		return NewGqlProvider(dc)
	}

	if os.Getenv(EnvProviderConfigPathsEnvKey) != "" {
		return NewEnvProvider()
	}

	return NewFileSystemProvider(configPathsFlag)
}

