package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/source"
)

// Config Kubernetes configuration
type Config struct {
	KubeConfig           string `yaml:"kubeConfig"`
	ClusterName          string
	InformerReSyncPeriod time.Duration     `yaml:"informerReSyncPeriod"`
	Log                  *Log              `yaml:"log"`
	Recommendations      Recommendations   `yaml:"recommendations"`
	Event                KubernetesEvent   `yaml:"event"`
	Resources            []Resource        `yaml:"resources" validate:"dive"`
	Namespaces           Namespaces        `yaml:"namespaces"`
	Annotations          map[string]string `yaml:"annotations"`
	Labels               map[string]string `yaml:"labels"`
}

// Recommendations contains configuration for various recommendation insights.
type Recommendations struct {
	Ingress IngressRecommendations `yaml:"ingress"`
	Pod     PodRecommendations     `yaml:"pod"`
}

// IngressRecommendations contains configuration for ingress recommendations.
type IngressRecommendations struct {
	// BackendServiceValid notifies about Ingress resources with invalid backend service reference.
	BackendServiceValid *bool `yaml:"backendServiceValid,omitempty"`

	// TLSSecretValid notifies about Ingress resources with invalid TLS secret reference.
	TLSSecretValid *bool `yaml:"tlsSecretValid,omitempty"`
}

// PodRecommendations contains configuration for pods recommendations.
type PodRecommendations struct {
	// NoLatestImageTag notifies about Pod containers that use `latest` tag for images.
	NoLatestImageTag *bool `yaml:"noLatestImageTag,omitempty"`

	// LabelsSet notifies about Pod resources created without labels.
	LabelsSet *bool `yaml:"labelsSet,omitempty"`
}

// KubernetesEvent contains configuration for Kubernetes events.
type KubernetesEvent struct {
	Reason  string                       `yaml:"reason"`
	Message string                       `yaml:"message"`
	Types   KubernetesResourceEventTypes `yaml:"types"`
}

// KubernetesResourceEventTypes contains events to watch for a resource.
type KubernetesResourceEventTypes []EventType

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

// Resource contains resources to watch
type Resource struct {
	Type          string            `yaml:"type"`
	Name          string            `yaml:"name"`
	Namespaces    Namespaces        `yaml:"namespaces"`
	Annotations   map[string]string `yaml:"annotations"`
	Labels        map[string]string `yaml:"labels"`
	Event         KubernetesEvent   `yaml:"event"`
	UpdateSetting UpdateSetting     `yaml:"updateSetting"`
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

// UpdateSetting struct defines updateEvent fields specification
type UpdateSetting struct {
	Fields      []string `yaml:"fields"`
	IncludeDiff bool     `yaml:"includeDiff"`
}

// Log logging configuration
type Log struct {
	Level string `yaml:"level"`
}

// MergeConfigs merges all input configuration.
func MergeConfigs(configs []*source.Config) (Config, error) {
	out := Config{
		Log: &Log{
			Level: "info",
		},
	}
	for _, rawCfg := range configs {
		var cfg Config
		err := yaml.Unmarshal(rawCfg.RawYAML, &cfg)
		if err != nil {
			return Config{}, fmt.Errorf("while unmarshalling YAML config: %w", err)
		}

		if cfg.Log != nil && cfg.Log.Level != "" {
			out.Log = &Log{Level: cfg.Log.Level}
		}

		if cfg.KubeConfig != "" {
			out.KubeConfig = cfg.KubeConfig
		}
	}

	return out, nil
}
