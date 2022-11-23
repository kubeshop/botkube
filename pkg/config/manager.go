package config

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/strings/slices"
)

// PersistenceManager manages persistence of the configuration.
type PersistenceManager struct {
	log    logrus.FieldLogger
	cfg    PersistentConfig
	k8sCli kubernetes.Interface
}

// ErrUnsupportedPlatform is an error returned when a platform is not supported.
var ErrUnsupportedPlatform = errors.New("unsupported platform to persist data")

// NewManager creates a new PersistenceManager instance.
func NewManager(log logrus.FieldLogger, cfg PersistentConfig, k8sCli kubernetes.Interface) *PersistenceManager {
	return &PersistenceManager{
		log:    log,
		cfg:    cfg,
		k8sCli: k8sCli,
	}
}

// PersistSourceBindings persists source bindings configuration for a given channel in a given platform.
func (m *PersistenceManager) PersistSourceBindings(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, sourceBindings []string) error {
	supportedPlatforms := []string{
		string(SlackCommPlatformIntegration),
		string(SocketSlackCommPlatformIntegration),
		string(DiscordCommPlatformIntegration),
		string(MattermostCommPlatformIntegration),
		string(TeamsCommPlatformIntegration),
	}

	if !slices.Contains(supportedPlatforms, string(platform)) {
		return ErrUnsupportedPlatform
	}

	configMapStorage := configMapStorage[RuntimeState]{k8sCli: m.k8sCli, cfg: m.cfg.Runtime}

	state, cm, err := configMapStorage.Get(ctx)
	if err != nil {
		return err
	}

	if state.Communications == nil {
		state.Communications = make(map[string]CommunicationsRuntimeState)
	}
	commGroup, exists := state.Communications[commGroupName]
	if !exists {
		commGroup = make(CommunicationsRuntimeState)
		state.Communications[commGroupName] = commGroup
	}

	platformCfg, exists := commGroup[platform]
	if !exists {
		platformCfg = BotRuntimeState{}
		state.Communications[commGroupName][platform] = platformCfg
	}

	if platform == TeamsCommPlatformIntegration {
		if platformCfg.MSTeamsOnlyRuntimeState == nil {
			platformCfg.MSTeamsOnlyRuntimeState = &ChannelRuntimeState{}
		}

		platformCfg.MSTeamsOnlyRuntimeState.Bindings.Sources = sourceBindings
		state.Communications[commGroupName][platform] = platformCfg

		err = configMapStorage.Update(ctx, cm, state)
		if err != nil {
			return err
		}

		return nil
	}

	if platformCfg.Channels == nil {
		platformCfg.Channels = make(map[string]ChannelRuntimeState)
		state.Communications[commGroupName][platform] = platformCfg
	}

	channel, exists := platformCfg.Channels[channelAlias]
	if !exists {
		channel = ChannelRuntimeState{}
	}

	channel.Bindings.Sources = sourceBindings
	state.Communications[commGroupName][platform].Channels[channelAlias] = channel

	err = configMapStorage.Update(ctx, cm, state)
	if err != nil {
		return err
	}

	return nil
}

// PersistNotificationsEnabled persists notifications state for a given channel.
// While this method updates the Botkube ConfigMap, it doesn't reload Botkube itself.
func (m *PersistenceManager) PersistNotificationsEnabled(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, enabled bool) error {
	supportedPlatforms := []string{
		string(SlackCommPlatformIntegration),
		string(SocketSlackCommPlatformIntegration),
		string(DiscordCommPlatformIntegration),
		string(MattermostCommPlatformIntegration),
	}

	if !slices.Contains(supportedPlatforms, string(platform)) {
		return ErrUnsupportedPlatform
	}

	cmStorage := configMapStorage[StartupState]{k8sCli: m.k8sCli, cfg: m.cfg.Startup}
	state, cm, err := cmStorage.Get(ctx)
	if err != nil {
		return err
	}

	if state.Communications == nil {
		state.Communications = make(map[string]CommunicationsStartupState)
	}
	commGroup, exists := state.Communications[commGroupName]
	if !exists {
		commGroup = make(CommunicationsStartupState)
		state.Communications[commGroupName] = commGroup
	}

	platformCfg, exists := commGroup[platform]
	if !exists {
		platformCfg = BotStartupState{}
		state.Communications[commGroupName][platform] = platformCfg
	}

	if platformCfg.Channels == nil {
		platformCfg.Channels = make(map[string]ChannelStartupState)
		state.Communications[commGroupName][platform] = platformCfg
	}

	channel, exists := platformCfg.Channels[channelAlias]
	if !exists {
		channel = ChannelStartupState{}
	}

	channel.Notification.Disabled = !enabled
	state.Communications[commGroupName][platform].Channels[channelAlias] = channel

	err = cmStorage.Update(ctx, cm, state)
	if err != nil {
		return err
	}

	return nil
}

// PersistFilterEnabled persists status for a given filter.
// While this method updates the Botkube ConfigMap, it doesn't reload Botkube itself.
func (m *PersistenceManager) PersistFilterEnabled(ctx context.Context, name string, enabled bool) error {
	cmStorage := configMapStorage[StartupState]{k8sCli: m.k8sCli, cfg: m.cfg.Startup}

	state, cm, err := cmStorage.Get(ctx)
	if err != nil {
		return err
	}

	err = state.Filters.Kubernetes.SetEnabled(name, enabled)
	if err != nil {
		return err
	}

	err = cmStorage.Update(ctx, cm, state)
	if err != nil {
		return err
	}

	return nil
}

// PersistActionEnabled updates runtime config map with desired action.enabled parameter
func (m *PersistenceManager) PersistActionEnabled(ctx context.Context, name string, enabled bool) error {
	cmStorage := configMapStorage[RuntimeState]{k8sCli: m.k8sCli, cfg: m.cfg.Runtime}

	state, cm, err := cmStorage.Get(ctx)
	if err != nil {
		return err
	}
	if state.Actions == nil {
		state.Actions = ActionsRuntimeState{}
	}
	if err := state.Actions.SetEnabled(name, enabled); err != nil {
		return err
	}
	return cmStorage.Update(ctx, cm, state)
}
