package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
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
	// NormalEvent for Normal events
	NormalEvent EventType = "normal"
	// InfoEvent for insignificant Info events
	InfoEvent EventType = "info"
	// AllEvent to watch all events
	AllEvent EventType = "all"
	// ShortNotify is the Default NotifType
	ShortNotify NotifType = "short"
	// LongNotify for short events notification
	LongNotify NotifType = "long"
)

// EventType to watch
type EventType string

// ResourceConfigFileName is a name of botkube resource configuration file
var ResourceConfigFileName = "resource_config.yaml"

// CommunicationConfigFileName is a name of botkube communication configuration file
var CommunicationConfigFileName = "comm_config.yaml"

// Notify flag to toggle event notification
var Notify = true

// NotifType to change notification type
type NotifType string

// Config structure of configuration yaml file
type Config struct {
	Resources       []Resource
	Recommendations bool
	Communications  Communications
	Settings        Settings
}

// Resource contains resources to watch
type Resource struct {
	Name          string
	Namespaces    Namespaces
	Events        []EventType
	UpdateSetting UpdateSetting `yaml:"updateSetting"`
}

//UpdateSetting struct defines updateEvent fields specification
type UpdateSetting struct {
	Fields      []string
	IncludeDiff bool `yaml:"includeDiff"`
}

// Namespaces contains namespaces to include and ignore
// Include contains a list of namespaces to be watched,
//  - "all" to watch all the namespaces
// Ignore contains a list of namespaces to be ignored when all namespaces are included
// It is an optional (omitempty) field which is tandem with Include [all]
// example : include [all], ignore [x,y,z]
type Namespaces struct {
	Include []string
	Ignore  []string `yaml:",omitempty"`
}

// Communications channels to send events to
type Communications struct {
	Slack         Slack
	ElasticSearch ElasticSearch
	Mattermost    Mattermost
	Webhook       Webhook
}

// Slack configuration to authentication and send notifications
type Slack struct {
	Enabled   bool
	Channel   string
	NotifType NotifType `yaml:",omitempty"`
	Token     string    `yaml:",omitempty"`
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
	Enabled   bool
	URL       string
	Token     string
	Team      string
	Channel   string
	NotifType NotifType `yaml:",omitempty"`
}

// Webhook configuration to send notifications
type Webhook struct {
	Enabled bool
	URL     string
}

// Settings for multicluster support
type Settings struct {
	ClusterName     string
	AllowKubectl    bool
	RestrictAccess  bool `yaml:"restrictAccess"`
	ConfigWatcher   bool
	UpgradeNotifier bool `yaml:"upgradeNotifier"`
}

func (eventType EventType) String() string {
	return string(eventType)
}

// New returns new Config
func New() (*Config, error) {
	c := &Config{}
	configPath := os.Getenv("CONFIG_PATH")
	resourceConfigFilePath := filepath.Join(configPath, ResourceConfigFileName)
	resourceConfigFile, err := os.Open(resourceConfigFilePath)
	defer resourceConfigFile.Close()
	if err != nil {
		return c, err
	}

	b, err := ioutil.ReadAll(resourceConfigFile)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		yaml.Unmarshal(b, c)
	}

	communicationConfigFilePath := filepath.Join(configPath, CommunicationConfigFileName)
	communicationConfigFile, err := os.Open(communicationConfigFilePath)
	defer communicationConfigFile.Close()
	if err != nil {
		return c, err
	}

	b, err = ioutil.ReadAll(communicationConfigFile)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		yaml.Unmarshal(b, c)
	}
	return c, nil
}
