package status

import (
	"context"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ StatusReporter = (*GraphQLStatusReporter)(nil)

// GraphQLClient defines GraphQL client.
type GraphQLClient interface {
	Client() *graphql.Client
	DeploymentID() string
}

// ResVerClient defines client for getting resource version.
type ResVerClient interface {
	GetResourceVersion(ctx context.Context) (int, error)
}

// GraphQLStatusReporter reports status to GraphQL server.
type GraphQLStatusReporter struct {
	log             logrus.FieldLogger
	gql             GraphQLClient
	resVerClient    ResVerClient
	resourceVersion int
	resVerMutex     sync.RWMutex
}

func newGraphQLStatusReporter(logger logrus.FieldLogger, client GraphQLClient, resVerClient ResVerClient, cfgVersion int) *GraphQLStatusReporter {
	return &GraphQLStatusReporter{
		log:             logger,
		gql:             client,
		resVerClient:    resVerClient,
		resourceVersion: cfgVersion,
	}
}

// ReportDeploymentStartup reports deployment startup to GraphQL server.
func (r *GraphQLStatusReporter) ReportDeploymentStartup(ctx context.Context) error {
	logger := r.log.WithFields(logrus.Fields{
		"deploymentID":    r.gql.DeploymentID,
		"resourceVersion": r.getResourceVersion(),
		"type":            "startup",
	})
	logger.Debug("Reporting...")

	err := r.withRetry(ctx, logger, func() error {
		var mutation struct {
			Success bool `graphql:"reportDeploymentStartup(id: $id, resourceVersion: $resourceVersion)"`
		}
		variables := map[string]interface{}{
			"id":              graphql.ID(r.gql.DeploymentID()),
			"resourceVersion": r.getResourceVersion(),
		}
		err := r.gql.Client().Mutate(ctx, &mutation, variables)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "while reporting deployment startup")
	}

	logger.Debug("Reporting successful.")
	return nil
}

// ReportDeploymentShutdown reports deployment shutdown to GraphQL server.
func (r *GraphQLStatusReporter) ReportDeploymentShutdown(ctx context.Context) error {
	logger := r.log.WithFields(logrus.Fields{
		"deploymentID":    r.gql.DeploymentID,
		"resourceVersion": r.getResourceVersion(),
		"type":            "shutdown",
	})
	logger.Debug("Reporting...")

	err := r.withRetry(ctx, logger, func() error {
		var mutation struct {
			Success bool `graphql:"reportDeploymentShutdown(id: $id, resourceVersion: $resourceVersion)"`
		}
		variables := map[string]interface{}{
			"id":              graphql.ID(r.gql.DeploymentID()),
			"resourceVersion": r.getResourceVersion(),
		}
		return r.gql.Client().Mutate(ctx, &mutation, variables)
	})
	if err != nil {
		return errors.Wrap(err, "while reporting deployment shutdown")
	}

	logger.Debug("Reporting successful.")
	return nil
}

// ReportDeploymentFailed reports deployment failure to GraphQL server.
func (r *GraphQLStatusReporter) ReportDeploymentFailed(ctx context.Context) error {
	logger := r.log.WithFields(logrus.Fields{
		"deploymentID":    r.gql.DeploymentID,
		"resourceVersion": r.getResourceVersion(),
		"type":            "failure",
	})
	logger.Debug("Reporting...")

	err := r.withRetry(ctx, logger, func() error {
		var mutation struct {
			Success bool `graphql:"reportDeploymentFailed(id: $id, resourceVersion: $resourceVersion)"`
		}
		variables := map[string]interface{}{
			"id":              graphql.ID(r.gql.DeploymentID()),
			"resourceVersion": r.getResourceVersion(),
		}
		return r.gql.Client().Mutate(ctx, &mutation, variables)
	})
	if err != nil {
		return errors.Wrap(err, "while reporting deployment shutdown")
	}

	logger.Debug("Reporting successful.")
	return nil
}

// SetResourceVersion sets resource version.
func (r *GraphQLStatusReporter) SetResourceVersion(resourceVersion int) {
	r.resVerMutex.Lock()
	defer r.resVerMutex.Unlock()
	r.resourceVersion = resourceVersion
}

const (
	retries = 3
	delay   = 200 * time.Millisecond
)

func (r *GraphQLStatusReporter) withRetry(ctx context.Context, logger logrus.FieldLogger, fn func() error) error {
	err := retry.Do(
		fn,
		retry.OnRetry(func(n uint, err error) {
			logger.Debugf("Retrying (attempt no %d/%d): %s.\nFetching latest resource version...", n+1, retries, err)
			resVer, err := r.resVerClient.GetResourceVersion(ctx)
			if err != nil {
				logger.Errorf("Error while fetching resource version: %s", err)
			}
			r.SetResourceVersion(resVer)
		}),
		retry.Delay(delay),
		retry.Attempts(retries),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
	if err != nil {
		return errors.Wrap(err, "while retrying")
	}

	return nil
}

func (r *GraphQLStatusReporter) ReportHeartbeat(ctx context.Context, heartbeat DeploymentHeartbeatInput) error {
	logger := r.log.WithFields(logrus.Fields{
		"deploymentID": r.gql.DeploymentID,
		"heartbeat":    heartbeat,
	})
	logger.Debug("Sending heartbeat...")
	var mutation struct {
		Success bool `graphql:"reportDeploymentHeartbeat(id: $id, in: $heartbeat)"`
	}
	variables := map[string]interface{}{
		"id":        graphql.ID(r.gql.DeploymentID()),
		"heartbeat": heartbeat,
	}
	err := r.gql.Client().Mutate(ctx, &mutation, variables)
	if err != nil {
		return errors.Wrap(err, "while sending heartbeat")
	}

	logger.Debug("Sending heartbeat successful.")
	return nil
}

func (r *GraphQLStatusReporter) getResourceVersion() int {
	r.resVerMutex.RLock()
	defer r.resVerMutex.RUnlock()
	return r.resourceVersion
}
