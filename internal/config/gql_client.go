package config

import (
	"context"
	"fmt"

	"github.com/hasura/go-graphql-client"

	gql "github.com/kubeshop/botkube/internal/graphql"
)

// DeploymentClient defines GraphQL client.
type DeploymentClient interface {
	GetConfigWithResourceVersion(ctx context.Context) (Deployment, error)
}

// Gql defines GraphQL client data structure.
type Gql struct {
	client       *gql.Gql
	deploymentID string
}

// NewDeploymentClient initializes GraphQL client.
func NewDeploymentClient(client *gql.Gql) *Gql {
	return &Gql{client: client, deploymentID: client.DeploymentID}
}

// Deployment returns deployment with Botkube configuration.
type Deployment struct {
	ResourceVersion int
	YAMLConfig      string
}

// GetConfigWithResourceVersion retrieves deployment by id.
func (g *Gql) GetConfigWithResourceVersion(ctx context.Context) (Deployment, error) {
	var query struct {
		Deployment Deployment `graphql:"deployment(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(g.deploymentID),
	}
	err := g.client.Cli.Query(ctx, &query, variables)
	if err != nil {
		return Deployment{}, fmt.Errorf("while getting config with resource version for %q: %w", g.deploymentID, err)
	}
	return query.Deployment, nil
}

// GetResourceVersion retrieves resource version for Deployment.
func (g *Gql) GetResourceVersion(ctx context.Context) (int, error) {
	var query struct {
		Deployment struct {
			ResourceVersion int
		} `graphql:"deployment(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(g.deploymentID),
	}
	err := g.client.Cli.Query(ctx, &query, variables)
	if err != nil {
		return 0, fmt.Errorf("while querying deployment details for %q: %w", g.deploymentID, err)
	}
	return query.Deployment.ResourceVersion, nil
}
