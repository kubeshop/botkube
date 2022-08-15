package config

import (
	"context"
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
	"k8s.io/client-go/tools/cache"
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
	Sources        IndexableMap[Sources]     `yaml:"sources" validate:"dive"`
	Executors      map[string]Executors      `yaml:"executors" validate:"dive"`
	Communications map[string]Communications `yaml:"communications"  validate:"required,min=1"`

	Analytics Analytics `yaml:"analytics"`
	Settings  Settings  `yaml:"settings"`
}

type mergedEvents map[string]map[EventType]struct{}

type SourceRoutes struct {
	source     string
	namespaces []string
}

type RoutedEvent struct {
	event  EventType
	routes []SourceRoutes
}

type Informer struct {
	informer     cache.SharedIndexInformer
	events       []EventType
	srcResources []string
	srcEvents    []EventType

	//cache.ResourceEventHandlerFuncs
}

func (i Informer) handle(target EventType, handlerFn func(obj interface{}, oldObj interface{})) {
	switch {
	case target == CreateEvent:
		//i.AddFunc = func(obj interface{}) { handlerFn(obj, nil) }
		i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) { handlerFn(obj, nil) },
		})
	case target == DeleteEvent:
		//i.DeleteFunc = func(obj interface{}) { handlerFn(obj, nil) }
		i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) { handlerFn(obj, nil) },
		})
	case target == UpdateEvent:
		//i.UpdateFunc = handlerFn
		i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			UpdateFunc: handlerFn,
		})

	case target == ErrorEvent:
	}
}

func (i Informer) canHandle(target EventType) bool {
	for _, e := range i.events {
		if target == e {
			return true
		}
	}
	return false
}

type SourcesRouter struct {
	table     map[string][]RoutedEvent
	bindings  map[string]struct{}
	informers map[string]Informer
}

func (r *SourcesRouter) AddAnySlackBindings(c IdentifiableMap[ChannelBindingsByName]) {
	for _, name := range c {
		for _, source := range name.Bindings.Sources {
			r.bindings[source] = struct{}{}
		}
	}
}

func (r *SourcesRouter) BuildTable(cfg *Config) *SourcesRouter {
	sources := r.GetBoundSources(cfg.Sources)
	mergedEvents := mergeResourceEvents(sources)

	for resource, resourceEvents := range mergedEvents {
		eventRoutes := mergeEventRoutes(resource, sources)
		fmt.Printf("\n\nCOLLECTED_ROUTES: %+v\n\n", eventRoutes)
		r.buildTable(resource, resourceEvents, eventRoutes)
	}

	return r
}

func mergeResourceEvents(sources IndexableMap[Sources]) mergedEvents {
	out := map[string]map[EventType]struct{}{}
	for _, srcGroupCfg := range sources {
		for _, resource := range srcGroupCfg.Kubernetes.Resources {
			if _, ok := out[resource.Name]; !ok {
				out[resource.Name] = make(map[EventType]struct{})
			}
			for _, e := range flattenEvents(resource.Events) {
				out[resource.Name][e] = struct{}{}
			}
		}
	}
	return out
}

func mergeEventRoutes(resource string, sources IndexableMap[Sources]) map[EventType][]SourceRoutes {
	out := make(map[EventType][]SourceRoutes)
	for srcGroupName, srcGroupCfg := range sources {
		for _, r := range srcGroupCfg.Kubernetes.Resources {
			for _, e := range flattenEvents(r.Events) {
				if resource == r.Name {
					out[e] = append(out[e], SourceRoutes{source: srcGroupName, namespaces: r.Namespaces.Include})
				}
			}
		}
	}
	return out
}

func (r *SourcesRouter) buildTable(resource string, events map[EventType]struct{}, pairings map[EventType][]SourceRoutes) {
	for evt := range events {
		if _, ok := r.table[resource]; !ok {

			r.table[resource] = []RoutedEvent{{
				event:  evt,
				routes: pairings[evt],
			}}

		} else {
			r.table[resource] = append(r.table[resource], RoutedEvent{event: evt, routes: pairings[evt]})
		}
	}
}

func (r *SourcesRouter) GetBoundSources(sources IndexableMap[Sources]) IndexableMap[Sources] {
	out := make(IndexableMap[Sources])
	for name, t := range sources {
		if _, ok := r.bindings[name]; ok {
			out[name] = t
		}
	}
	return out
}

type registrationHandler func(resource string) cache.SharedIndexInformer
type eventHandler func(ctx context.Context, resource string, routes []SourceRoutes) func(obj, oldObj interface{})

func (r *SourcesRouter) RegisterRoutedInformers(handler registrationHandler) {
	for resource := range r.table {
		targetEvents := r.resourceEvents(resource)
		informer := handler(resource)
		r.informers[resource] = Informer{
			informer: informer,
			events:   targetEvents,
			//ResourceEventHandlerFuncs: cache.ResourceEventHandlerFuncs{},
		}
	}
}

func (r *SourcesRouter) RegisterMappedInformers(src EventType, dstResource string, dstEvents []EventType, handler registrationHandler) {
	srcResources := r.resourcesForEvent(src)
	if len(srcResources) == 0 {
		return
	}

	informer := handler(dstResource)
	r.informers[dstResource] = Informer{
		informer:     informer,
		events:       dstEvents,
		srcResources: srcResources,
		srcEvents:    []EventType{src},
		//ResourceEventHandlerFuncs: cache.ResourceEventHandlerFuncs{},
	}

}

//func (r *SourcesRouter) RegisterInformers(targetEvents []EventType, handler registrationHandler) {
//	resources := r.resourcesForEvents(targetEvents)
//	for resource := range resources {
//		informer := handler(resource)
//		r.informers[resource] = Informer{
//			informer: informer,
//			events:   targetEvents,
//			//ResourceEventHandlerFuncs: cache.ResourceEventHandlerFuncs{},
//		}
//	}
//}

func (r *SourcesRouter) HandleEvent(ctx context.Context, target EventType, handlerFn eventHandler) {
	for resource, informer := range r.informers {
		if informer.canHandle(target) {
			routes := r.sourceRoutes(resource, target)
			informer.handle(target, handlerFn(ctx, resource, routes))
		}
	}
}

func (r *SourcesRouter) sourceRoutes(resource string, target EventType) []SourceRoutes {
	var out []SourceRoutes
	for _, routedEvent := range r.table[resource] {
		if routedEvent.event == target {
			out = append(out, routedEvent.routes...)
		}
	}
	return out
}

func (r *SourcesRouter) resourceEvents(resource string) []EventType {
	var out []EventType
	for _, routedEvent := range r.table[resource] {
		out = append(out, routedEvent.event)
	}
	return out
}

func (r *SourcesRouter) resourcesForEvent(target EventType) []string {
	var out []string
	for resource, routedEvents := range r.table {
		for _, routedEvent := range routedEvents {
			if routedEvent.event == target {
				out = append(out, resource)
			}
		}
	}
	return out
}

func flattenEvents(events []EventType) []EventType {
	var out []EventType
	for _, event := range events {
		if event == AllEvent {
			out = append(out, []EventType{CreateEvent, UpdateEvent, DeleteEvent, ErrorEvent}...)
		} else {
			out = append(out, event)
		}
	}
	return out
}

func NewSourcesRouter() *SourcesRouter {
	return &SourcesRouter{
		table: make(map[string][]RoutedEvent),
		//informers: make(map[string]cache.SharedIndexInformer),
	}
}

// ChannelBindingsByName contains configuration bindings per channel.
type ChannelBindingsByName struct {
	Name     string      `yaml:"name"`
	Bindings BotBindings `yaml:"bindings"`
}

// Identifier returns ChannelBindingsByID identifier.
func (c ChannelBindingsByName) Identifier() string {
	return c.Name
}

// ChannelBindingsByID contains configuration bindings per channel.
type ChannelBindingsByID struct {
	ID       string      `yaml:"id"`
	Bindings BotBindings `yaml:"bindings"`
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

// Sources contains configuration for BotKube app sources.
type Sources struct {
	Kubernetes KubernetesSource `yaml:"kubernetes"`
}

// KubernetesSource contains configuration for Kubernetes sources.
type KubernetesSource struct {
	Recommendations Recommendations `yaml:"recommendations"`
	Resources       []Resource      `yaml:"resources" validate:"dive"`
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

// IsAllowed checks if a given Namespace is allowed based on the config.
func (n *Namespaces) IsAllowed(givenNs string) bool {
	if n == nil {
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
	Enabled      bool                                   `yaml:"enabled"`
	Channels     IdentifiableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required,eq=1"`
	Notification Notification                           `yaml:"notification,omitempty"`
	Token        string                                 `yaml:"token,omitempty"`
}

// Elasticsearch config auth settings
type Elasticsearch struct {
	Enabled       bool                `yaml:"enabled"`
	Username      string              `yaml:"username"`
	Password      string              `yaml:"password"`
	Server        string              `yaml:"server"`
	SkipTLSVerify bool                `yaml:"skipTLSVerify"`
	AWSSigning    AWSSigning          `yaml:"awsSigning"`
	Indices       map[string]ELSIndex `yaml:"indices"  validate:"required,eq=1"`
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
	Channels     IdentifiableMap[ChannelBindingsByName] `yaml:"channels"  validate:"required,eq=1"`
	Notification Notification                           `yaml:"notification,omitempty"`
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
	// TODO: Be consistent with other communicators when MS Teams support multiple channels
	//Channels     IdentifiableMap[ChannelBindingsByName] `yaml:"channels"`
	Bindings     BotBindings  `yaml:"bindings"`
	Notification Notification `yaml:"notification,omitempty"`
}

// Discord configuration for authentication and send notifications
type Discord struct {
	Enabled      bool                                 `yaml:"enabled"`
	Token        string                               `yaml:"token"`
	BotID        string                               `yaml:"botID"`
	Channels     IdentifiableMap[ChannelBindingsByID] `yaml:"channels"  validate:"required,eq=1"`
	Notification Notification                         `yaml:"notification,omitempty"`
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

// LoadWithDefaultsDetails holds the LoadWithDefaults function details.
type LoadWithDefaultsDetails struct {
	LoadedCfgFilesPaths []string
	ValidateWarnings    error
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
		LoadedCfgFilesPaths: configPaths,
		ValidateWarnings:    result.Warnings.ErrorOrNil(),
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
