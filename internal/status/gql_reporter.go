package status

import (
	"context"
	"sync"

	"github.com/hasura/go-graphql-client"
	"github.com/sirupsen/logrus"

	gql "github.com/kubeshop/botkube/internal/graphql"
)

var _ StatusReporter = (*GraphQLStatusReporter)(nil)

type GraphQLStatusReporter struct {
	log logrus.FieldLogger
	gql *gql.Gql
	resourceVersion int
	resVerMutex sync.RWMutex
}

func newGraphQLStatusReporter(logger logrus.FieldLogger, client *gql.Gql) *GraphQLStatusReporter {
	return &GraphQLStatusReporter{
		log: logger,
		gql: client,
	}
}

func (r *GraphQLStatusReporter) ReportDeploymentStartup(ctx context.Context) (bool, error) {
	r.log.WithFields(logrus.Fields{
			"deploymentID": r.gql.DeploymentID,
			"resourceVersion": r.getResourceVersion(),
	}).Debugf("Reporting deployment startup...")
	var mutation struct {
		Success bool `graphql:"reportDeploymentStartup(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.gql.DeploymentID),
		"resourceVersion": r.getResourceVersion(),
	}
	if err := r.gql.Cli.Mutate(ctx, &mutation, variables); err != nil {
		return false, err
	}
	return mutation.Success, nil
}

func (r *GraphQLStatusReporter) ReportDeploymentShutdown(ctx context.Context) (bool, error) {
	r.log.WithFields(logrus.Fields{
		"deploymentID": r.gql.DeploymentID,
		"resourceVersion": r.getResourceVersion(),
	}).Debugf("Reporting deployment shutdown...")
	var mutation struct {
		Success bool `graphql:"reportDeploymentShutdown(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.gql.DeploymentID),
		"resourceVersion": r.getResourceVersion(),
	}
	if err := r.gql.Cli.Mutate(ctx, &mutation, variables); err != nil {
		return false, err
	}
	return mutation.Success, nil
}

func (r *GraphQLStatusReporter) ReportDeploymentFailed(ctx context.Context) (bool, error) {
	r.log.WithFields(logrus.Fields{
		"deploymentID": r.gql.DeploymentID,
		"resourceVersion": r.getResourceVersion(),
	}).Debugf("Reporting deployment failure...")
	var mutation struct {
		Success bool `graphql:"reportDeploymentFailed(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.gql.DeploymentID),
		"resourceVersion": r.getResourceVersion(),
	}
	if err := r.gql.Cli.Mutate(ctx, &mutation, variables); err != nil {
		return false, err
	}
	return mutation.Success, nil
}

func (r *GraphQLStatusReporter) getResourceVersion() int {
	r.resVerMutex.RLock()
	defer r.resVerMutex.RUnlock()
	return r.resourceVersion
}

func (r *GraphQLStatusReporter) SetResourceVersion(resourceVersion int) {
	r.resVerMutex.Lock()
	defer r.resVerMutex.Unlock()
	r.resourceVersion = resourceVersion
}
