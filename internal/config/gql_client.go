package config

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hasura/go-graphql-client"
)

const (
	defaultTimeout = 30 * time.Second
	//nolint:gosec // warns us about 'Potential hardcoded credentials' but there is no security issue here
	apiKeyHeaderName = "X-API-Key"
)

// GqlClient defines GraphQL client.
type GqlClient interface {
	GetDeployment(ctx context.Context) (Deployment, error)
}

// Option define GraphQL client option.
type Option func(*Gql)

// WithEndpoint configures ApiURL for GraphQL endpoint.
func WithEndpoint(url string) Option {
	return func(client *Gql) {
		client.Endpoint = url
	}
}

// WithAPIKey configures API key for GraphQL endpoint.
func WithAPIKey(apiKey string) Option {
	return func(client *Gql) {
		client.APIKey = apiKey
	}
}

// WithDeploymentID configures deployment id for GraphQL endpoint.
func WithDeploymentID(id string) Option {
	return func(client *Gql) {
		client.DeploymentID = id
	}
}

// Gql defines GraphQL client data structure.
type Gql struct {
	Cli          *graphql.Client
	Endpoint     string
	APIKey       string
	DeploymentID string
}

// NewGqlClient initializes GraphQL client.
func NewGqlClient(options ...Option) *Gql {
	c := &Gql{}
	for _, opt := range options {
		opt(c)
	}

	httpCli := &http.Client{
		Transport: newAPIKeySecuredTransport(c.APIKey),
		Timeout:   defaultTimeout,
	}

	c.Cli = graphql.NewClient(c.Endpoint, httpCli)
	return c
}

// Deployment returns deployment with Botkube configuration.
type Deployment struct {
	BotkubeConfig string
}

// GetDeployment retrieves deployment by id.
func (c *Gql) GetDeployment(ctx context.Context) (Deployment, error) {
	var query struct {
		Deployment Deployment `graphql:"deployment(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(c.DeploymentID),
	}
	err := c.Cli.Query(ctx, &query, variables)
	if err != nil {
		return Deployment{}, fmt.Errorf("while querying deployment details for %q: %w", c.DeploymentID, err)
	}
	return query.Deployment, nil
}

type apiKeySecuredTransport struct {
	apiKey    string
	transport *http.Transport
}

func newAPIKeySecuredTransport(apiKey string) *apiKeySecuredTransport {
	return &apiKeySecuredTransport{
		apiKey:    apiKey,
		transport: http.DefaultTransport.(*http.Transport).Clone(),
	}
}

func (t *apiKeySecuredTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.apiKey != "" {
		req.Header.Set(apiKeyHeaderName, t.apiKey)
	}
	return t.transport.RoundTrip(req)
}
