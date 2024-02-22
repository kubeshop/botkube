package config

import (
	"context"
	"errors"

	"github.com/hasura/go-graphql-client"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

var (
	supportedPlatformsSourceBindings = map[CommPlatformIntegration]struct{}{
		CloudSlackCommPlatformIntegration:  {},
		SocketSlackCommPlatformIntegration: {},
		DiscordCommPlatformIntegration:     {},
		MattermostCommPlatformIntegration:  {},
		CloudTeamsCommPlatformIntegration:  {},
	}
	supportedPlatformsNotifications = map[CommPlatformIntegration]struct{}{
		CloudSlackCommPlatformIntegration:  {},
		SocketSlackCommPlatformIntegration: {},
		DiscordCommPlatformIntegration:     {},
		MattermostCommPlatformIntegration:  {},
		CloudTeamsCommPlatformIntegration:  {},
	}
)

// GraphQLClient defines GraphQL client.
type GraphQLClient interface {
	Client() *graphql.Client
	DeploymentID() string
}

// PersistenceManager manages persistence of the configuration.
type PersistenceManager interface {
	PersistSourceBindings(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, sourceBindings []string) error
	PersistNotificationsEnabled(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, enabled bool) error
	PersistActionEnabled(ctx context.Context, name string, enabled bool) error
	SetResourceVersion(resourceVersion int)
}

// ErrUnsupportedPlatform is an error returned when a platform is not supported.
var ErrUnsupportedPlatform = errors.New("unsupported platform to persist data")

// NewManager creates a new PersistenceManager instance.
func NewManager(remoteCfgEnabled bool, log logrus.FieldLogger, cfg PersistentConfig, cfgVersion int, k8sCli kubernetes.Interface, client GraphQLClient, resVerClient ResVerClient) PersistenceManager {
	if remoteCfgEnabled {
		return &RemotePersistenceManager{
			log:             log,
			gql:             client,
			resourceVersion: cfgVersion,
			resVerClient:    resVerClient,
		}
	}
	return &K8sConfigPersistenceManager{
		log:    log,
		cfg:    cfg,
		k8sCli: k8sCli,
	}
}
