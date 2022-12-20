package config

import (
	"context"

	"github.com/hasura/go-graphql-client"
)

// GqlClient GraphQL client
type GqlClient interface {
	GetDeployment(ctx context.Context, id string) (Deployment, error)
}

// Option GraphQL client options
type Option func(*Gql)

// WithAPIURL configures ApiURL for GraphQL endpoint
func WithAPIURL(url string) Option {
	return func(client *Gql) {
		client.APIURL = url
	}
}

// Gql GraphQL client data structure
type Gql struct {
	Gql    *graphql.Client
	APIURL string
}

// NewGqlClient initializes GraphQL client
func NewGqlClient(options ...Option) *Gql {
	c := &Gql{}
	for _, opt := range options {
		opt(c)
	}
	return &Gql{
		Gql: graphql.NewClient(c.APIURL, nil),
	}
}

// Deployment returns deployment with Botkube configuration
type Deployment struct {
	BotkubeConfig string
}

// GetDeployment retrieves deployment by id
func (c *Gql) GetDeployment(ctx context.Context, id string) (Deployment, error) {
	var query struct {
		Deployment Deployment `graphql:"deployment(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(id),
	}
	err := c.Gql.Query(ctx, &query, variables)
	if err != nil {
		return Deployment{}, err
	}
	return query.Deployment, nil
}
