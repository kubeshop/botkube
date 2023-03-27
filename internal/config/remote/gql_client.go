package remote

import (
	"net/http"
	"time"

	"github.com/hasura/go-graphql-client"
)

const (
	defaultTimeout = 30 * time.Second
	//nolint:gosec // warns us about 'Potential hardcoded credentials' but there is no security issue here
	apiKeyHeaderName = "X-API-Key"
)

// Option define GraphQL client option.
type Option func(*Gql)

// WithEndpoint configures ApiURL for GraphQL endpoint.
func WithEndpoint(url string) Option {
	return func(client *Gql) {
		client.endpoint = url
	}
}

// WithAPIKey configures API key for GraphQL endpoint.
func WithAPIKey(apiKey string) Option {
	return func(client *Gql) {
		client.apiKey = apiKey
	}
}

// WithDeploymentID configures deployment id for GraphQL endpoint.
func WithDeploymentID(id string) Option {
	return func(client *Gql) {
		client.deployID = id
	}
}

// Gql defines GraphQL client data structure.
type Gql struct {
	cli      *graphql.Client
	endpoint string
	apiKey   string
	deployID string
}

// NewGqlClient initializes GraphQL client.
func NewGqlClient(options ...Option) *Gql {
	c := &Gql{}
	for _, opt := range options {
		opt(c)
	}

	httpCli := &http.Client{
		Transport: newAPIKeySecuredTransport(c.apiKey),
		Timeout:   defaultTimeout,
	}

	c.cli = graphql.NewClient(c.endpoint, httpCli)
	return c
}

// NewDefaultGqlClient initializes GraphQL client with default options.
func NewDefaultGqlClient(remoteCfg Config) *Gql {
	return NewGqlClient(
		WithEndpoint(remoteCfg.Endpoint),
		WithAPIKey(remoteCfg.APIKey),
		WithDeploymentID(remoteCfg.Identifier),
	)
}

// DeploymentID returns deployment ID.
func (g *Gql) DeploymentID() string {
	return g.deployID
}

// Client returns GraphQL client.
func (g *Gql) Client() *graphql.Client {
	return g.cli
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

// RoundTrip adds API key to request header and executes RoundTrip for the underlying transport.
func (t *apiKeySecuredTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.apiKey != "" {
		req.Header.Set(apiKeyHeaderName, t.apiKey)
	}
	return t.transport.RoundTrip(req)
}
