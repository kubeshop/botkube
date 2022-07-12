package analytics

import "github.com/google/uuid"

// Config contains configuration parameters for analytics collection.
type Config struct {
	InstallationID string `yaml:"installationID"`
}

// NewConfig creates new Config instance.
func NewConfig() Config {
	return Config{
		InstallationID: uuid.NewString(),
	}
}
