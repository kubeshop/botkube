package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	// AllowedEventType K8s event types allowed to forward
	AllowedEventType EventType = WarningEvent

	// CreateEvent when resource is created
	CreateEvent EventType = "create"
	// UpdateEvent when resource is updated
	UpdateEvent EventType = "update"
	// DeleteEvent when resource deleted
	DeleteEvent EventType = "delete"
	// ErrorEvent on errors in resources
	ErrorEvent EventType = "error"
	// WarningEvent for warning events
	WarningEvent EventType = "warning"
	// AllEvent to watch all events
	AllEvent EventType = "all"
)

// EventType to watch
type EventType string

// ConfigFileName is a name of botkube configuration file
var ConfigFileName = "config.yaml"

// Notify flag to toggle event notification
var Notify = true

// Config structure of configuration yaml file
type Config struct {
	Resources       []Resource
	Recommendations bool
	Communications  Communications
	Settings        Settings
}

// Resource contains resources to watch
type Resource struct {
	Name       string
	Namespaces []string
	Events     []EventType
}

// Communications channels to send events to
type Communications struct {
	Slack         Slack
	ElasticSearch ElasticSearch
	Mattermost    Mattermost
}

// Slack configuration to authentication and send notifications
type Slack struct {
	Enabled bool
	Channel string
	Token   string `yaml:",omitempty"`
}

// ElasticSearch config auth settings
type ElasticSearch struct {
	Enabled  bool
	Username string
	Password string `yaml:",omitempty"`
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

// Mattermost configuration to authentication and send notifications
type Mattermost struct {
	Enabled bool
	URL     string
	Token   string
	Team    string
	Channel string
}

// Settings for multicluster support
type Settings struct {
	ClusterName     string
	AllowKubectl    bool
	UpgradeNotifier bool `yaml:"upgradeNotifier"`
}

func (eventType EventType) String() string {
	return string(eventType)
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
