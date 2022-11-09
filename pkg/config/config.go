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
)

//go:embed default.yaml
var defaultConfiguration []byte

var configPathsFlag []string

const (
	configEnvVariablePrefix     = "BOTKUBE_"
	configDelimiter             = "."
	camelCaseDelimiter          = "__"
	nestedFieldDelimiter        = "_"
	specialConfigFileNamePrefix = "_"
)

const (
	// AllNamespaceIndicator represents a keyword for allowing all Kubernetes Namespaces.
	AllNamespaceIndicator = ".*"
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
	Actions        Actions                   `yaml:"actions" validate:"dive"`
	Sources        map[string]Sources        `yaml:"sources" validate:"dive"`
	Executors      map[string]Executors      `yaml:"executors" validate:"dive"`
	Communications map[string]Communications `yaml:"communications"  validate:"required,min=1,dive"`
	Filters        Filters                   `yaml:"filters"`

	Analytics     Analytics  `yaml:"analytics"`
	Settings      Settings   `yaml:"settings"`
	ConfigWatcher CfgWatcher `yaml:"configWatcher"`
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
type ActionBindings BotBindings

// Sources contains configuration for Botkube app sources.
type Sources struct {
	DisplayName string           `yaml:"displayName"`
	Kubernetes  KubernetesSource `yaml:"kubernetes"`
}

// KubernetesSource contains configuration for Kubernetes sources.
type KubernetesSource struct {
	Recommendations Recommendations          `yaml:"recommendations"`
	Events          KubernetesResourceEvents `yaml:"events"`
	Resources       []Resource               `yaml:"resources" validate:"dive"`
	Namespaces      Namespaces               `yaml:"namespaces"`
}

// IsAllowed checks if a given resource event is allowed according to the configuration.
func (r *KubernetesSource) IsAllowed(resourceName, namespace string, eventType EventType) bool {
	if r == nil || len(r.Resources) == 0 {
		return false
	}

	isEventAllowed := func(resourceEvents KubernetesResourceEvents) bool {
		if len(resourceEvents) > 0 { // if resource overrides the global events, use them
			return resourceEvents.Contains(eventType)
		}
		return r.Events.Contains(eventType) // check global events
	}

	for _, resource := range r.Resources {
		if resource.Name == resourceName &&
			isEventAllowed(resource.Events) &&
			resource.Namespaces.IsAllowed(namespace) {
			return true
		}
	}

	return false
}

// Recommendations contains configuration for various recommendation insights.
type Recommendations struct {
	Ingress IngressRecommendations `yaml:"ingress"`
	Pod     PodRecommendations     `yaml:"pod"`
}

// PodRecommendations contains configuration for pods recommendations.
type PodRecommendations struct {
	// NoLatestImageTag notifies about Pod containers that use `latest` tag for images.
	NoLatestImageTag *bool `yaml:"noLatestImageTag,omitempty"`

	// LabelsSet notifies about Pod resources created without labels.
	LabelsSet *bool `yaml:"labelsSet,omitempty"`
}

// IngressRecommendations contains configuration for ingress recommendations.
type IngressRecommendations struct {
	// BackendServiceValid notifies about Ingress resources with invalid backend service reference.
	BackendServiceValid *bool `yaml:"backendServiceValid,omitempty"`

	// TLSSecretValid notifies about Ingress resources with invalid TLS secret reference.
	TLSSecretValid *bool `yaml:"tlsSecretValid,omitempty"`
}

// Executors contains executors configuration parameters.
type Executors struct {
	Kubectl Kubectl `yaml:"kubectl"`
}

// Filters contains configuration for built-in filters.
type Filters struct {
	Kubernetes KubernetesFilters `yaml:"kubernetes"`
}

// KubernetesFilters contains configuration for Kubernetes-related filters.
type KubernetesFilters struct {
	// ObjectAnnotationChecker enables support for `botkube.io/disable` and `botkube.io/channel` resource annotations.
	ObjectAnnotationChecker bool `yaml:"objectAnnotationChecker"`

	// NodeEventsChecker filters out Node-related events that are not important.
	NodeEventsChecker bool `yaml:"nodeEventsChecker"`
}

// SetEnabled enables or disables a given filter.
func (f *KubernetesFilters) SetEnabled(name string, enabled bool) error {
	if name == "ObjectAnnotationChecker" {
		f.ObjectAnnotationChecker = enabled
		return nil
	}

	if name == "NodeEventsChecker" {
		f.NodeEventsChecker = enabled
		return nil
	}

	return fmt.Errorf("Filter with name %q not found", name)
}

// Analytics contains configuration parameters for analytics collection.
type Analytics struct {
	Disable bool `yaml:"disable"`
}

// Resource contains resources to watch
type Resource struct {
	Name          string                   `yaml:"name"`
	Namespaces    Namespaces               `yaml:"namespaces"`
	Events        KubernetesResourceEvents `yaml:"events"`
	UpdateSetting UpdateSetting            `yaml:"updateSetting"`
}

// KubernetesResourceEvents contains events to watch for a resource.
type KubernetesResourceEvents []EventType

// Contains checks if event is contained in the events slice.
// If the slice contains AllEvent, then the result is true.
func (e *KubernetesResourceEvents) Contains(eventType EventType) bool {
	if e == nil {
		return false
	}

	for _, event := range *e {
		if event == AllEvent {
			return true
		}

		if event == eventType {
			return true
		}
	}

	return false
}

// UpdateSetting struct defines updateEvent fields specification
type UpdateSetting struct {
	Fields      []string `yaml:"fields"`
	IncludeDiff bool     `yaml:"includeDiff"`
}

// Namespaces provides an option to include and exclude given Namespaces.
type Namespaces struct {
	// Include contains a list of allowed Namespaces.
	// It can also contain a regex expressions:
	//  - ".*" - to specify all Namespaces.
	Include []string `yaml:"include"`

	// Exclude contains a list of Namespaces to be ignored even if allowed by Include.
	// It can also contain a regex expressions:
	//  - "test-.*" - to specif all Namespaces with `test-` prefix.
	Exclude []string `yaml:"exclude,omitempty"`
}

// IsConfigured checks whether the Namespace has any Include/Exclude configuration.
func (n *Namespaces) IsConfigured() bool {
	return len(n.Include) > 0 || len(n.Exclude) > 0
}

// IsAllowed checks if a given Namespace is allowed based on the config.
func (n *Namespaces) IsAllowed(givenNs string) bool {
	if n == nil || givenNs == "" {
		return false
	}

	// 1. Check if excluded
	if len(n.Exclude) > 0 {
		for _, excludeNamespace := range n.Exclude {
			if strings.TrimSpace(excludeNamespace) == "" {
				continue
			}
			// exact match
			if excludeNamespace == givenNs {
				return false
			}

			// regexp
			matched, err := regexp.MatchString(excludeNamespace, givenNs)
			if err == nil && matched {
				return false
			}
		}
	}

	// 2. Check if included, if matched, return true
	if len(n.Include) > 0 {
		for _, includeNamespace := range n.Include {
			if strings.TrimSpace(includeNamespace) == "" {
				continue
			}

			// exact match
			if includeNamespace == givenNs {
				return true
			}

			// regexp
			matched, err := regexp.MatchString(includeNamespace, givenNs)
			if err == nil && matched {
				return true
			}
		}
	}

	// 2.1. If not included, return false
	return false
}

// Notification holds notification configuration.
type Notification struct {
	Type NotificationType
}

// ChannelNotification contains notification configuration for a given platform.
type ChannelNotification struct {
	Disabled bool `yaml:"disabled"`
}

// Communications contains communication platforms that are supported.
type Communications struct {
	Slack         Slack         `yaml:"slack"`
	SocketSlack   SocketSlack   `yaml:"socketSlack"`
	Mattermost    Mattermost    `yaml:"mattermost"`
	Discord       Discord       `yaml:"discord"`
	Teams         Teams         `yaml:"teams"`
	Webhook       Webhook       `yaml:"webhook"`
	Elasticsearch Elasticsearch `yaml:"elasticsearch"`
}

// Slack configuration to authentication and send notifications
type Slack struct {
	Enabled      bool                                   `yaml:"enabled"`
	Channels     IdentifiableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
	Notification Notification                           `yaml:"notification,omitempty"`
	Token        string                                 `yaml:"token,omitempty"`
}

// SocketSlack configuration to authentication and send notifications
type SocketSlack struct {
	Enabled      bool                                   `yaml:"enabled"`
	Channels     IdentifiableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
	Notification Notification                           `yaml:"notification,omitempty"`
	BotToken     string                                 `yaml:"botToken,omitempty"`
	AppToken     string                                 `yaml:"appToken,omitempty"`
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
	Enabled      bool                                   `yaml:"enabled"`
	BotName      string                                 `yaml:"botName"`
	URL          string                                 `yaml:"url"`
	Token        string                                 `yaml:"token"`
	Team         string                                 `yaml:"team"`
	Channels     IdentifiableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
	Notification Notification                           `yaml:"notification,omitempty"`
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
	Bindings     BotBindings  `yaml:"bindings" validate:"required_if=Enabled true"`
	Notification Notification `yaml:"notification,omitempty"`
}

// Discord configuration for authentication and send notifications
type Discord struct {
	Enabled      bool                                 `yaml:"enabled"`
	Token        string                               `yaml:"token"`
	BotID        string                               `yaml:"botID"`
	Channels     IdentifiableMap[ChannelBindingsByID] `yaml:"channels"  validate:"required_if=Enabled true,dive,omitempty,min=1"`
	Notification Notification                         `yaml:"notification,omitempty"`
}

// Webhook configuration to send notifications
type Webhook struct {
	Enabled  bool         `yaml:"enabled"`
	URL      string       `yaml:"url"`
	Bindings SinkBindings `yaml:"bindings" validate:"required_if=Enabled true"`
}

// Kubectl configuration for executing commands inside cluster
type Kubectl struct {
	Namespaces       Namespaces `yaml:"namespaces,omitempty"`
	Enabled          bool       `yaml:"enabled"`
	Commands         Commands   `yaml:"commands,omitempty"`
	DefaultNamespace string     `yaml:"defaultNamespace,omitempty"`
	RestrictAccess   *bool      `yaml:"restrictAccess,omitempty"`
}

// Commands allowed in bot
type Commands struct {
	Verbs     []string `yaml:"verbs"`
	Resources []string `yaml:"resources"`
}

// CfgWatcher describes configuration for watching the configuration.
type CfgWatcher struct {
	Enabled            bool          `yaml:"enabled"`
	InitialSyncTimeout time.Duration `yaml:"initialSyncTimeout"`
	TmpDir             string        `yaml:"tmpDir"`
}

// Settings contains Botkube's related configuration.
type Settings struct {
	ClusterName      string           `yaml:"clusterName"`
	UpgradeNotifier  bool             `yaml:"upgradeNotifier"`
	SystemConfigMap  K8sResourceRef   `yaml:"systemConfigMap"`
	PersistentConfig PersistentConfig `yaml:"persistentConfig"`
	MetricsPort      string           `yaml:"metricsPort"`
	LifecycleServer  LifecycleServer  `yaml:"lifecycleServer"`
	Log              struct {
		Level         string `yaml:"level"`
		DisableColors bool   `yaml:"disableColors"`
	} `yaml:"log"`
	InformersResyncPeriod time.Duration `yaml:"informersResyncPeriod"`
	Kubeconfig            string        `yaml:"kubeconfig"`
}

// LifecycleServer contains configuration for the server with app lifecycle methods.
type LifecycleServer struct {
	Enabled    bool           `yaml:"enabled"`
	Port       int            `yaml:"port"` // String for consistency
	Deployment K8sResourceRef `yaml:"deployment"`
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

// PathsGetter returns the list of absolute paths to the config files.
type PathsGetter func() []string

// LoadWithDefaultsDetails holds the LoadWithDefaults function details.
type LoadWithDefaultsDetails struct {
	ValidateWarnings error
}

// LoadWithDefaults loads new configuration from files and environment variables.
func LoadWithDefaults(getCfgPaths PathsGetter) (*Config, LoadWithDefaultsDetails, error) {
	configPaths := getCfgPaths()
	k := koanf.New(configDelimiter)

	// load default settings
	if err := k.Load(rawbytes.Provider(defaultConfiguration), koanfyaml.Parser()); err != nil {
		return nil, LoadWithDefaultsDetails{}, fmt.Errorf("while loading default configuration: %w", err)
	}

	// merge with user conf files
	configPaths = sortCfgFiles(configPaths)
	for _, path := range configPaths {
		if err := k.Load(file.Provider(filepath.Clean(path)), koanfyaml.Parser()); err != nil {
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
	err = k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "yaml"})
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

// sortCfgFiles sorts the config files so that the files that has specialConfigFileNamePrefix are moved to the end of the slice.
func sortCfgFiles(paths []string) []string {
	var ordinaryCfgFiles []string
	var specialCfgFiles []string
	for _, path := range paths {
		_, filename := filepath.Split(path)

		if strings.HasPrefix(filename, specialConfigFileNamePrefix) {
			specialCfgFiles = append(specialCfgFiles, path)
			continue
		}

		ordinaryCfgFiles = append(ordinaryCfgFiles, path)
	}

	return append(ordinaryCfgFiles, specialCfgFiles...)
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
