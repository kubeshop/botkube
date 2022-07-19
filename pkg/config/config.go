package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf"
	koanfyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"gopkg.in/yaml.v3"
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

	// Info level
	Info Level = "info"
	// Warn level
	Warn Level = "warn"
	// Debug level
	Debug Level = "debug"
	// Error level
	Error Level = "error"
	// Critical level
	Critical Level = "critical"

	communicationEnvVariablePrefix = "COMMUNICATIONS_"
	communicationConfigDelimiter   = "."
)

// EventType to watch
type EventType string

// Level type to store event levels
type Level string

// CommPlatformIntegration defines integrations with communication platforms.
type CommPlatformIntegration string

const (
	// SlackCommPlatformIntegration defines Slack integration.
	SlackCommPlatformIntegration CommPlatformIntegration = "slack"

	// MattermostCommPlatformIntegration defines Mattermost integration.
	MattermostCommPlatformIntegration CommPlatformIntegration = "mattermost"

	// TeamsCommPlatformIntegration defines Teams integration.
	TeamsCommPlatformIntegration CommPlatformIntegration = "teams"

	// DiscordCommPlatformIntegration defines Discord integration.
	DiscordCommPlatformIntegration CommPlatformIntegration = "discord"

	//ElasticsearchCommPlatformIntegration defines Elasticsearch integration.
	ElasticsearchCommPlatformIntegration CommPlatformIntegration = "elasticsearch"

	// WebhookCommPlatformIntegration defines an outgoing webhook integration.
	WebhookCommPlatformIntegration CommPlatformIntegration = "webhook"
)

// IntegrationType describes the type of integration with a communication platform.
type IntegrationType string

const (
	// BotIntegrationType describes two-way integration.
	BotIntegrationType IntegrationType = "bot"

	// SinkIntegrationType describes one-way integration.
	SinkIntegrationType IntegrationType = "sink"
)

// ResourceConfigFileName is a name of BotKube resource configuration file
var ResourceConfigFileName = "resource_config.yaml"

// CommunicationConfigFileName is a name of BotKube communication configuration file
var CommunicationConfigFileName = "comm_config.yaml"

// Notify flag to toggle event notification
var Notify = true

// NotifType to change notification type
type NotifType string

// Config structure of configuration yaml file
type Config struct {
	Resources       []Resource
	Recommendations bool
	Communications  CommunicationsConfig
	Settings        Settings
}

// Communications contains communication config
type Communications struct {
	Communications CommunicationsConfig
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
// It can also contain a * that would expand to zero or more arbitrary characters
// example : include [all], ignore [x,y,secret-ns-*]
type Namespaces struct {
	Include []string
	Ignore  []string `yaml:",omitempty"`
}

// CommunicationsConfig channels to send events to
type CommunicationsConfig struct {
	Slack         Slack
	Mattermost    Mattermost
	Discord       Discord
	Webhook       Webhook
	Teams         Teams
	ElasticSearch ElasticSearch
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
	Enabled       bool
	Username      string
	Password      string `yaml:",omitempty"`
	Server        string
	SkipTLSVerify bool       `yaml:"skipTLSVerify"`
	AWSSigning    AWSSigning `yaml:"awsSigning"`
	Index         Index
}

// AWSSigning contains AWS configurations
type AWSSigning struct {
	Enabled   bool
	AWSRegion string `yaml:"awsRegion"`
	RoleArn   string `yaml:"roleArn"`
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
	BotName   string `yaml:"botName"`
	URL       string
	Token     string
	Team      string
	Channel   string
	NotifType NotifType `yaml:",omitempty"`
}

// Teams creds for authentication with MS Teams
type Teams struct {
	Enabled     bool
	AppID       string `yaml:"appID,omitempty"`
	AppPassword string `yaml:"appPassword,omitempty"`
	Team        string
	Port        string
	MessagePath string    `yaml:"messagePath,omitempty"`
	NotifType   NotifType `yaml:",omitempty"`
}

// Discord configuration for authentication and send notifications
type Discord struct {
	Enabled   bool
	Token     string
	BotID     string
	Channel   string
	NotifType NotifType `yaml:",omitempty"`
}

// Webhook configuration to send notifications
type Webhook struct {
	Enabled bool
	URL     string
}

// Kubectl configuration for executing commands inside cluster
type Kubectl struct {
	Enabled          bool
	Commands         Commands
	DefaultNamespace string `yaml:"defaultNamespace"`
	RestrictAccess   bool   `yaml:"restrictAccess"`
}

// Commands allowed in bot
type Commands struct {
	Verbs     []string
	Resources []string
}

// Settings for multicluster support
type Settings struct {
	ClusterName     string
	Kubectl         Kubectl
	ConfigWatcher   bool
	UpgradeNotifier bool `yaml:"upgradeNotifier"`
}

func (eventType EventType) String() string {
	return string(eventType)
}

// NewCommunicationsConfig return new communication config object
func NewCommunicationsConfig() (*Communications, error) {
	configPath := os.Getenv("CONFIG_PATH")
	commCfgFilePath := filepath.Join(configPath, CommunicationConfigFileName)

	k := koanf.New(communicationConfigDelimiter)

	// Load base YAML config.
	if err := k.Load(file.Provider(filepath.Clean(commCfgFilePath)), koanfyaml.Parser()); err != nil {
		return nil, err
	}

	// Load environment variables and merge into the loaded config.
	err := k.Load(env.Provider(
		communicationEnvVariablePrefix,
		communicationConfigDelimiter,
		normalizeCommunicationConfigEnvName,
	), nil)
	if err != nil {
		return nil, err
	}

	var cfg Communications
	err = k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func normalizeCommunicationConfigEnvName(name string) string {
	name = strings.ToLower(name)
	return strings.ReplaceAll(name, "_", ".")
}

// Load loads new configuration from file.
func Load(dir string) (*Config, error) {
	resCfgFilePath := filepath.Join(dir, ResourceConfigFileName)
	rawCfg, err := os.ReadFile(filepath.Clean(resCfgFilePath))
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(rawCfg, cfg); err != nil {
		return nil, err
	}

	comm, err := NewCommunicationsConfig()
	if err != nil {
		return nil, err
	}
	cfg.Communications = comm.Communications

	return cfg, nil
}
