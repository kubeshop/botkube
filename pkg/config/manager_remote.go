package config

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hasura/go-graphql-client"
	gql "github.com/kubeshop/botkube/internal/graphql"
	"github.com/sirupsen/logrus"
)

// RemotePersistenceManager manages persistence of the configuration.
type RemoteConfigPersistenceManager struct {
	log             logrus.FieldLogger
	gql             *gql.Gql
	resourceVersion int
	resVerMutex     sync.RWMutex
}

var _ ConfigPersistenceManager = (*RemoteConfigPersistenceManager)(nil)

func (m *RemoteConfigPersistenceManager) PersistNotificationsEnabled(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, enabled bool) error {
	m.log.Debugf("Sending new notification configuration for: %s", m.gql.DeploymentID)

	if _, ok := supportedPlatformsSourceBindings[platform]; !ok {
		return ErrUnsupportedPlatform
	}

	p, err := NewBotPlatform(string(platform))
	if err != nil {
		return ErrUnsupportedPlatform
	}
	var mutation struct {
		PatchDeploymentConfig struct {
			ID graphql.ID
		} `graphql:"patchDeploymentConfig(input: $input)"`
	}
	variables := map[string]interface{}{
		"input": PatchDeploymentConfigInput{
			ResourceVersion: m.getResourceVersion(),
			Notification: &NotificationPatchDeploymentConfigInput{
				CommunicationGroupName: commGroupName,
				Platform:               p,
				ChannelAlias:           channelAlias,
				Disabled:               !enabled,
			},
		},
	}

	return m.gql.Cli.Mutate(ctx, &mutation, variables)
}

func (m *RemoteConfigPersistenceManager) PersistSourceBindings(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, sourceBindings []string) error {
	m.log.Debugf("Sending new source binding configuration for: %s", m.gql.DeploymentID)

	if _, ok := supportedPlatformsNotifications[platform]; !ok {
		return ErrUnsupportedPlatform
	}

	p, err := NewBotPlatform(string(platform))
	if err != nil {
		return ErrUnsupportedPlatform
	}
	var mutation struct {
		PatchDeploymentConfig struct {
			ID graphql.ID
		} `graphql:"patchDeploymentConfig(input: $input)"`
	}
	variables := map[string]interface{}{
		"input": PatchDeploymentConfigInput{
			ResourceVersion: m.getResourceVersion(),
			SourceBinding: &SourceBindingPatchDeploymentConfigInput{
				CommunicationGroupName: commGroupName,
				Platform:               p,
				ChannelAlias:           channelAlias,
				SourceBindings:         sourceBindings,
			},
		},
	}

	return m.gql.Cli.Mutate(ctx, &mutation, variables)
}

func (m *RemoteConfigPersistenceManager) PersistFilterEnabled(ctx context.Context, name string, enabled bool) error {
	panic("Filter moved to kubectl plugin")
}

func (m *RemoteConfigPersistenceManager) PersistActionEnabled(ctx context.Context, name string, enabled bool) error {
	panic("Implement me")
}

type PatchDeploymentConfigInput struct {
	ResourceVersion int                                      `json:"resourceVersion"`
	Notification    *NotificationPatchDeploymentConfigInput  `json:"notification"`
	SourceBinding   *SourceBindingPatchDeploymentConfigInput `json:"sourceBinding"`
}

type NotificationPatchDeploymentConfigInput struct {
	CommunicationGroupName string      `json:"communicationGroupName"`
	Platform               BotPlatform `json:"platform"`
	ChannelAlias           string      `json:"channelAlias"`
	Disabled               bool        `json:"disabled"`
}

type SourceBindingPatchDeploymentConfigInput struct {
	CommunicationGroupName string      `json:"communicationGroupName"`
	Platform               BotPlatform `json:"platform"`
	ChannelAlias           string      `json:"channelAlias"`
	SourceBindings         []string    `json:"sourceBindings"`
}

type BotPlatform string

const (
	// BotPlatformSlack is the slack platform
	BotPlatformSlack BotPlatform = "SLACK"
	// BotPlatformDiscord is the discord platform
	BotPlatformDiscord BotPlatform = "DISCORD"
	// BotPlatformMattermost is the mattermost platform
	BotPlatformMattermost BotPlatform = "MATTERMOST"
	// BotPlatformMsTeams is the teams platform
	BotPlatformMsTeams BotPlatform = "MS_TEAMS"
)

// NewBotPlatform creates new BotPlatform from string
func NewBotPlatform(s string) (BotPlatform, error) {
	switch strings.ToUpper(s) {
	case "SLACK":
		fallthrough
	case "SOCKETSLACK":
		return BotPlatformSlack, nil
	case "DISCORD":
		return BotPlatformDiscord, nil
	case "MATTERMOST":
		return BotPlatformMattermost, nil
	case "TEAMS":
		fallthrough
	case "MS_TEAMS":
		return BotPlatformMsTeams, nil
	default:
		return "", fmt.Errorf("given BotPlatform %s is not supported", s)
	}
}

func (m *RemoteConfigPersistenceManager) SetResourceVersion(resourceVersion int) {
	m.resVerMutex.Lock()
	defer m.resVerMutex.Unlock()
	m.resourceVersion = resourceVersion
}

func (m *RemoteConfigPersistenceManager) getResourceVersion() int {
	m.resVerMutex.RLock()
	defer m.resVerMutex.RUnlock()
	return m.resourceVersion
}
