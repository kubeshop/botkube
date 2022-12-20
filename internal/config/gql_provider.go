package config

import (
	"context"
	"fmt"
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
func (g *GqlProvider) Configs(ctx context.Context) ([]string, error) {
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
	temp, err := os.CreateTemp("/tmp", fmt.Sprintf("botkube-%s", d))
	if err != nil {
		return nil, errors.Wrapf(err, "while creating configuration yaml file for deployment with id %s", d)
	}
	_, err = temp.Write(conf)
	if err != nil {
		return nil, errors.Wrapf(err, "while adding configuration to %s for deployment with id %s", temp.Name(), d)
	}
	return []string{temp.Name()}, nil
}
