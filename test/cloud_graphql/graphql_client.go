//go:build cloud_slack_dev_e2e

package cloud_graphql

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hasura/go-graphql-client"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/ptr"
	gqlModel "github.com/kubeshop/botkube/internal/remote/graphql"
)

const (
	botkubeOrganizationHeaderName  = "X-Botkube-Organization-Id"
	botkubeAuthorizationHeaderName = "Authorization"
	//nolint:gosec // G101: Potential hardcoded credentials
	botkubeAPIKeyHeaderName = "X-API-Key"
)

// Client provides helper functions for queries and mutations that  across different test cases.
// It simplifies setting up a given test prerequisites that are not a part of the test itself.
type Client struct {
	*graphql.Client
}

// MustCreateEmptyDeployment create empty deployment (without platform, plugins, etc.)
func (c *Client) MustCreateEmptyDeployment(t *testing.T) *gqlModel.Deployment {
	t.Helper()

	var mutation struct {
		CreateDeployment struct {
			ID                         string                                            `json:"id"`
			Name                       string                                            `json:"name"`
			Status                     *gqlModel.DeploymentStatus                        `json:"status"`
			APIKey                     *gqlModel.APIKey                                  `json:"apiKey"`
			YamlConfig                 *string                                           `json:"yamlConfig"`
			HelmCommand                *string                                           `json:"helmCommand"`
			InstallUpgradeInstructions []*gqlModel.InstallUpgradeInstructionsForPlatform `json:"installUpgradeInstructions"`
			ResourceVersion            int                                               `json:"resourceVersion"`
			Heartbeat                  *gqlModel.Heartbeat                               `json:"heartbeat"`
		} `graphql:"createDeployment(input: $input)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"input": gqlModel.DeploymentCreateInput{
			Name:      fmt.Sprintf("test/%s", t.Name()),
			Platforms: &gqlModel.PlatformsCreateInput{},
		},
	})
	require.NoError(t, err)

	return &gqlModel.Deployment{
		ID:                         mutation.CreateDeployment.ID,
		Name:                       mutation.CreateDeployment.Name,
		Status:                     mutation.CreateDeployment.Status,
		APIKey:                     mutation.CreateDeployment.APIKey,
		YamlConfig:                 mutation.CreateDeployment.YamlConfig,
		HelmCommand:                mutation.CreateDeployment.HelmCommand,
		InstallUpgradeInstructions: mutation.CreateDeployment.InstallUpgradeInstructions,
		ResourceVersion:            mutation.CreateDeployment.ResourceVersion,
		Heartbeat:                  mutation.CreateDeployment.Heartbeat,
	}
}

// MustCreateBasicDeployment create deployment with Slack platform and three plugins.
func (c *Client) MustCreateBasicDeployment(t *testing.T) *gqlModel.Deployment {
	t.Helper()

	var mutation struct {
		CreateDeployment gqlModel.Deployment `graphql:"createDeployment(input: $input)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"input": gqlModel.DeploymentCreateInput{
			Name:                 fmt.Sprintf("test/%s", t.Name()),
			AttachDefaultAliases: ptr.FromType(true),
			AttachDefaultActions: ptr.FromType(true),
			Plugins: []*gqlModel.PluginsCreateInput{
				{
					Groups: []*gqlModel.PluginConfigurationGroupInput{
						{
							Name:        "botkube/kubernetes",
							DisplayName: "Kubernetes Info",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "kubernetes_config",
									Configuration: "{\"recommendations\":{\"pod\":{\"noLatestImageTag\":true,\"labelsSet\":true},\"ingress\":{\"backendServiceValid\":true,\"tlsSecretValid\":true}}}",
								},
							},
							Enabled: true,
						},
						{
							Name:        "botkube/kubernetes",
							DisplayName: "Kubernetes Info2",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "kubernetes_config2",
									Configuration: "{\"recommendations\":{\"pod\":{\"noLatestImageTag\":true,\"labelsSet\":true},\"ingress\":{\"backendServiceValid\":true,\"tlsSecretValid\":true}}}",
								},
							},
							Enabled: true,
						},
						{
							Name:        "botkube/kubectl",
							DisplayName: "Kubectl",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "kubectl_config",
									Configuration: "{\"recommendations\":{\"pod\":{\"noLatestImageTag\":true,\"labelsSet\":true},\"ingress\":{\"backendServiceValid\":true,\"tlsSecretValid\":true}}}",
								},
							},
							Enabled: true,
						},
					},
				},
			},
			Platforms: &gqlModel.PlatformsCreateInput{
				SocketSlacks: []*gqlModel.SocketSlackCreateInput{
					{
						Name:     "slack",
						AppToken: "app token",
						BotToken: "bot token",
						Channels: []*gqlModel.ChannelBindingsByNameCreateInput{
							{
								Name: "foo",
								Bindings: &gqlModel.BotBindingsCreateInput{
									Sources:   []*string{ptr.FromType("kubernetes_config")},
									Executors: []*string{ptr.FromType("kubectl_config")},
								},
								NotificationsDisabled: ptr.FromType(true),
							},
							{
								Name: "bar",
								Bindings: &gqlModel.BotBindingsCreateInput{
									Sources:   []*string{ptr.FromType("kubernetes_config2")},
									Executors: []*string{},
								},
								NotificationsDisabled: ptr.FromType(true),
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	return &mutation.CreateDeployment
}

// CreateBasicDeploymentWithCloudSlack create deployment with Slack platform and three plugins.
func (c *Client) CreateBasicDeploymentWithCloudSlack(t *testing.T, clusterName, slackTeamID, channelName string) (*gqlModel.Deployment, error) {
	t.Helper()

	var mutation struct {
		CreateDeployment *gqlModel.Deployment `graphql:"createDeployment(input: $input)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"input": gqlModel.DeploymentCreateInput{
			Name: clusterName,
			Plugins: []*gqlModel.PluginsCreateInput{
				{
					Groups: []*gqlModel.PluginConfigurationGroupInput{
						{
							Name:        "botkube/kubernetes",
							DisplayName: "Kubernetes Info",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "kubernetes_config",
									Configuration: "{\"recommendations\":{\"pod\":{\"noLatestImageTag\":true,\"labelsSet\":true},\"ingress\":{\"backendServiceValid\":true,\"tlsSecretValid\":true}},\"namespaces\":{\"include\":[\"default\"],\"exclude\":[]},\"event\":{\"types\":[\"create\"]},\"resources\":[{\"type\":\"v1/pods\"},{\"type\":\"v1/services\"},{\"type\":\"networking.k8s.io/v1/ingresses\"},{\"type\":\"v1/nodes\"},{\"type\":\"v1/namespaces\"},{\"type\":\"v1/persistentvolumes\"},{\"type\":\"v1/persistentvolumeclaims\"},{\"type\":\"v1/configmaps\"},{\"type\":\"rbac.authorization.k8s.io/v1/roles\"},{\"type\":\"rbac.authorization.k8s.io/v1/rolebindings\"},{\"type\":\"rbac.authorization.k8s.io/v1/clusterrolebindings\"},{\"type\":\"rbac.authorization.k8s.io/v1/clusterroles\"},{\"type\":\"apps/v1/deployments\"},{\"type\":\"apps/v1/statefulsets\"},{\"type\":\"apps/v1/daemonsets\"},{\"type\":\"batch/v1/jobs\"}],\"commands\":{\"verbs\":[\"api-resources\",\"api-versions\",\"cluster-info\",\"describe\",\"explain\",\"get\",\"logs\",\"top\"],\"resources\":[\"deployments\",\"pods\",\"namespaces\",\"daemonsets\",\"statefulsets\",\"storageclasses\",\"nodes\",\"configmaps\",\"services\",\"ingresses\"]},\"filters\":{\"objectAnnotationChecker\":true,\"nodeEventsChecker\":true},\"informerResyncPeriod\":\"30m\",\"log\":{\"level\":\"info\",\"disableColors\":false}}",
								},
							},
							Enabled: true,
						},
						{
							Name:        "botkube/kubernetes",
							DisplayName: "Kubernetes Info2",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "kubernetes_config2",
									Configuration: "{\"recommendations\":{\"pod\":{\"noLatestImageTag\":true,\"labelsSet\":true},\"ingress\":{\"backendServiceValid\":true,\"tlsSecretValid\":true}}}",
								},
							},
							Enabled: true,
						},
						{
							Name:        "botkube/kubectl",
							DisplayName: "Kubectl",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "kubectl_config",
									Configuration: "{\"recommendations\":{\"pod\":{\"noLatestImageTag\":true,\"labelsSet\":true},\"ingress\":{\"backendServiceValid\":true,\"tlsSecretValid\":true}}}",
								},
							},
							Enabled: true,
						},
					},
				},
			},
			Platforms: &gqlModel.PlatformsCreateInput{
				CloudSlacks: []*gqlModel.CloudSlackCreateInput{
					{
						Name:   "Cloud Slack",
						TeamID: slackTeamID,
						Channels: []*gqlModel.ChannelBindingsByNameCreateInput{
							{
								Name: channelName,
								Bindings: &gqlModel.BotBindingsCreateInput{
									Sources:   []*string{ptr.FromType("kubernetes_config")},
									Executors: []*string{ptr.FromType("kubectl_config")},
								},
								NotificationsDisabled: nil,
							},
						},
					},
				},
			},
		},
	})
	return mutation.CreateDeployment, err
}

// MustCreateBasicDeploymentWithCloudSlack is like CreateBasicDeploymentWithCloudSlack but fails on error.
func (c *Client) MustCreateBasicDeploymentWithCloudSlack(t *testing.T, clusterName, slackTeamID, channelName string) *gqlModel.Deployment {
	t.Helper()
	deployment, err := c.CreateBasicDeploymentWithCloudSlack(t, clusterName, slackTeamID, channelName)
	require.NoError(t, err)
	return deployment
}

type (
	// Organization is a custom model that allow us to skip the 'connectedPlatforms.slack' field. Otherwise, we get such error:
	//
	//   Field "slack" argument "id" of type "ID!" is required, but it was not provided.
	Organization struct {
		ID                      string                                 `json:"id"`
		DisplayName             string                                 `json:"displayName"`
		Subscription            *gqlModel.OrganizationSubscription     `json:"subscription"`
		ConnectedPlatforms      *OrganizationConnectedPlatforms        `json:"connectedPlatforms"`
		OwnerID                 string                                 `json:"ownerId"`
		Owner                   *gqlModel.User                         `json:"owner"`
		Members                 []*gqlModel.User                       `json:"members"`
		Quota                   *gqlModel.Quota                        `json:"quota"`
		BillingHistoryAvailable bool                                   `json:"billingHistoryAvailable"`
		UpdateOperations        *gqlModel.OrganizationUpdateOperations `json:"updateOperations"`
		Usage                   *gqlModel.Usage                        `json:"usage"`
	}
	// Organizations holds organization collection.
	Organizations []Organization

	// OrganizationConnectedPlatforms skips the 'slack' field.
	OrganizationConnectedPlatforms struct {
		Slacks []*gqlModel.SlackWorkspace `json:"slacks"`
	}
)

// ToModel returns official gql model.
func (o Organization) ToModel() gqlModel.Organization {
	return gqlModel.Organization{
		ID:           o.ID,
		DisplayName:  o.DisplayName,
		Subscription: o.Subscription,
		ConnectedPlatforms: &gqlModel.OrganizationConnectedPlatforms{
			Slacks: o.ConnectedPlatforms.Slacks,
		},
		OwnerID:                 o.OwnerID,
		Owner:                   o.Owner,
		Members:                 o.Members,
		Quota:                   o.Quota,
		BillingHistoryAvailable: o.BillingHistoryAvailable,
		UpdateOperations:        o.UpdateOperations,
		Usage:                   o.Usage,
	}
}

// ToModel returns official gql model.
func (o Organizations) ToModel() []gqlModel.Organization {
	var out []gqlModel.Organization
	for _, item := range o {
		out = append(out, item.ToModel())
	}
	return out
}

// MustCreateOrganization creates organization.
func (c *Client) MustCreateOrganization(t *testing.T) gqlModel.Organization {
	t.Helper()

	var mutation struct {
		CreateOrganization Organization `graphql:"createOrganization(input: $input)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"input": gqlModel.OrganizationCreateInput{
			DisplayName: fmt.Sprintf("My %s organization:%s", t.Name(), uuid.NewString()),
		},
	})
	require.NoError(t, err)

	return mutation.CreateOrganization.ToModel()
}

// MustGetOrganization gets organization.
func (c *Client) MustGetOrganization(t *testing.T, id graphql.ID) gqlModel.Organization {
	t.Helper()

	var query struct {
		Organization Organization `graphql:"organization(id: $id)"`
	}

	err := c.Client.Query(context.Background(), &query, map[string]interface{}{
		"id": id,
	})
	require.NoError(t, err)

	return query.Organization.ToModel()
}

// MustAddMember adds member to organization.
func (c *Client) MustAddMember(t *testing.T, input gqlModel.AddMemberForOrganizationInput) gqlModel.Organization {
	t.Helper()

	var mutation struct {
		AddMember Organization `graphql:"addMemberForOrganization(input: $input)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"input": input,
	})
	require.NoError(t, err)

	return mutation.AddMember.ToModel()
}

// MustRemoveMember removes member from organization.
func (c *Client) MustRemoveMember(t *testing.T, input gqlModel.RemoveMemberFromOrganizationInput) gqlModel.Organization {
	t.Helper()

	var mutation struct {
		RemoveMember Organization `graphql:"removeMemberFromOrganization(input: $input)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"input": input,
	})
	require.NoError(t, err)

	return mutation.RemoveMember.ToModel()
}

// MustListAliases returns all aliases scoped to a given user.
func (c *Client) MustListAliases(t *testing.T) []*gqlModel.Alias {
	t.Helper()

	var page struct {
		Aliases gqlModel.AliasPage `graphql:"aliases(offset: $offset, limit: $limit)"`
	}

	err := c.Client.Query(context.Background(), &page, c.pagingVariables())
	require.NoError(t, err)
	return page.Aliases.Data
}

// MustGetDeployment returns a given deployment scoped to a given user.
func (c *Client) MustGetDeployment(t *testing.T, id graphql.ID) gqlModel.Deployment {
	t.Helper()

	var query struct {
		Deployment gqlModel.Deployment `graphql:"deployment(id: $id)"`
	}

	err := c.Client.Query(context.Background(), &query, map[string]interface{}{"id": id})
	require.NoError(t, err)
	return query.Deployment
}

// MustDeleteDeployment is like DeleteDeployment but panics on error.
func (c *Client) MustDeleteDeployment(t *testing.T, id graphql.ID) {
	err := c.DeleteDeployment(t, id)
	require.NoError(t, err)
}

// DeleteDeployment deletes a given deployment scoped to a given user.
func (c *Client) DeleteDeployment(t *testing.T, id graphql.ID) error {
	t.Helper()

	var mutation struct {
		Deployment bool `graphql:"deleteDeployment(id: $id)"`
	}

	return c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{"id": id})
}

// MustListDeployments returns all deployments scoped to a given user.
func (c *Client) MustListDeployments(t *testing.T) []*gqlModel.Deployment {
	t.Helper()

	var page struct {
		Deployments gqlModel.DeploymentPage `graphql:"deployments(offset: $offset, limit: $limit)"`
	}

	err := c.Client.Query(context.Background(), &page, c.pagingVariables())
	require.NoError(t, err)
	return page.Deployments.Data
}

// MustListAudits returns all audits scoped to a given user.
func (c *Client) MustListAudits(t *testing.T) []gqlModel.AuditEvent {
	t.Helper()

	var page struct {
		Audits gqlModel.AuditEventPage `graphql:"auditEvents(offset: $offset, limit: $limit)"`
	}

	err := c.Client.Query(context.Background(), &page, c.pagingVariables())
	require.NoError(t, err)
	return page.Audits.Data
}

// MustReportDeploymentHeartbeat sends heartbeat info.
func (c *Client) MustReportDeploymentHeartbeat(t *testing.T, deploymentId string, nodeCount int) bool {
	t.Helper()

	var mutation struct {
		ReportDeploymentHeartbeat bool `graphql:"reportDeploymentHeartbeat(id: $id, in: $in)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"id": graphql.ID(deploymentId),
		"in": gqlModel.DeploymentHeartbeatInput{
			NodeCount: nodeCount,
		},
	})
	require.NoError(t, err)

	return mutation.ReportDeploymentHeartbeat
}

// MustReportDeploymentStartup sends startup report.
func (c *Client) MustReportDeploymentStartup(t *testing.T, deploymentId string) bool {
	t.Helper()

	var mutation struct {
		ReportDeploymentStartup bool `graphql:"reportDeploymentStartup(id: $id, resourceVersion: $in)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"id": graphql.ID(deploymentId),
		"in": 1,
	})
	require.NoError(t, err)

	return mutation.ReportDeploymentStartup
}

// MustDeleteSlackWorkspace deletes a slack workspace.
func (c *Client) MustDeleteSlackWorkspace(t *testing.T, orgID, slackWorkspaceID string) {
	t.Helper()

	type Identifiable struct {
		ID string `graphql:"id"`
	}

	var mutation struct {
		RemovePlatformFromOrganization Identifiable `graphql:"removePlatformFromOrganization(input: $input)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"input": gqlModel.RemovePlatformFromOrganizationInput{
			OrganizationID: orgID,
			Slack: &gqlModel.RemoveSlackFromOrganizationInput{
				ID: slackWorkspaceID,
			},
		},
	})
	require.NoError(t, err)
}

// MustListSlackWorkspacesForOrg returns all slack workspaces scoped to a given organization.
func (c *Client) MustListSlackWorkspacesForOrg(t *testing.T, orgID string) []*gqlModel.SlackWorkspace {
	t.Helper()

	var query struct {
		Organization Organization `graphql:"organization(id: $id)"`
	}

	err := c.Client.Query(context.Background(), &query, map[string]interface{}{
		"id": graphql.ID(orgID),
	})
	require.NoError(t, err)

	require.NotNil(t, query.Organization.ConnectedPlatforms)
	require.NotEmpty(t, query.Organization.ConnectedPlatforms.Slacks)
	return query.Organization.ConnectedPlatforms.Slacks
}

// NewClientForOrganization returns new GraphQL client with organization header.
func (c *Client) NewClientForOrganization(id string) *Client {
	return &Client{
		Client: c.Client.WithRequestModifier(func(request *http.Request) {
			request.Header.Set(botkubeOrganizationHeaderName, id)
		}),
	}
}

// NewClientForAuthAndOrg returns new GraphQL client with organization and authorization headers.
func NewClientForAuthAndOrg(apiEndpoint, orgID, authValue string) *Client {
	gqLCli := graphql.NewClient(apiEndpoint, nil)

	return &Client{
		Client: gqLCli.WithRequestModifier(func(request *http.Request) {
			request.Header.Set(botkubeOrganizationHeaderName, orgID)
			request.Header.Set(botkubeAuthorizationHeaderName, authValue)
		}),
	}
}

// NewClientForAPIKey returns new GraphQL client with API Key header.
func (c *Client) NewClientForAPIKey(key string) *Client {
	return &Client{
		Client: c.Client.WithRequestModifier(func(request *http.Request) {
			request.Header.Set(botkubeAPIKeyHeaderName, key)
		}),
	}
}

func (c *Client) pagingVariables() map[string]interface{} {
	return map[string]interface{}{
		"offset": 0,
		"limit":  100,
	}
}
