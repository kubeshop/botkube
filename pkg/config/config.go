package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/knadh/koanf"
	koanfyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/spf13/pflag"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/utils/strings/slices"
)

//go:embed default.yaml
var defaultConfiguration []byte

var configPathsFlag []string

const (
	configEnvVariablePrefix = "BOTKUBE_"
	configDelimiter         = "."
	camelCaseDelimiter      = "__"
	nestedFieldDelimiter    = "_"
)

const (
	// AllNamespaceIndicator represents a keyword for allowing all Kubernetes Namespaces.
	AllNamespaceIndicator = "all"
)

// EventType to watch
type EventType string

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
)

// Level type to store event levels
type Level string

const (
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
)

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

// NotificationType to change notification type
type NotificationType string

const (
	// ShortNotification is the default NotificationType
	ShortNotification NotificationType = "short"
	// LongNotification for short events notification
	LongNotification NotificationType = "long"
)

// Config structure of configuration yaml file
type Config struct {
	Sources        IndexableMap[Sources]        `yaml:"sources"`
	Executors      IndexableMap[Executors]      `yaml:"executors" validate:"required,eq=1"`
	Communications IndexableMap[Communications] `yaml:"communications"  validate:"required,eq=1"`

	Analytics Analytics `yaml:"analytics"`
	Settings  Settings  `yaml:"settings"`
}

// ChannelBindingsByName contains configuration bindings per channel.
type ChannelBindingsByName struct {
	Name     string      `yaml:"name"`
	Bindings BotBindings `yaml:"bindings"`
}

// ChannelBindingsByID contains configuration bindings per channel.
type ChannelBindingsByID struct {
	ID       string      `yaml:"id"`
	Bindings BotBindings `yaml:"bindings"`
}

// BotBindings contains configuration for possible Bot bindings.
type BotBindings struct {
	Sources   []string `yaml:"sources"`
	Executors []string `yaml:"executors"`
}

// SinkBindings contains configuration for possible Sink bindings.
type SinkBindings struct {
	Sources []string `yaml:"sources"`
}

// Sources contains configuration for BotKube app sources.
type Sources struct {
	Kubernetes      KubernetesSource `yaml:"kubernetes"`
	Recommendations bool             `yaml:"recommendations"`
}

// KubernetesSource contains configuration for Kubernetes sources.
type KubernetesSource struct {
	Resources []Resource `yaml:"resources"`
}

// Executors contains executors configuration parameters.
type Executors struct {
	Kubectl Kubectl `yaml:"kubectl"`
}

// Analytics contains configuration parameters for analytics collection.
type Analytics struct {
	InstallationID string `yaml:"installationID"`
	Disable        bool   `yaml:"disable"`
}

// Resource contains resources to watch
type Resource struct {
	Name          string        `yaml:"name"`
	Namespaces    Namespaces    `yaml:"namespaces"`
	Events        []EventType   `yaml:"events"`
	UpdateSetting UpdateSetting `yaml:"updateSetting"`
}

//UpdateSetting struct defines updateEvent fields specification
type UpdateSetting struct {
	Fields      []string `yaml:"fields"`
	IncludeDiff bool     `yaml:"includeDiff"`
}

// Namespaces contains namespaces to include and ignore
// Include contains a list of namespaces to be watched,
//  - "all" to watch all the namespaces
// Ignore contains a list of namespaces to be ignored when all namespaces are included
// It is an optional (omitempty) field which is tandem with Include [all]
// It can also contain a * that would expand to zero or more arbitrary characters
// example : include [all], ignore [x,y,secret-ns-*]
type Namespaces struct {
	Include []string `yaml:"include"`
	Ignore  []string `yaml:"ignore,omitempty"`
}

// IsAllowed checks if a given Namespace is allowed based on the config.
// Copied from https://github.com/kubeshop/botkube/blob/b6b7d449278617d40f05d0792b419a7692ad980f/pkg/filterengine/filters/namespace_checker.go#L54-L76
// TODO(https://github.com/kubeshop/botkube/issues/596): adjust contract.
func (n *Namespaces) IsAllowed(givenNs string) bool {
	if n == nil {
		return false
	}

	isAll := len(n.Include) == 1 && n.Include[0] == AllNamespaceIndicator

	// Ignore contains a list of namespaces to be ignored when 'all' namespaces are included.
	// It can also contain a * that would expand to zero or more arbitrary characters.
	// Example: include [all], ignore [x,y,secret-ns-*]
	if isAll && len(n.Ignore) > 0 {
		for _, ignoredNamespace := range n.Ignore {
			// exact match
			if ignoredNamespace == givenNs {
				return false
			}

			// regexp
			if strings.Contains(ignoredNamespace, "*") {
				ns := strings.Replace(ignoredNamespace, "*", ".*", -1)
				matched, err := regexp.MatchString(ns, givenNs)
				if err == nil && matched {
					return false
				}
			}
		}
	}
	if isAll {
		return true
	}

	return slices.Contains(n.Include, givenNs)
}

// Notification holds notification configuration.
type Notification struct {
	Type NotificationType
}

// Communications channels to send events to
type Communications struct {
	Slack         Slack         `yaml:"slack"`
	Mattermost    Mattermost    `yaml:"mattermost"`
	Discord       Discord       `yaml:"discord"`
	Teams         Teams         `yaml:"teams"`
	Webhook       Webhook       `yaml:"webhook"`
	Elasticsearch Elasticsearch `yaml:"elasticsearch"`
}

// Slack configuration to authentication and send notifications
type Slack struct {
	Enabled      bool                                `yaml:"enabled"`
	Channels     IndexableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required,eq=1"`
	Notification Notification                        `yaml:"notification,omitempty"`
	Token        string                              `yaml:"token,omitempty"`
}

// Elasticsearch config auth settings
type Elasticsearch struct {
	Enabled       bool                   `yaml:"enabled"`
	Username      string                 `yaml:"username"`
	Password      string                 `yaml:"password"`
	Server        string                 `yaml:"server"`
	SkipTLSVerify bool                   `yaml:"skipTLSVerify"`
	AWSSigning    AWSSigning             `yaml:"awsSigning"`
	Indices       IndexableMap[ELSIndex] `yaml:"indices"  validate:"required,eq=1"`
}

// AWSSigning contains AWS configurations
type AWSSigning struct {
	Enabled   bool   `yaml:"enabled"`
	AWSRegion string `yaml:"awsRegion"`
	RoleArn   string `yaml:"roleArn"`
}

// ELSIndex settings for ELS
type ELSIndex struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Shards   int    `yaml:"shards"`
	Replicas int    `yaml:"replicas"`

	Bindings SinkBindings `yaml:"bindings"`
}

// Mattermost configuration to authentication and send notifications
type Mattermost struct {
	Enabled      bool                                `yaml:"enabled"`
	BotName      string                              `yaml:"botName"`
	URL          string                              `yaml:"url"`
	Token        string                              `yaml:"token"`
	Team         string                              `yaml:"team"`
	Channels     IndexableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required,eq=1"`
	Notification Notification                        `yaml:"notification,omitempty"`
}

// Teams creds for authentication with MS Teams
type Teams struct {
	Enabled     bool   `yaml:"enabled"`
	BotName     string `yaml:"botName,omitempty"`
	AppID       string `yaml:"appID,omitempty"`
	AppPassword string `yaml:"appPassword,omitempty"`
	Team        string `yaml:"team"`
	Port        string `yaml:"port"`
	MessagePath string `yaml:"messagePath,omitempty"`
	// TODO: not used yet.
	Channels     IndexableMap[ChannelBindingsByName] `yaml:"channels"`
	Notification Notification                        `yaml:"notification,omitempty"`
}

// Discord configuration for authentication and send notifications
type Discord struct {
	Enabled      bool                              `yaml:"enabled"`
	Token        string                            `yaml:"token"`
	BotID        string                            `yaml:"botID"`
	Channels     IndexableMap[ChannelBindingsByID] `yaml:"channels"  validate:"required,eq=1"`
	Notification Notification                      `yaml:"notification,omitempty"`
}

// Webhook configuration to send notifications
type Webhook struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
	// TODO: not used yet.
	Bindings SinkBindings
}

// Kubectl configuration for executing commands inside cluster
type Kubectl struct {
	Namespaces       Namespaces `yaml:"namespaces,omitempty"`
	Enabled          bool       `yaml:"enabled,omitempty"`
	Commands         Commands   `yaml:"commands,omitempty"`
	DefaultNamespace string     `yaml:"defaultNamespace,omitempty"`
	RestrictAccess   *bool      `yaml:"restrictAccess,omitempty"`
}

// Commands allowed in bot
type Commands struct {
	Verbs     []string `yaml:"verbs"`
	Resources []string `yaml:"resources"`
}

// Settings contains BotKube's related configuration.
type Settings struct {
	ClusterName     string `yaml:"clusterName"`
	ConfigWatcher   bool   `yaml:"configWatcher"`
	UpgradeNotifier bool   `yaml:"upgradeNotifier"`

	MetricsPort string `yaml:"metricsPort"`
	Log         struct {
		Level         string `yaml:"level"`
		DisableColors bool   `yaml:"disableColors"`
	} `yaml:"log"`
	InformersResyncPeriod time.Duration `yaml:"informersResyncPeriod"`
	Kubeconfig            string        `yaml:"kubeconfig"`
}

func (eventType EventType) String() string {
	return string(eventType)
}

// PathsGetter returns the list of absolute paths to the config files.
type PathsGetter func() []string

// LoadWithDefaults loads new configuration from files and environment variables.
func LoadWithDefaults(getCfgPaths PathsGetter) (*Config, []string, error) {
	configPaths := getCfgPaths()
	k := koanf.New(configDelimiter)

	// load default settings
	if err := k.Load(rawbytes.Provider(defaultConfiguration), koanfyaml.Parser()); err != nil {
		return nil, nil, fmt.Errorf("while loading default configuration: %w", err)
	}

	// merge with user conf files
	for _, path := range configPaths {
		if err := k.Load(file.Provider(filepath.Clean(path)), koanfyaml.Parser()); err != nil {
			return nil, nil, err
		}
	}

	// LoadWithDefaults environment variables and merge into the loaded config.
	err := k.Load(env.Provider(
		configEnvVariablePrefix,
		configDelimiter,
		normalizeConfigEnvName,
	), nil)
	if err != nil {
		return nil, nil, err
	}

	var cfg Config
	err = k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return nil, nil, err
	}

	if err := ValidateStruct(cfg); err != nil {
		return nil, nil, fmt.Errorf("while validating loaded configuration: %w", err)
	}

	return &cfg, configPaths, nil
}

// FromEnvOrFlag resolves and returns paths for config files.
// It reads them the 'BOTKUBE_CONFIG_PATHS' env variable. If not found, then it uses '--config' flag.
func FromEnvOrFlag() []string {
	envCfgs := os.Getenv("BOTKUBE_CONFIG_PATHS")
	if envCfgs != "" {
		return strings.Split(envCfgs, ",")
	}

	return configPathsFlag
}

// RegisterFlags registers config related flags.
func RegisterFlags(flags *pflag.FlagSet) {
	flags.StringSliceVarP(&configPathsFlag, "config", "c", nil, "Specify configuration file in YAML format (can specify multiple).")
}

func normalizeConfigEnvName(name string) string {
	name = strings.TrimPrefix(name, configEnvVariablePrefix)

	words := strings.Split(name, camelCaseDelimiter)
	toTitle := cases.Title(language.AmericanEnglish)

	var buff strings.Builder

	buff.WriteString(strings.ToLower(words[0]))
	for _, word := range words[1:] {
		word = strings.ToLower(word)
		buff.WriteString(toTitle.String(word))
	}

	return strings.ReplaceAll(buff.String(), nestedFieldDelimiter, configDelimiter)
}

// IndexableMap provides an option to construct an indexable map.
type IndexableMap[T any] map[string]T

// GetFirst returns the first map element.
// It's not deterministic if map has more than one element.
// TODO(remove): https://github.com/kubeshop/botkube/issues/596
func (t IndexableMap[T]) GetFirst() T {
	var empty T

	for _, v := range t {
		return v
	}

	return empty
}
