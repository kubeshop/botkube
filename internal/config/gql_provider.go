package config

import (
	"context"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// GqlProvider is GraphQL provider
type GqlProvider struct {
	client DeploymentClient
}

// NewGqlProvider initializes new GraphQL config source provider
func NewGqlProvider(dc DeploymentClient) *GqlProvider {
	return &GqlProvider{client: dc}
}

// Configs returns list of config files
func (g *GqlProvider) Configs(ctx context.Context) (YAMLFiles, error) {
	deployment, err := g.client.GetDeployment(ctx)
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
