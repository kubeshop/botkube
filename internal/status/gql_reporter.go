package status

import (
	"context"

	"github.com/hasura/go-graphql-client"
	"github.com/sirupsen/logrus"

	gql "github.com/kubeshop/botkube/internal/graphql"
)

var _ StatusReporter = (*GraphQLStatusReporter)(nil)

type GraphQLStatusReporter struct {
	log logrus.FieldLogger
	gql *gql.Gql
}

func newGraphQLStatusReporter(logger logrus.FieldLogger, client *gql.Gql) *GraphQLStatusReporter {
	return &GraphQLStatusReporter{
		log: logger,
		gql: client,
	}
}

func (r *GraphQLStatusReporter) ReportDeploymentStartup(ctx context.Context) (bool, error) {
	r.log.Debugf("Reporting deployment startup for ID: %s", r.gql.DeploymentID)
	var mutation struct {
		Success bool `graphql:"reportDeploymentStartup(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.gql.DeploymentID),
	}
	if err := r.gql.Cli.Mutate(ctx, &mutation, variables); err != nil {
		return false, err
	}
	return mutation.Success, nil
}

func (r *GraphQLStatusReporter) ReportDeploymentShutdown(ctx context.Context) (bool, error) {
	r.log.Debugf("Reporting deployment shutdown for ID: %s", r.gql.DeploymentID)
	var mutation struct {
		Success bool `graphql:"reportDeploymentShutdown(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.gql.DeploymentID),
	}
	if err := r.gql.Cli.Mutate(ctx, &mutation, variables); err != nil {
		return false, err
	}
	return mutation.Success, nil
}

func (r *GraphQLStatusReporter) ReportDeploymentFailed(ctx context.Context) (bool, error) {
	r.log.Debugf("Reporting deployment failure for ID: %s", r.gql.DeploymentID)
	var mutation struct {
		Success bool `graphql:"reportDeploymentFailed(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": graphql.ID(r.gql.DeploymentID),
	}
	if err := r.gql.Cli.Mutate(ctx, &mutation, variables); err != nil {
		return false, err
	}
	return mutation.Success, nil
}
