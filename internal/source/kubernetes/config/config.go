package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/pluginx"
	"github.com/kubeshop/botkube/pkg/ptr"
)

// Config Kubernetes configuration
type Config struct {
	InformerReSyncPeriod *time.Duration     `yaml:"informerReSyncPeriod"`
	Log                  *Log               `yaml:"log"`
	Recommendations      *Recommendations   `yaml:"recommendations"`
	Event                *KubernetesEvent   `yaml:"event"`
	Resources            []Resource         `yaml:"resources" validate:"dive"`
	ActionVerbs          []string           `yaml:"actionVerbs" validate:"dive"`
	ActionResources      []string           `yaml:"actionResources" validate:"dive"`
	Namespaces           *RegexConstraints  `yaml:"namespaces"`
	Annotations          *map[string]string `yaml:"annotations"`
	Labels               *map[string]string `yaml:"labels"`
	Filters              *Filters           `yaml:"filters"`
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
	Reason  RegexConstraints             `yaml:"reason"`
	Message RegexConstraints             `yaml:"message"`
	Types   KubernetesResourceEventTypes `yaml:"types"`
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

// AreConstraintsDefined checks if any of the event constraints are defined.
func (e KubernetesEvent) AreConstraintsDefined() bool {
	return e.Reason.AreConstraintsDefined() || e.Message.AreConstraintsDefined()
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

func (eventType EventType) String() string {
	return string(eventType)
}

// Resource contains resources to watch
type Resource struct {
	Type          string            `yaml:"type"`
	Name          RegexConstraints  `yaml:"name"`
	Namespaces    RegexConstraints  `yaml:"namespaces"`
	Annotations   map[string]string `yaml:"annotations"`
	Labels        map[string]string `yaml:"labels"`
	Event         KubernetesEvent   `yaml:"event"`
	UpdateSetting UpdateSetting     `yaml:"updateSetting"`
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

// MergeConfigs merges all input configuration.
func MergeConfigs(configs []*source.Config) (Config, error) {
	t := 30 * time.Minute
	defaults := Config{
		Log: &Log{
			Level: "info",
		},
		InformerReSyncPeriod: &t,
		Recommendations: &Recommendations{
			Pod: PodRecommendations{
				NoLatestImageTag: ptr.Bool(false),
				LabelsSet:        ptr.Bool(false),
			},
			Ingress: IngressRecommendations{
				BackendServiceValid: ptr.Bool(false),
				TLSSecretValid:      ptr.Bool(false),
			},
		},
		ActionVerbs:     []string{"api-resources", "api-versions", "cluster-info", "describe", "explain", "get", "logs", "top"},
		ActionResources: []string{"deployments", "pods", "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes", "configmaps", "services", "ingresses"},
		Filters: &Filters{Kubernetes: KubernetesFilters{
			ObjectAnnotationChecker: true,
			NodeEventsChecker:       true,
		}},
	}
	var out Config
	if err := pluginx.MergeSourceConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, err
	}

	return out, nil
}

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

const (
	// AllNamespaceIndicator represents a keyword for allowing all Kubernetes Namespaces.
	AllNamespaceIndicator = ".*"
)
