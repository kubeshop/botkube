package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/hasura/go-graphql-client"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/ptr"
	gqlModel "github.com/kubeshop/botkube/internal/remote/graphql"
)

const (
	//nolint:gosec // G101: Potential hardcoded credentials
	botkubeAPIKeyHeaderName = "X-API-Key"
)

// Client provides helper functions for queries and mutations that  across different test cases.
// It simplifies setting up a given test prerequisites that are not a part of the test itself.
type Client struct {
	*graphql.Client
}

// CreateBasicDeploymentWithCloudSlack create deployment with Slack platform and three plugins.
func (c *Client) CreateBasicDeploymentWithCloudSlack(t *testing.T, clusterName, slackTeamID, firstChannel, secondChannel, thirdChannel string) (*gqlModel.Deployment, error) {
	t.Helper()

	var mutation struct {
		CreateDeployment *gqlModel.Deployment `graphql:"createDeployment(input: $input)"`
	}

	rbac := gqlModel.RBACInput{
		User: &gqlModel.UserPolicySubjectInput{
			Type:   gqlModel.PolicySubjectTypeStatic,
			Static: &gqlModel.UserStaticSubjectInput{Value: "botkube-plugins-default"},
		},
		Group: &gqlModel.GroupPolicySubjectInput{
			Type:   gqlModel.PolicySubjectTypeStatic,
			Static: &gqlModel.GroupStaticSubjectInput{Values: []string{"botkube-plugins-default"}},
		},
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"input": gqlModel.DeploymentCreateInput{
			Name: clusterName,
			Plugins: []*gqlModel.PluginsCreateInput{
				{
					Groups: []*gqlModel.PluginConfigurationGroupInput{
						{
							Name:        "botkube/kubernetes",
							DisplayName: "K8s recommendations",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "k8s-events",
									Configuration: "{\"log\":{\"level\":\"debug\"},\"recommendations\":{\"pod\":{\"noLatestImageTag\":true,\"labelsSet\":true},\"ingress\":{\"backendServiceValid\":false,\"tlsSecretValid\":false}},\"namespaces\":{\"include\":[\"botkube\"]},\"event\":{\"types\":[\"create\",\"update\"]},\"resources\":[{\"type\":\"v1/configmaps\",\"updateSetting\":{\"includeDiff\":false,\"fields\":[\"data\"]}}]}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/kubernetes",
							DisplayName: "K8s ConfigMap delete events",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "k8s-annotated-cm-delete",
									Configuration: "{\"log\":{\"level\":\"debug\"},\"namespaces\":{\"include\":[\"botkube\"]},\"labels\":{\"test.botkube.io\":\"true\"},\"event\":{\"types\":[\"delete\"]},\"resources\":[{\"type\":\"v1/configmaps\"}]}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/kubernetes",
							DisplayName: "Pod Create Events",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "k8s-pod-create-events",
									Configuration: "{\"log\":{\"level\":\"debug\"},\"namespaces\":{\"include\":[\"botkube\"]},\"event\":{\"types\":[\"create\"]},\"resources\":[{\"type\":\"v1/pods\"}]}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/kubernetes",
							DisplayName: "K8s Service creation, used only by action",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "k8s-service-create-event-for-action-only",
									Configuration: "{\"namespaces\":{\"include\":[\"botkube\"]},\"event\":{\"types\":[\"create\"]},\"resources\":[{\"type\":\"v1/services\"}]}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/kubernetes",
							DisplayName: "K8s ConfigMaps updates",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "k8s-updates",
									Configuration: "{\"log\":{\"level\":\"debug\"},\"namespaces\":{\"include\":[\"default\"]},\"event\":{\"types\":[\"create\",\"update\",\"delete\"]},\"resources\":[{\"type\":\"v1/configmaps\",\"namespaces\":{\"include\":[\"botkube\"]},\"event\":{\"types\":[\"update\"]},\"updateSetting\":{\"includeDiff\":false,\"fields\":[\"data\"]}}]}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/kubernetes",
							DisplayName: "K8s ConfigMaps updates",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "rbac-with-static-mapping",
									Configuration: "{\"namespaces\":{\"include\":[\"botkube\"]},\"annotations\":{\"rbac.botkube.io\":\"true\"},\"event\":{\"types\":[\"create\"]},\"resources\":[{\"type\":\"v1/configmaps\"}]}",
									Rbac: &gqlModel.RBACInput{
										User: &gqlModel.UserPolicySubjectInput{
											Type:   gqlModel.PolicySubjectTypeStatic,
											Static: &gqlModel.UserStaticSubjectInput{Value: "kc-watch-cm"},
											Prefix: ptr.FromType[string](""),
										},
										Group: &gqlModel.GroupPolicySubjectInput{
											Type:   gqlModel.PolicySubjectTypeStatic,
											Static: &gqlModel.GroupStaticSubjectInput{Values: []string{"kc-watch-cm"}},
											Prefix: ptr.FromType[string](""),
										},
									},
								},
							},
						},
						{
							Name:        "botkube/kubernetes",
							DisplayName: "Kubernetes Resource Created Events",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "k8s-create-events",
									Configuration: "{\"namespaces\":{\"include\":[\"*\"],\"event\":{\"types\":[\"create\"]},\"resources\":[{\"type\":\"v1/pods\"},{\"type\":\"v1/services\"},{\"type\":\"networking.k8s.io/v1/ingresses\"},{\"type\":\"v1/nodes\"},{\"type\":\"v1/namespaces\"},{\"type\":\"v1/configmaps\"},{\"type\":\"apps/v1/deployments\"},{\"type\":\"apps/v1/statefulsets\"},{\"type\":\"apps/v1/daemonsets\"},{\"type\":\"batch/v1/jobs\"}]}}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/kubernetes",
							DisplayName: "Kubernetes Errors for resources with logs",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "k8s-err-with-logs-events",
									Configuration: "{\"namespaces\":{\"include\":[\"*\"],\"event\":{\"types\":[\"error\"]},\"resources\":[{\"type\":\"v1/pods\"},{\"type\":\"apps/v1/deployments\"},{\"type\":\"apps/v1/statefulsets\"},{\"type\":\"apps/v1/daemonsets\"},{\"type\":\"batch/v1/jobs\"}]}}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/cm-watcher",
							DisplayName: "K8s ConfigMaps changes",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "other-plugins",
									Configuration: "{\"configMap\":{\"name\":\"cm-watcher-trigger\",\"namespace\":\"botkube\",\"event\":\"ADDED\"}}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/cm-watcher",
							DisplayName: "CM watcher RBAC",
							Type:        gqlModel.PluginTypeSource,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "rbac-with-default-configuration",
									Configuration: "{\"configMap\":{\"name\":\"cm-rbac\",\"namespace\":\"botkube\",\"event\":\"DELETED\"}}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/kubectl",
							DisplayName: "Default Tools",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "k8s-default-tools",
									Configuration: "{}",
								},
							},
						},
						{
							Name:        "botkube/kubectl",
							DisplayName: "First channel",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "kubectl-first-channel-cmd",
									Configuration: "{}",
									Rbac: &gqlModel.RBACInput{
										User: &gqlModel.UserPolicySubjectInput{
											Type:   gqlModel.PolicySubjectTypeStatic,
											Static: &gqlModel.UserStaticSubjectInput{Value: "kubectl-first-channel"},
										},
										Group: &gqlModel.GroupPolicySubjectInput{
											Type:   gqlModel.PolicySubjectTypeStatic,
											Static: &gqlModel.GroupStaticSubjectInput{Values: []string{}},
										}},
								},
							},
						},
						{
							Name:        "botkube/kubectl",
							DisplayName: "Not bounded",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "kubectl-not-bound-to-any-channel",
									Configuration: "{}",
									Rbac: &gqlModel.RBACInput{
										User: &gqlModel.UserPolicySubjectInput{
											Type:   gqlModel.PolicySubjectTypeStatic,
											Static: &gqlModel.UserStaticSubjectInput{Value: "kubectl-first-channel"},
										},
										Group: &gqlModel.GroupPolicySubjectInput{
											Type:   gqlModel.PolicySubjectTypeStatic,
											Static: &gqlModel.GroupStaticSubjectInput{Values: []string{}},
										},
									},
								},
							},
						},
						{
							Name:        "botkube/kubectl",
							DisplayName: "Service label perms",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "kubectl-with-svc-label-perms",
									Configuration: "{}",
									Rbac: &gqlModel.RBACInput{
										User: &gqlModel.UserPolicySubjectInput{
											Type:   gqlModel.PolicySubjectTypeStatic,
											Static: &gqlModel.UserStaticSubjectInput{Value: "kc-label-svc-all"},
										},
										Group: &gqlModel.GroupPolicySubjectInput{
											Type:   gqlModel.PolicySubjectTypeStatic,
											Static: &gqlModel.GroupStaticSubjectInput{Values: []string{}},
										},
									},
								},
							},
						},
						{
							Name:        "botkube/kubectl",
							DisplayName: "Rbac Channel Mapping",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "rbac-with-channel-mapping",
									Configuration: "{\"defaultNamespace\":\"botkube\"}",
									Rbac: &gqlModel.RBACInput{
										Group: &gqlModel.GroupPolicySubjectInput{
											Type:   gqlModel.PolicySubjectTypeChannelName,
											Static: &gqlModel.GroupStaticSubjectInput{Values: []string{""}},
										},
									},
								},
							},
						},
						{
							Name:        "botkube/helm",
							DisplayName: "Helm",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "helm",
									Configuration: "{}",
									Rbac:          &rbac,
								},
							},
						},
						{
							Name:        "botkube/echo@v0.0.0-latest",
							DisplayName: "Echo",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "other-plugins",
									Configuration: "{\"changeResponseToUpperCase\":true}",
								},
							},
						},
						{
							Name:        "botkube/echo@v0.0.0-latest",
							DisplayName: "Echo with no RBAC",
							Type:        gqlModel.PluginTypeExecutor,
							Configurations: []*gqlModel.PluginConfigurationInput{
								{
									Name:          "rbac-with-no-configuration",
									Configuration: "{\"changeResponseToUpperCase\":true}",
								},
							},
						},
					},
				},
			},
			Actions: []*gqlModel.ActionCreateUpdateInput{
				{
					Name:        "get-created-resource",
					DisplayName: "Get created resource",
					Enabled:     true,
					Command:     "kubectl get {{ .Event.Kind | lower }}{{ if .Event.Namespace }} -n {{ .Event.Namespace }}{{ end }} {{ .Event.Name }}",
					Bindings: &gqlModel.ActionCreateUpdateInputBindings{
						Sources:   []string{"k8s-pod-create-events"},
						Executors: []string{"k8s-default-tools"},
					},
				},
				{
					Name:        "label-created-svc-resource",
					DisplayName: "Label created Service",
					Enabled:     true,
					Command:     "kubectl label svc {{ if .Event.Namespace }} -n {{ .Event.Namespace }}{{ end }} {{ .Event.Name }} botkube-action=true",
					Bindings: &gqlModel.ActionCreateUpdateInputBindings{
						Sources:   []string{"k8s-service-create-event-for-action-only"},
						Executors: []string{"kubectl-with-svc-label-perms"},
					},
				},
				{
					Name:        "describe-created-resource",
					DisplayName: "Describe created resource",
					Enabled:     false,
					Command:     "kubectl describe {{ .Event.Kind | lower }}{{ if .Event.Namespace }} -n {{ .Event.Namespace }}{{ end }} {{ .Event.Name }}",
					Bindings: &gqlModel.ActionCreateUpdateInputBindings{
						Sources:   []string{"k8s-create-events"},
						Executors: []string{"k8s-default-tools"},
					},
				},
				{
					Name:        "show-logs-on-error",
					DisplayName: "Show logs on error",
					Enabled:     false,
					Command:     "kubectl logs {{ .Event.Kind | lower }}/{{ .Event.Name }} -n {{ .Event.Namespace }}",
					Bindings: &gqlModel.ActionCreateUpdateInputBindings{
						Sources:   []string{"k8s-err-with-logs-events"},
						Executors: []string{"k8s-default-tools"},
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
								Name: firstChannel,
								Bindings: &gqlModel.BotBindingsCreateInput{
									Sources:   []*string{ptr.FromType("k8s-events"), ptr.FromType("k8s-annotated-cm-delete"), ptr.FromType("k8s-pod-create-events"), ptr.FromType("other-plugins")},
									Executors: []*string{ptr.FromType("kubectl-first-channel-cmd"), ptr.FromType("other-plugins"), ptr.FromType("helm")},
								},
								NotificationsDisabled: ptr.FromType[bool](false),
							},
							{
								Name: secondChannel,
								Bindings: &gqlModel.BotBindingsCreateInput{
									Sources:   []*string{ptr.FromType("k8s-updates")},
									Executors: []*string{ptr.FromType("k8s-default-tools")},
								},
								NotificationsDisabled: ptr.FromType[bool](true),
							},
							{
								Name: thirdChannel,
								Bindings: &gqlModel.BotBindingsCreateInput{
									Sources:   []*string{ptr.FromType("rbac-with-static-mapping"), ptr.FromType("rbac-with-default-configuration")},
									Executors: []*string{ptr.FromType("rbac-with-channel-mapping"), ptr.FromType("rbac-with-no-configuration")},
								},
								NotificationsDisabled: ptr.FromType[bool](false),
							},
						},
					},
				},
			},
			AttachDefaultAliases: ptr.FromType[bool](true),
		},
	})
	return mutation.CreateDeployment, err
}

// MustCreateBasicDeploymentWithCloudSlack is like CreateBasicDeploymentWithCloudSlack but fails on error.
func (c *Client) MustCreateBasicDeploymentWithCloudSlack(t *testing.T, clusterName, slackTeamID, firstChannel, secondChannel, thirdChannel string) *gqlModel.Deployment {
	t.Helper()
	deployment, err := c.CreateBasicDeploymentWithCloudSlack(t, clusterName, slackTeamID, firstChannel, secondChannel, thirdChannel)
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

// MustCreateAlias creates alias.
func (c *Client) MustCreateAlias(t *testing.T, name, displayName, command, deploymentId string) gqlModel.Alias {
	t.Helper()

	var mutation struct {
		CreateAlias gqlModel.Alias `graphql:"createAlias(input: $input)"`
	}

	err := c.Client.Mutate(context.Background(), &mutation, map[string]interface{}{
		"input": gqlModel.AliasCreateInput{
			Name:          name,
			DisplayName:   displayName,
			Command:       command,
			DeploymentIds: []string{deploymentId},
		},
	})
	require.NoError(t, err)

	return mutation.CreateAlias
}

// NewClientForAPIKey returns new GraphQL client with API Key header.
func NewClientForAPIKey(apiEndpoint, key string) *Client {
	gqLCli := graphql.NewClient(apiEndpoint, nil)

	return &Client{
		Client: gqLCli.WithRequestModifier(func(request *http.Request) {
			request.Header.Set(botkubeAPIKeyHeaderName, key)
		}),
	}
}
