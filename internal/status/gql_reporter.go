package status

import (
	"context"

	"github.com/hasura/go-graphql-client"
	"github.com/sirupsen/logrus"
)

var _ StatusReporter = (*GraphQLStatusReporter)(nil)

type GraphQLStatusReporter struct {
	log          logrus.FieldLogger
	deploymentID string
	gql          *graphql.Client
}

func newGraphQLStatusReporter(logger logrus.FieldLogger, url, deploymentID string) *GraphQLStatusReporter {
	return &GraphQLStatusReporter{
		log:          logger,
		deploymentID: deploymentID,
		gql:          graphql.NewClient(url, nil),
	}
}

func (r *GraphQLStatusReporter) ReportDeploymentStartup(ctx context.Context) (bool, error) {
	r.log.Debugf("Reporting deployment startup for ID: %s", r.deploymentID)
	var mutation struct {
		Success bool `graphql:"reportDeploymentStartup(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.deploymentID),
	}
	if err := r.gql.Mutate(ctx, &mutation, variables); err != nil {
		return false, err
	}
	return mutation.Success, nil
}

func (r *GraphQLStatusReporter) ReportDeploymentShutdown(ctx context.Context) (bool, error) {
	r.log.Debugf("Reporting deployment shutdown for ID: %s", r.deploymentID)
	var mutation struct {
		Success bool `graphql:"reportDeploymentShutdown(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.deploymentID),
	}
	if err := r.gql.Mutate(ctx, &mutation, variables); err != nil {
		return false, err
	}
	return mutation.Success, nil
}

func (r *GraphQLStatusReporter) ReportDeploymentFailed(ctx context.Context) (bool, error) {
	r.log.Debugf("Reporting deployment failure for ID: %s", r.deploymentID)
	var mutation struct {
		Success bool `graphql:"reportDeploymentFailed(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.deploymentID),
	}
	if err := r.gql.Mutate(ctx, &mutation, variables); err != nil {
		return false, err
	}
	return mutation.Success, nil
}
