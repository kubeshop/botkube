package config

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RuntimeState represents the runtime state.
type RuntimeState struct {
	Communications map[string]CommunicationsRuntimeState `yaml:"communications,omitempty"`
}

// MarshalToMap marshals the runtime state to a string map.
func (s RuntimeState) MarshalToMap(cfg PartialPersistentConfig) (map[string]string, error) {
	return marshalToMap(&s, cfg.FileName)
}

// CommunicationsRuntimeState represents the runtime state for communication groups.
type CommunicationsRuntimeState map[CommPlatformIntegration]BotRuntimeState

// BotRuntimeState represents the runtime state for a bot.
type BotRuntimeState struct {
	Channels map[string]ChannelRuntimeState `yaml:"channels,omitempty"`

	// Teams integration only, ignored for other communication platforms.
	MSTeamsOnlyRuntimeState *ChannelRuntimeState `yaml:",inline,omitempty"`
}

// ChannelRuntimeState represents the runtime state for a channel.
type ChannelRuntimeState struct {
	Bindings ChannelRuntimeBindings `yaml:"bindings"`
}

// ChannelRuntimeBindings represents the bindings for a channel.
type ChannelRuntimeBindings struct {
	Sources []string `yaml:"sources"`
}

// StartupState represents the startup state.
type StartupState struct {
	Communications map[string]CommunicationsStartupState `yaml:"communications,omitempty"`
	Filters        Filters                               `yaml:"filters,omitempty"`
}

// MarshalToMap marshals the startup state to a string map.
func (s StartupState) MarshalToMap(cfg PartialPersistentConfig) (map[string]string, error) {
	return marshalToMap(&s, cfg.FileName)
}

// CommunicationsStartupState represents the startup state for communication groups.
type CommunicationsStartupState map[CommPlatformIntegration]BotStartupState

// BotStartupState represents the startup state for a bot.
type BotStartupState struct {
	Channels map[string]ChannelStartupState `yaml:"channels"`
}

// ChannelStartupState represents the startup state for a channel.
type ChannelStartupState struct {
	Notification NotificationStartupState `yaml:"notification"`
}

// NotificationStartupState represents the startup state for a notification.
type NotificationStartupState struct {
	Disabled bool `yaml:"disabled"`
}

func marshalToMap(in interface{}, propertyName string) (map[string]string, error) {
	bytes, err := marshalToYAMLString(&in)
	if err != nil {
		return nil, fmt.Errorf("while marshalling %q: %w", propertyName, err)
	}

	return map[string]string{
		propertyName: bytes,
	}, nil
}

func marshalToYAMLString(in interface{}) (string, error) {
	var buff strings.Builder
	encode := yaml.NewEncoder(&buff)
	encode.SetIndent(2)
	err := encode.Encode(in)
	if err != nil {
		return "", err
	}

	return buff.String(), err
}

type marshalableState interface {
	MarshalToMap(cfg PartialPersistentConfig) (map[string]string, error)
}

type configMapStorage[T marshalableState] struct {
	k8sCli kubernetes.Interface
	cfg    PartialPersistentConfig
}

func (s *configMapStorage[T]) Get(ctx context.Context) (T, *v1.ConfigMap, error) {
	var emptyState T
	cm, err := s.k8sCli.CoreV1().ConfigMaps(s.cfg.ConfigMap.Namespace).Get(ctx, s.cfg.ConfigMap.Name, metav1.GetOptions{})
	if err != nil {
		return emptyState, nil, fmt.Errorf("while getting the ConfigMap: %w", err)
	}

	var state T
	runtimeStateStr, exists := cm.Data[s.cfg.FileName]
	if !exists {
		return emptyState, nil, nil
	}

	err = yaml.Unmarshal([]byte(runtimeStateStr), &state)
	if err != nil {
		return emptyState, nil, fmt.Errorf("while unmarshalling %q: %w", s.cfg.FileName, err)
	}

	return state, cm, nil
}

func (s *configMapStorage[T]) Update(ctx context.Context, originalCM *v1.ConfigMap, state T) error {
	data, err := state.MarshalToMap(s.cfg)
	if err != nil {
		return fmt.Errorf("while marshalling data")
	}

	cmToUpdate := originalCM.DeepCopy()
	cmToUpdate.Data = data
	_, err = s.k8sCli.CoreV1().ConfigMaps(cmToUpdate.Namespace).Update(ctx, cmToUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("while updating the ConfigMap with help details: %w", err)
	}

	return nil
}
