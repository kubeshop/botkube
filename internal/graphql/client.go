package graphql

import (
	"net/http"
	"os"
	"time"

	"github.com/hasura/go-graphql-client"
)

const (
	defaultTimeout = 30 * time.Second
	//nolint:gosec // warns us about 'Potential hardcoded credentials' but there is no security issue here
	apiKeyHeaderName = "X-API-Key"

	GqlProviderEndpointEnvKey   = "CONFIG_PROVIDER_ENDPOINT"
	GqlProviderIdentifierEnvKey = "CONFIG_PROVIDER_IDENTIFIER"
	//nolint:gosec // warns us about 'Potential hardcoded credentials' but there is no security issue here
	GqlProviderAPIKeyEnvKey = "CONFIG_PROVIDER_API_KEY"
)

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

	// skip client creation when not requested
	if c.Endpoint == "" {
		return c
	}

	httpCli := &http.Client{
		Transport: newAPIKeySecuredTransport(c.APIKey),
		Timeout:   defaultTimeout,
	}

	c.Cli = graphql.NewClient(c.Endpoint, httpCli)
	return c
}

func NewDefaultGqlClient() *Gql {
	return NewGqlClient(
		WithEndpoint(os.Getenv(GqlProviderEndpointEnvKey)),
		WithAPIKey(os.Getenv(GqlProviderAPIKeyEnvKey)),
		WithDeploymentID(os.Getenv(GqlProviderIdentifierEnvKey)),
	)
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
