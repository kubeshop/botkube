package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
)

var (
	configFilePath = filepath.Clean(filepath.Join(homedir.HomeDir(), ".botkube", "cloud.json"))
	loginCmd       = heredoc.WithCLIName(`login with: <cli> login`, cli.Name)
)

// Config is botkube cli config
type Config struct {
	Token string `json:"token"`
}

// New creates new Config from local data
func New() (*Config, error) {
	c := &Config{}
	err := c.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v\n%s", err, loginCmd)
	}
	return c, nil
}

// Save saves Config to local FS
func (c *Config) Save() error {
	if _, err := os.Stat(filepath.Dir(configFilePath)); os.IsNotExist(err) {
		// #nosec G301
		err := os.MkdirAll(filepath.Dir(configFilePath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %v", err)
	}

	// #nosec G306
	err = os.WriteFile(configFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
}

// Read reads Config from local FS
func (c *Config) Read() error {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = json.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file: %v", err)
	}

	return nil
}
