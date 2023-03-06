package config

import (
	"context"
	"errors"

	"github.com/kubeshop/botkube/internal/graphql"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

var (
	supportedPlatformsSourceBindings = map[CommPlatformIntegration]struct{}{
		SlackCommPlatformIntegration:       {},
		SocketSlackCommPlatformIntegration: {},
		DiscordCommPlatformIntegration:     {},
		MattermostCommPlatformIntegration:  {},
		TeamsCommPlatformIntegration:       {},
	}
	supportedPlatformsNotifications = map[CommPlatformIntegration]struct{}{
		SlackCommPlatformIntegration:       {},
		SocketSlackCommPlatformIntegration: {},
		DiscordCommPlatformIntegration:     {},
		MattermostCommPlatformIntegration:  {},
	}
)

// ConfigPersistenceManager manages persistence of the configuration.
type ConfigPersistenceManager interface {
	PersistSourceBindings(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, sourceBindings []string) error
	PersistNotificationsEnabled(ctx context.Context, commGroupName string, platform CommPlatformIntegration, channelAlias string, enabled bool) error
	PersistFilterEnabled(ctx context.Context, name string, enabled bool) error
	PersistActionEnabled(ctx context.Context, name string, enabled bool) error
	SetResourceVersion(resourceVersion int)
}

// ErrUnsupportedPlatform is an error returned when a platform is not supported.
var ErrUnsupportedPlatform = errors.New("unsupported platform to persist data")

// NewManager creates a new PersistenceManager instance.
func NewManager(remoteCfgEnabled bool, log logrus.FieldLogger, cfg PersistentConfig, k8sCli kubernetes.Interface, gql *graphql.Gql) ConfigPersistenceManager {
	if remoteCfgEnabled {
		return &RemoteConfigPersistenceManager{
			log: log,
			gql: gql,
		}
	}
	return &LocalConfigPersistenceManager{
		log:    log,
		cfg:    cfg,
		k8sCli: k8sCli,
	}
}
