package argocd

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// Config contains configuration for ArgoCD source plugin.
type Config struct {
	Log config.Logger `yaml:"log"`

	ArgoCD ArgoCD `yaml:"argoCD"`

	DefaultSubscriptions DefaultNotificationSubscriptions `yaml:"defaultSubscriptions"`

	Webhook       Webhook        `yaml:"webhook"`
	Notifications []Notification `yaml:"notifications"`
	Templates     []Template     `yaml:"templates"`

	Interactivity Interactivity `yaml:"interactivity"`
}

// Interactivity contains configuration related to interactivity.
type Interactivity struct {
	EnableViewInUIButton       bool     `yaml:"enableViewInUIButton"`
	EnableOpenRepositoryButton bool     `yaml:"enableOpenRepositoryButton"`
	CommandVerbs               []string `yaml:"commandVerbs"`
}

// ArgoCD contains configuration related to ArgoCD installation.
type ArgoCD struct {
	UIBaseURL              string                `yaml:"uiBaseUrl"`
	NotificationsConfigMap config.K8sResourceRef `yaml:"notificationsConfigMap"`
}

// Webhook contains configuration related to webhook.
type Webhook struct {
	Register bool   `yaml:"register"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
}

// Notification contains configuration related to notification.
type Notification struct {
	Trigger       NotificationTrigger       `yaml:"trigger"`
	Subscriptions NotificationSubscriptions `yaml:"subscriptions"`
}

// NotificationTrigger contains configuration related to notification trigger.
type NotificationTrigger struct {
	FromExisting *TriggerFromExisting `yaml:"fromExisting"`
	Create       *NewTrigger          `yaml:"create"`
}

// TriggerFromExisting contains configuration related to trigger from existing.
type TriggerFromExisting struct {
	Name         string `yaml:"name"`
	TemplateName string `yaml:"templateName"`
}

// NewTrigger contains configuration related to new trigger.
type NewTrigger struct {
	Name       string             `yaml:"name"`
	Conditions []TriggerCondition `yaml:"conditions"`
}

// RefByName contains configuration related to reference by name.
type RefByName struct {
	Name string `yaml:"name"`
}

// NotificationSubscriptions contains configuration related to notification subscriptions.
type NotificationSubscriptions struct {
	Create       bool                    `yaml:"create"`
	Applications []config.K8sResourceRef `yaml:"applications"`
}

// DefaultNotificationSubscriptions contains configuration related to default notification subscriptions.
type DefaultNotificationSubscriptions struct {
	Applications []config.K8sResourceRef `yaml:"applications"`
}

// Template contains configuration related to template.
type Template struct {
	Name string `yaml:"name"`
	Body string `yaml:"body"`
}

// TriggerCondition holds expression and template that must be used to create notification is expression is returns true
// Copied from `github.com/argoproj/notifications-engine@v0.4.0/pkg/triggers/service.go` and replaced json tags with yaml.
type TriggerCondition struct {
	OncePer     string   `yaml:"oncePer,omitempty"`
	When        string   `yaml:"when,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Send        []string `yaml:"send,omitempty"`
}

// mergeConfigs merges all input configuration.
func mergeConfigs(configs []*source.Config) (Config, error) {
	var defaultCfg Config
	err := yaml.Unmarshal([]byte(defaultConfigYAML), &defaultCfg)
	if err != nil {
		return Config{}, fmt.Errorf("while unmarshalling default config: %w", err)
	}

	var out Config
	if err := pluginx.MergeSourceConfigsWithDefaults(defaultCfg, configs, &out); err != nil {
		return Config{}, err
	}

	return out, nil
}
