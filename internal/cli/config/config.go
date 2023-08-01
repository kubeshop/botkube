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
	configFilePath = filepath.Join(homedir.HomeDir(), ".botkube", "cloud.json")
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

const (
	dirPerms  = 0755
	filePerms = 0644
)

// Save saves Config to local FS
func (c *Config) Save() error {
	cfgFileDir := filepath.Clean(filepath.Dir(configFilePath))
	cfgFilePath := filepath.Clean(configFilePath)
	if _, err := os.Stat(cfgFileDir); os.IsNotExist(err) {
		err = os.MkdirAll(cfgFileDir, dirPerms)
		if err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %v", err)
	}

	// #nosec G306
	err = os.WriteFile(cfgFilePath, data, filePerms)
	if err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
}

// Read reads Config from local FS
func (c *Config) Read() error {
	data, err := os.ReadFile(filepath.Clean(configFilePath))
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = json.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file: %v", err)
	}

	return nil
}
