package config

import (
	"context"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

const (
	GqlProviderEndpointEnvKey   = "CONFIG_PROVIDER_ENDPOINT"
	GqlProviderIdentifierEnvKey = "CONFIG_PROVIDER_IDENTIFIER"
	//nolint:gosec // warns us about 'Potential hardcoded credentials' but there is no security issue here
	GqlProviderAPIKeyEnvKey = "CONFIG_PROVIDER_API_KEY"
)

// GqlProvider is GraphQL provider
type GqlProvider struct {
	GqlClient GqlClient
}

// NewGqlProvider initializes new GraphQL config source provider
func NewGqlProvider(gql GqlClient) *GqlProvider {
	return &GqlProvider{GqlClient: gql}
}

// Configs returns list of config files
func (g *GqlProvider) Configs(ctx context.Context) (YAMLFiles, error) {
	deployment, err := g.GqlClient.GetDeployment(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "while getting deployment")
	}
	conf, err := yaml.JSONToYAML([]byte(deployment.BotkubeConfig))
	if err != nil {
		return nil, errors.Wrapf(err, "while converting json to yaml for deployment")
	}

	return [][]byte{
		conf,
	}, nil
}
