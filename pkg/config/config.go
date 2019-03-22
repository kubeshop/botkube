package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// ConfigFileName is a name of botkube configuration file
var ConfigFileName = "config.yaml"

// Notify flag to toggle event notification
var Notify = true

// Config structure of configuration yaml file
type Config struct {
	Resources       []Resource
	Recommendations bool
	Communications  Communications
	Events          K8SEvents
	Settings        Settings
}

// K8SEvents contains event types
type K8SEvents struct {
	Types []string
}

// Resource contains resources to watch
type Resource struct {
	Name       string
	Namespaces []string
	Events     []string
}

// Communications channels to send events to
type Communications struct {
	Slack Slack

	ElasticSearch ElasticSearch
}

// Slack configuration to authentication and send notifications
type Slack struct {
	Enable  bool
	Channel string
	Token   string
}

// ElasticSearch config auth settings
type ElasticSearch struct {
	Enable   bool
	Username string
	Password string
	Server   string
	Index    Index
}

// Index settings for ELS
type Index struct {
	Name     string
	Type     string
	Shards   int
	Replicas int
}

// Settings for multicluster support
type Settings struct {
	ClusterName  string
	AllowKubectl bool
}

// New returns new Config
func New() (*Config, error) {
	c := &Config{}
	configPath := os.Getenv("CONFIG_PATH")
	configFile := filepath.Join(configPath, ConfigFileName)
	file, err := os.Open(configFile)
	defer file.Close()
	if err != nil {
		return c, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		yaml.Unmarshal(b, c)
	}
	return c, nil
}
