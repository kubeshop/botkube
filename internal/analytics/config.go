package analytics

import "github.com/google/uuid"

type Config struct {
	InstallationID string `yaml:"installationID"`
}

func NewConfig() Config {
	return Config{
		InstallationID: uuid.NewString(),
	}
}
