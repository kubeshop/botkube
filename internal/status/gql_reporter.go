package status

import (
	"context"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/remote"
	"github.com/kubeshop/botkube/pkg/version"
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

func newGraphQLStatusReporter(logger logrus.FieldLogger, client GraphQLClient, resVerClient ResVerClient) *GraphQLStatusReporter {
	return &GraphQLStatusReporter{
		log:          logger,
		gql:          client,
		resVerClient: resVerClient,
	}
}

// ReportDeploymentConnectionInit reports connection initialization.
func (r *GraphQLStatusReporter) ReportDeploymentConnectionInit(ctx context.Context, k8sVer string) error {
	logger := r.log.WithFields(logrus.Fields{
		"deploymentID": r.gql.DeploymentID(),
		"type":         "connecting",
	})
	logger.Debug("Reporting...")

	err := r.withRetry(ctx, logger, func() error {
		var mutation struct {
			Success bool `graphql:"reportDeploymentConnectionInit(id: $id, botkubeVersion: $botkubeVersion, k8sVer: $k8sVer)"`
		}
		variables := map[string]interface{}{
			"id":             graphql.ID(r.gql.DeploymentID()),
			"botkubeVersion": version.Info().Version,
			"k8sVer":         k8sVer,
		}
		err := r.gql.Client().Mutate(ctx, &mutation, variables)
		if err != nil {
			return err
		}
		if !mutation.Success {
			return errors.New("failed to report connection initialization")
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "while reporting deployment connection initialization")
	}
	logger.Debug("Reporting successful.")

	return nil
}

// ReportDeploymentStartup reports deployment startup to GraphQL server.
func (r *GraphQLStatusReporter) ReportDeploymentStartup(ctx context.Context) error {
	logger := r.log.WithFields(logrus.Fields{
		"deploymentID":    r.gql.DeploymentID(),
		"resourceVersion": r.getResourceVersion(),
		"type":            "startup",
	})
	logger.Debug("Reporting...")

	err := r.withRetry(ctx, logger, func() error {
		var mutation struct {
			Success bool `graphql:"reportDeploymentStartup(id: $id, resourceVersion: $resourceVersion, botkubeVersion: $botkubeVersion)"`
		}
		variables := map[string]interface{}{
			"id":              graphql.ID(r.gql.DeploymentID()),
			"resourceVersion": r.getResourceVersion(),
			"botkubeVersion":  version.Info().Version,
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
		"deploymentID":    r.gql.DeploymentID(),
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

// ReportDeploymentFailure reports deployment failure to GraphQL server.
func (r *GraphQLStatusReporter) ReportDeploymentFailure(ctx context.Context, errMsg string) error {
	logger := r.log.WithFields(logrus.Fields{
		"deploymentID":    r.gql.DeploymentID(),
		"resourceVersion": r.getResourceVersion(),
		"type":            "failure",
	})
	logger.Debug("Reporting...")

	err := r.withRetry(ctx, logger, func() error {
		var mutation struct {
			Success bool `graphql:"reportDeploymentFailure(id: $id, in: $input)"`
		}

		variables := map[string]interface{}{
			"id": graphql.ID(r.gql.DeploymentID()),
			"input": remote.DeploymentFailureInput{
				Message:         errMsg,
				ResourceVersion: r.getResourceVersion(),
			},
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

func (r *GraphQLStatusReporter) SetLogger(logger logrus.FieldLogger) {
	r.log = logger.WithField("component", "GraphQLStatusReporter")
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

func (r *GraphQLStatusReporter) getResourceVersion() int {
	r.resVerMutex.RLock()
	defer r.resVerMutex.RUnlock()
	return r.resourceVersion
}
