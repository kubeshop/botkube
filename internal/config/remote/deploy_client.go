package remote

import (
	"context"
	"fmt"

	"github.com/hasura/go-graphql-client"
)

// GraphQLClient defines GraphQL client.
type GraphQLClient interface {
	Client() *graphql.Client
	DeploymentID() string
}

// DeploymentClient defines GraphQL client for Deployment.
type DeploymentClient struct {
	client GraphQLClient
}

// NewDeploymentClient initializes GraphQL client.
func NewDeploymentClient(client GraphQLClient) *DeploymentClient {
	return &DeploymentClient{client: client}
}

// Deployment returns deployment with Botkube configuration.
type Deployment struct {
	ResourceVersion int
	YAMLConfig      string
}

// GetConfigWithResourceVersion retrieves deployment by id.
func (g *DeploymentClient) GetConfigWithResourceVersion(ctx context.Context) (Deployment, error) {
	var query struct {
		Deployment Deployment `graphql:"deployment(id: $id)"`
	}
	deployID := g.client.DeploymentID()
	variables := map[string]interface{}{
		"id": graphql.ID(deployID),
	}
	err := g.client.Client().Query(ctx, &query, variables)
	if err != nil {
		return Deployment{}, fmt.Errorf("while getting config with resource version for %q: %w", deployID, err)
	}
	return query.Deployment, nil
}

// GetResourceVersion retrieves resource version for Deployment.
func (g *DeploymentClient) GetResourceVersion(ctx context.Context) (int, error) {
	var query struct {
		Deployment struct {
			ResourceVersion int
		} `graphql:"deployment(id: $id)"`
	}
	deployID := g.client.DeploymentID()
	variables := map[string]interface{}{
		"id": graphql.ID(deployID),
	}
	err := g.client.Client().Query(ctx, &query, variables)
	if err != nil {
		return 0, fmt.Errorf("while querying deployment details for %q: %w", deployID, err)
	}
	return query.Deployment.ResourceVersion, nil
}
