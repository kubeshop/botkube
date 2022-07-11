package analytics

import "github.com/google/uuid"

type Config struct {
	InstallationID string `json:"installationID"`
}

func NewConfig() Config {
	return Config{
		InstallationID: uuid.NewString(),
	}
}
