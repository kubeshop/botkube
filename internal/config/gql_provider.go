package config

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
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
	d := os.Getenv("CONFIG_SOURCE_IDENTIFIER")
	if d == "" {
		return nil, nil
	}
	deployment, err := g.GqlClient.GetDeployment(ctx, d)
	if err != nil {
		return nil, errors.Wrapf(err, "while getting deployment with id %s", d)
	}
	conf, err := yaml.JSONToYAML([]byte(deployment.BotkubeConfig))
	if err != nil {
		return nil, errors.Wrapf(err, "while converting json to yaml for deployment with id %s", d)
	}

	return [][]byte{
		conf,
	}, nil
}
