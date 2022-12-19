package config

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"sigs.k8s.io/yaml"
)

type GqlProvider struct {
	gqlClient *GqlClient
}

func NewGqlProvider(gql *GqlClient) *GqlProvider {
	return &GqlProvider{gqlClient: gql}
}
func (g *GqlProvider) Configs(ctx context.Context) ([]string, error) {
	d := os.Getenv("CONFIG_SOURCE_IDENTIFIER")
	if d == "" {
		return nil, nil
	}
	deployment, err := g.gqlClient.GetDeployment(ctx, d)
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
