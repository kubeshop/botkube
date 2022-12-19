package config

import (
	"context"
	"github.com/hasura/go-graphql-client"
)

type option func(*GqlClient)

func WithApiUrl(url string) option {
	return func(client *GqlClient) {
		client.ApiURL = url
	}
}

type GqlClient struct {
	Gql    *graphql.Client
	ApiURL string
}

func NewGqlClient(options ...option) *GqlClient {
	c := &GqlClient{}
	for _, opt := range options {
		opt(c)
	}
	return &GqlClient{
		Gql: graphql.NewClient(c.ApiURL, nil),
	}
}

type Deployment struct {
	BotkubeConfig string
}

func (c *GqlClient) GetDeployment(ctx context.Context, id string) (Deployment, error) {
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
