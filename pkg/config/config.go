package config

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/knadh/koanf"
	koanfyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/rawbytes"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed default.yaml
var defaultConfiguration []byte

const (
	configEnvVariablePrefix = "BOTKUBE_"
	configDelimiter         = "."
	camelCaseDelimiter      = "__"
	nestedFieldDelimiter    = "_"
)

const (
	// allValuesPattern represents a keyword for allowing all values.
	allValuesPattern = ".*"
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

	// SocketSlackCommPlatformIntegration defines Slack integration.
	SocketSlackCommPlatformIntegration CommPlatformIntegration = "socketSlack"

	// CloudSlackCommPlatformIntegration defines Slack integration.
	CloudSlackCommPlatformIntegration CommPlatformIntegration = "cloudSlack"

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

func (c CommPlatformIntegration) IsInteractive() bool {
	return c == SocketSlackCommPlatformIntegration
}

// String returns string platform name.
func (c CommPlatformIntegration) String() string {
	return string(c)
}

// IntegrationType describes the type of integration with a communication platform.
type IntegrationType string

const (
	// BotIntegrationType describes two-way integration.
	BotIntegrationType IntegrationType = "bot"

	// SinkIntegrationType describes one-way integration.
	SinkIntegrationType IntegrationType = "sink"
)

// Config structure of configuration yaml file
type Config struct {
	Actions        Actions                   `yaml:"actions" validate:"dive"`
	Sources        map[string]Sources        `yaml:"sources" validate:"dive"`
	Executors      map[string]Executors      `yaml:"executors" validate:"dive"`
	Aliases        Aliases                   `yaml:"aliases" validate:"dive"`
	Communications map[string]Communications `yaml:"communications"  validate:"required,min=1,dive"`

	Analytics     Analytics        `yaml:"analytics"`
	Settings      Settings         `yaml:"settings"`
	ConfigWatcher CfgWatcher       `yaml:"configWatcher"`
	Plugins       PluginManagement `yaml:"plugins"`
}

// PluginManagement holds Botkube plugin management related configuration.
type PluginManagement struct {
	CacheDir     string                         `yaml:"cacheDir"`
	Repositories map[string]PluginsRepositories `yaml:"repositories"`
}

// PluginsRepositories holds the Plugin repository information.
type PluginsRepositories struct {
	URL string `yaml:"url"`
}

// ChannelBindingsByName contains configuration bindings per channel.
type ChannelBindingsByName struct {
	Name         string              `yaml:"name"`
	Notification ChannelNotification `yaml:"notification"` // TODO: rename to `notifications` later
	Bindings     BotBindings         `yaml:"bindings"`
}

// Identifier returns ChannelBindingsByID identifier.
func (c ChannelBindingsByName) Identifier() string {
	return c.Name
}

// ChannelBindingsByID contains configuration bindings per channel.
type ChannelBindingsByID struct {
	ID           string              `yaml:"id"`
	Notification ChannelNotification `yaml:"notification"` // TODO: rename to `notifications` later
	Bindings     BotBindings         `yaml:"bindings"`
}

// Identifier returns ChannelBindingsByID identifier.
func (c ChannelBindingsByID) Identifier() string {
	return c.ID
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

// Actions contains configuration for Botkube app event automations.
type Actions map[string]Action

// Action contains configuration for Botkube app event automations.
type Action struct {
	Enabled     bool           `yaml:"enabled"`
	DisplayName string         `yaml:"displayName"`
	Command     string         `yaml:"command" validate:"required_if=Enabled true"`
	Bindings    ActionBindings `yaml:"bindings"`
}

// ActionBindings contains configuration for action bindings.
type ActionBindings struct {
	Sources   []string `yaml:"sources"`
	Executors []string `yaml:"executors"`
}

// Sources contains configuration for Botkube app sources.
type Sources struct {
	DisplayName string  `yaml:"displayName"`
	Plugins     Plugins `yaml:",inline" koanf:",remain"`
}

// GetPlugins returns Sources.Plugins.
func (s Sources) GetPlugins() Plugins {
	return s.Plugins
}

// Plugins contains plugins configuration parameters defined in groups.
type Plugins map[string]Plugin

// Plugin contains plugin specific configuration.
type Plugin struct {
	Enabled bool
	Config  any
	Context PluginContext
}

// PluginContext defines the context for given plugin.
type PluginContext struct {
	// RBAC defines the RBAC rules for given plugin.
	RBAC *PolicyRule `yaml:"rbac,omitempty"`
}

// PolicyRule is the RBAC rule.
type PolicyRule struct {
	// User is the policy subject for user.
	User UserPolicySubject `yaml:"user"`
	// Group is the policy subject for group.
	Group GroupPolicySubject `yaml:"group"`
}

// GroupPolicySubject is the RBAC subject.
type GroupPolicySubject struct {
	// Type is the type of policy subject.
	Type PolicySubjectType `yaml:"type"`
	// Static is static reference of subject for given static policy rule.
	Static GroupStaticSubject `yaml:"static"`
	// Prefix is optional string prefixed to subjects.
	Prefix string `yaml:"prefix"`
}

// GroupStaticSubject references static subjects for given static policy rule.
type GroupStaticSubject struct {
	// Values is the name of the subject.
	Values []string `yaml:"values"`
}

// UserPolicySubject is the RBAC subject.
type UserPolicySubject struct {
	// Type is the type of policy subject.
	Type PolicySubjectType `yaml:"type"`
	// Static is static reference of subject for given static policy rule.
	Static UserStaticSubject `yaml:"static"`
	// Prefix is optional string prefixed to subjects.
	Prefix string `yaml:"prefix"`
}

// UserStaticSubject references static subjects for given static policy rule.
type UserStaticSubject struct {
	// Value is the name of the subject.
	Value string `yaml:"value"`
}

// PolicySubjectType defines the types for policy subjects.
type PolicySubjectType string

const (
	// EmptyPolicySubjectType is the empty policy type.
	EmptyPolicySubjectType PolicySubjectType = ""
	// StaticPolicySubjectType is the static policy type.
	StaticPolicySubjectType PolicySubjectType = "Static"
	// ChannelNamePolicySubjectType is the channel name policy type.
	ChannelNamePolicySubjectType PolicySubjectType = "ChannelName"
)

// Executors contains executors configuration parameters.
type Executors struct {
	Plugins Plugins `yaml:",inline" koanf:",remain"`
}

// CollectCommandPrefixes returns list of command prefixes for all executors, even disabled ones.
func (e Executors) CollectCommandPrefixes() []string {
	var prefixes []string
	for pluginName := range e.Plugins {
		prefixes = append(prefixes, ExecutorNameForKey(pluginName))
	}
	return prefixes
}

// GetPlugins returns Executors.Plugins
func (e Executors) GetPlugins() Plugins {
	return e.Plugins
}

// Aliases contains aliases configuration.
type Aliases map[string]Alias

// Alias defines alias configuration for a given command.
type Alias struct {
	Command     string `yaml:"command" validate:"required"`
	DisplayName string `yaml:"displayName"`
}

// Analytics contains configuration parameters for analytics collection.
type Analytics struct {
	Disable bool `yaml:"disable"`
}

// RegexConstraints contains a list of allowed and excluded values.
type RegexConstraints struct {
	// Include contains a list of allowed values.
	// It can also contain a regex expressions:
	//  - ".*" - to specify all values.
	Include []string `yaml:"include"`

	// Exclude contains a list of values to be ignored even if allowed by Include.
	// It can also contain a regex expressions:
	//  - "test-.*" - to specify all values with `test-` prefix.
	Exclude []string `yaml:"exclude,omitempty"`
}

// AreConstraintsDefined checks whether the RegexConstraints has any Include/Exclude configuration.
func (r *RegexConstraints) AreConstraintsDefined() bool {
	return len(r.Include) > 0 || len(r.Exclude) > 0
}

// IsAllowed checks if a given value is allowed based on the config.
// Firstly, it checks if the value is excluded. If not, then it checks if the value is included.
func (r *RegexConstraints) IsAllowed(value string) (bool, error) {
	if r == nil {
		return false, nil
	}

	// 1. Check if excluded
	if len(r.Exclude) > 0 {
		for _, excludeValue := range r.Exclude {
			if strings.TrimSpace(excludeValue) == "" {
				continue
			}
			// exact match
			if excludeValue == value {
				return false, nil
			}

			// regexp
			matched, err := regexp.MatchString(excludeValue, value)
			if err != nil {
				return false, fmt.Errorf("while matching %q with exclude regex %q: %v", value, excludeValue, err)
			}
			if matched {
				return false, nil
			}
		}
	}

	// 2. Check if included, if matched, return true
	if len(r.Include) > 0 {
		for _, includeValue := range r.Include {
			// exact match
			if includeValue == value {
				return true, nil
			}

			// regexp
			matched, err := regexp.MatchString(includeValue, value)
			if err != nil {
				return false, fmt.Errorf("while matching %q with include regex %q: %v", value, includeValue, err)
			}
			if matched {
				return true, nil
			}
		}
	}

	// 2.1. If not included, return false
	return false, nil
}

// ChannelNotification contains notification configuration for a given platform.
type ChannelNotification struct {
	Disabled bool `yaml:"disabled"`
}

// Communications contains communication platforms that are supported.
type Communications struct {
	Slack         Slack         `yaml:"slack,omitempty"`
	SocketSlack   SocketSlack   `yaml:"socketSlack,omitempty"`
	CloudSlack    CloudSlack    `yaml:"cloudSlack,omitempty"`
	Mattermost    Mattermost    `yaml:"mattermost,omitempty"`
	Discord       Discord       `yaml:"discord,omitempty"`
	Teams         Teams         `yaml:"teams,omitempty"`
	Webhook       Webhook       `yaml:"webhook,omitempty"`
	Elasticsearch Elasticsearch `yaml:"elasticsearch,omitempty"`
}

// Slack holds Slack integration config.
// Deprecated: Legacy Slack integration has been deprecated and removed from the Slack App Directory.
// Use SocketSlack integration instead.
type Slack struct {
	Enabled  bool                                   `yaml:"enabled"`
	Channels IdentifiableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
	Token    string                                 `yaml:"token,omitempty"`
}

// SocketSlack configuration to authentication and send notifications
type SocketSlack struct {
	Enabled  bool                                   `yaml:"enabled"`
	Channels IdentifiableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
	BotToken string                                 `yaml:"botToken,omitempty"`
	AppToken string                                 `yaml:"appToken,omitempty"`
}

type CloudSlack struct {
	Enabled  bool                                   `yaml:"enabled"`
	Channels IdentifiableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
	BotToken string                                 `yaml:"botToken,omitempty"`
	AppToken string                                 `yaml:"appToken,omitempty"`
	Server   string                                 `yaml:"server"`
}

// Elasticsearch config auth settings
type Elasticsearch struct {
	Enabled       bool                `yaml:"enabled"`
	Username      string              `yaml:"username"`
	Password      string              `yaml:"password"`
	Server        string              `yaml:"server"`
	SkipTLSVerify bool                `yaml:"skipTLSVerify"`
	AWSSigning    AWSSigning          `yaml:"awsSigning"`
	Indices       map[string]ELSIndex `yaml:"indices"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
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
	Enabled  bool                                   `yaml:"enabled"`
	BotName  string                                 `yaml:"botName"`
	URL      string                                 `yaml:"url"`
	Token    string                                 `yaml:"token"`
	Team     string                                 `yaml:"team"`
	Channels IdentifiableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
}

// Teams creds for authentication with MS Teams
type Teams struct {
	Enabled     bool   `yaml:"enabled"`
	BotName     string `yaml:"botName,omitempty"`
	AppID       string `yaml:"appID,omitempty"`
	AppPassword string `yaml:"appPassword,omitempty"`
	Port        string `yaml:"port"`
	MessagePath string `yaml:"messagePath,omitempty"`
	// TODO: Be consistent with other communicators when MS Teams support multiple channels
	//Channels     IdentifiableMap[ChannelBindingsByName] `yaml:"channels"`
	Bindings BotBindings `yaml:"bindings" validate:"required_if=Enabled true"`
}

// Discord configuration for authentication and send notifications
type Discord struct {
	Enabled  bool                                 `yaml:"enabled"`
	Token    string                               `yaml:"token"`
	BotID    string                               `yaml:"botID"`
	Channels IdentifiableMap[ChannelBindingsByID] `yaml:"channels"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
}

// Webhook configuration to send notifications
type Webhook struct {
	Enabled  bool         `yaml:"enabled"`
	URL      string       `yaml:"url"`
	Bindings SinkBindings `yaml:"bindings" validate:"required_if=Enabled true"`
}

// CfgWatcher describes configuration for watching the configuration.
type CfgWatcher struct {
	Enabled bool             `yaml:"enabled"`
	Remote  RemoteCfgWatcher `yaml:"remote"`

	InitialSyncTimeout time.Duration  `yaml:"initialSyncTimeout"`
	TmpDir             string         `yaml:"tmpDir"`
	Deployment         K8sResourceRef `yaml:"deployment"`
}

// RemoteCfgWatcher describes configuration for watching the configuration using remote config provider.
type RemoteCfgWatcher struct {
	PollInterval time.Duration `yaml:"pollInterval"`
}

// Settings contains Botkube's related configuration.
type Settings struct {
	ClusterName           string           `yaml:"clusterName"`
	UpgradeNotifier       bool             `yaml:"upgradeNotifier"`
	SystemConfigMap       K8sResourceRef   `yaml:"systemConfigMap"`
	PersistentConfig      PersistentConfig `yaml:"persistentConfig"`
	MetricsPort           string           `yaml:"metricsPort"`
	HealthPort            string           `yaml:"healthPort"`
	LifecycleServer       LifecycleServer  `yaml:"lifecycleServer"`
	Log                   Logger           `yaml:"log"`
	InformersResyncPeriod time.Duration    `yaml:"informersResyncPeriod"`
	Kubeconfig            string           `yaml:"kubeconfig"`
}

// Logger holds logger configuration parameters.
type Logger struct {
	Level         string `yaml:"level"`
	DisableColors bool   `yaml:"disableColors"`
}

// LifecycleServer contains configuration for the server with app lifecycle methods.
type LifecycleServer struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"` // String for consistency
}

// PersistentConfig contains configuration for persistent storage.
type PersistentConfig struct {
	Startup PartialPersistentConfig `yaml:"startup"`
	Runtime PartialPersistentConfig `yaml:"runtime"`
}

// PartialPersistentConfig contains configuration for persistent storage of a given type.
type PartialPersistentConfig struct {
	FileName  string         `yaml:"fileName"`
	ConfigMap K8sResourceRef `yaml:"configMap"`
}

// K8sResourceRef holds the configuration for a Kubernetes resource.
type K8sResourceRef struct {
	Name      string `yaml:"name,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
}

func (eventType EventType) String() string {
	return string(eventType)
}

// LoadWithDefaultsDetails holds the LoadWithDefaults function details.
type LoadWithDefaultsDetails struct {
	ValidateWarnings error
}

// LoadWithDefaults loads new configuration from files and environment variables.
func LoadWithDefaults(configs [][]byte) (*Config, LoadWithDefaultsDetails, error) {
	k := koanf.New(configDelimiter)

	// load default settings
	if err := k.Load(rawbytes.Provider(defaultConfiguration), koanfyaml.Parser()); err != nil {
		return nil, LoadWithDefaultsDetails{}, fmt.Errorf("while loading default configuration: %w", err)
	}

	// merge with user configs
	for _, rawCfg := range configs {
		if err := k.Load(rawbytes.Provider(rawCfg), koanfyaml.Parser()); err != nil {
			return nil, LoadWithDefaultsDetails{}, err
		}
	}

	// LoadWithDefaults environment variables and merge into the loaded config.
	err := k.Load(env.Provider(
		configEnvVariablePrefix,
		configDelimiter,
		normalizeConfigEnvName,
	), nil)
	if err != nil {
		return nil, LoadWithDefaultsDetails{}, err
	}

	var cfg Config
	err = k.Unmarshal("", &cfg)
	if err != nil {
		return nil, LoadWithDefaultsDetails{}, err
	}

	result, err := ValidateStruct(cfg)
	if err != nil {
		return nil, LoadWithDefaultsDetails{}, fmt.Errorf("while validating loaded configuration: %w", err)
	}
	if err := result.Criticals.ErrorOrNil(); err != nil {
		return nil, LoadWithDefaultsDetails{}, fmt.Errorf("found critical validation errors: %w", err)
	}

	return &cfg, LoadWithDefaultsDetails{
		ValidateWarnings: result.Warnings.ErrorOrNil(),
	}, nil
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

// IdentifiableMap provides an option to construct an indexable map for identifiable items.
type IdentifiableMap[T Identifiable] map[string]T

// Identifiable exports an Identifier method.
type Identifiable interface {
	Identifier() string
}

// GetByIdentifier gets an item from a map by identifier.
func (t IdentifiableMap[T]) GetByIdentifier(val string) (T, bool) {
	for _, v := range t {
		if v.Identifier() != val {
			continue
		}
		return v, true
	}

	var empty T
	return empty, false
}
