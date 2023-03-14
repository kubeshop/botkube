package config

import (
	"context"
	"sync"

	"github.com/hasura/go-graphql-client"
	"github.com/sirupsen/logrus"

	remoteapi "github.com/kubeshop/botkube/internal/remote"
)

// RemotePersistenceManager manages persistence of the configuration.
type RemotePersistenceManager struct {
	log             logrus.FieldLogger
	gql             GraphQLClient
	resourceVersion int
	resVerMutex     sync.RWMutex
}

var _ PersistenceManager = (*RemotePersistenceManager)(nil)

func (m *RemotePersistenceManager) PersistNotificationsEnabled(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, enabled bool) error {
	logger := m.log.WithFields(logrus.Fields{
		"deploymentID":    m.gql.DeploymentID,
		"resourceVersion": m.getResourceVersion(),
		"commGroupName":   commGroupName,
		"platform":        platform.String(),
		"channelAlias":    channelAlias,
		"disabled":        !enabled,
	})
	logger.Debug("Updating notification configuration")

	if _, ok := supportedPlatformsSourceBindings[platform]; !ok {
		return ErrUnsupportedPlatform
	}

	p, err := remoteapi.NewBotPlatform(platform.String())
	if err != nil {
		return ErrUnsupportedPlatform
	}
	var mutation struct {
		Success bool `graphql:"patchDeploymentConfig(id: $id, input: $input)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(m.gql.DeploymentID()),
		"input": remoteapi.PatchDeploymentConfigInput{
			ResourceVersion: m.getResourceVersion(),
			Notification: &remoteapi.NotificationPatchDeploymentConfigInput{
				CommunicationGroupName: commGroupName,
				Platform:               p,
				ChannelAlias:           channelAlias,
				Disabled:               !enabled,
			},
		},
	}

	return m.gql.Client().Mutate(ctx, &mutation, variables)
}

func (m *RemotePersistenceManager) PersistSourceBindings(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, sourceBindings []string) error {
	logger := m.log.WithFields(logrus.Fields{
		"deploymentID":    m.gql.DeploymentID,
		"resourceVersion": m.getResourceVersion(),
		"commGroupName":   commGroupName,
		"platform":        platform.String(),
		"channelAlias":    channelAlias,
		"sourceBindings":  sourceBindings,
	})
	logger.Debug("Updating source bindings configuration")

	if _, ok := supportedPlatformsNotifications[platform]; !ok {
		return ErrUnsupportedPlatform
	}

	p, err := remoteapi.NewBotPlatform(string(platform))
	if err != nil {
		return ErrUnsupportedPlatform
	}
	var mutation struct {
		Success bool `graphql:"patchDeploymentConfig(id: $id, input: $input)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(m.gql.DeploymentID()),
		"input": remoteapi.PatchDeploymentConfigInput{
			ResourceVersion: m.getResourceVersion(),
			SourceBinding: &remoteapi.SourceBindingPatchDeploymentConfigInput{
				CommunicationGroupName: commGroupName,
				Platform:               p,
				ChannelAlias:           channelAlias,
				SourceBindings:         sourceBindings,
			},
		},
	}

	return m.gql.Client().Mutate(ctx, &mutation, variables)
}

func (m *RemotePersistenceManager) PersistActionEnabled(ctx context.Context, name string, enabled bool) error {
	panic("Implement me")
}

func (m *RemotePersistenceManager) SetResourceVersion(resourceVersion int) {
	m.resVerMutex.Lock()
	defer m.resVerMutex.Unlock()
	m.resourceVersion = resourceVersion
}

func (m *RemotePersistenceManager) getResourceVersion() int {
	m.resVerMutex.RLock()
	defer m.resVerMutex.RUnlock()
	return m.resourceVersion
}
