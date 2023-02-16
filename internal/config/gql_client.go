package config

import (
	"context"
	"fmt"

	"github.com/hasura/go-graphql-client"

	gql "github.com/kubeshop/botkube/internal/graphql"
)

// DeploymentClient defines GraphQL client.
type DeploymentClient interface {
	GetDeployment(ctx context.Context) (Deployment, error)
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
	BotkubeConfig string
}

// GetDeployment retrieves deployment by id.
func (g *Gql) GetDeployment(ctx context.Context) (Deployment, error) {
	var query struct {
		Deployment Deployment `graphql:"deployment(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(g.deploymentID),
	}
	err := g.client.Cli.Query(ctx, &query, variables)
	if err != nil {
		return Deployment{}, fmt.Errorf("while querying deployment details for %q: %w", g.deploymentID, err)
	}
	return query.Deployment, nil
}
