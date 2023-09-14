package remote

import "os"

const (
	// ProviderEndpointEnvKey holds config provider endpoint.
	//nolint:gosec // Potential hardcoded credentials
	ProviderEndpointEnvKey = "CONFIG_PROVIDER_ENDPOINT"
	// ProviderIdentifierEnvKey holds config provider identifier.
	ProviderIdentifierEnvKey = "CONFIG_PROVIDER_IDENTIFIER"
	// ProviderAPIKeyEnvKey holds config provider API key.
	//nolint:gosec // warns us about 'Potential hardcoded credentials' but there is no security issue here
	ProviderAPIKeyEnvKey = "CONFIG_PROVIDER_API_KEY"
)

// Config holds configuration for remote configuration.
type Config struct {
	Endpoint   string
	Identifier string
	APIKey     string
}

// GetConfig returns remote configuration if it is set.
func GetConfig() (Config, bool) {
	if os.Getenv(ProviderIdentifierEnvKey) == "" {
		return Config{}, false
	}

	return Config{
		Endpoint:   os.Getenv(ProviderEndpointEnvKey),
		Identifier: os.Getenv(ProviderIdentifierEnvKey),
		APIKey:     os.Getenv(ProviderAPIKeyEnvKey),
	}, true
}
