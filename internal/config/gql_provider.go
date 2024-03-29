package config

import (
	"context"

	"github.com/pkg/errors"

	"github.com/kubeshop/botkube/internal/config/remote"
	"github.com/kubeshop/botkube/pkg/config"
)

type DeploymentClient interface {
	GetConfigWithResourceVersion(ctx context.Context) (remote.Deployment, error)
}

// GqlProvider is GraphQL provider
type GqlProvider struct {
	client DeploymentClient
}

// NewGqlProvider initializes new GraphQL config source provider
func NewGqlProvider(dc DeploymentClient) *GqlProvider {
	return &GqlProvider{client: dc}
}

// Configs returns list of config files
func (g *GqlProvider) Configs(ctx context.Context) (config.YAMLFiles, int, error) {
	deployment, err := g.client.GetConfigWithResourceVersion(ctx)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "while getting deployment")
	}

	return [][]byte{
		[]byte(deployment.YAMLConfig),
	}, deployment.ResourceVersion, nil
}
