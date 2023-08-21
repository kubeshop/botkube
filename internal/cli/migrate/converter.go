package migrate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xyproto/randomstring"

	"github.com/kubeshop/botkube/internal/ptr"
	gqlModel "github.com/kubeshop/botkube/internal/remote/graphql"
	bkconfig "github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// Converter converts OS config into GraphQL create input.
type Converter struct {
	pluginNames map[string]string
}

// NewConverter returns a new Converter instance.
func NewConverter() *Converter {
	return &Converter{
		pluginNames: map[string]string{},
	}
}

// ConvertActions converts Actions.
func (c *Converter) ConvertActions(actions bkconfig.Actions, sources map[string]bkconfig.Sources, executors map[string]bkconfig.Executors) []*gqlModel.ActionCreateUpdateInput {
	var out []*gqlModel.ActionCreateUpdateInput
	for name, act := range actions {
		bindings, ok := c.checkActionBindingExists(act, sources, executors)
		if !ok {
			continue
		}
		out = append(out, &gqlModel.ActionCreateUpdateInput{
			Name:        name,
			DisplayName: act.DisplayName,
			Enabled:     act.Enabled,
			Command:     act.Command,
			Bindings:    bindings,
		})
	}
	return out
}

func (c *Converter) checkActionBindingExists(act bkconfig.Action, sources map[string]bkconfig.Sources, executors map[string]bkconfig.Executors) (*gqlModel.ActionCreateUpdateInputBindings, bool) {
	sourcesGenerated := make([]string, 0, len(act.Bindings.Sources))
	for _, source := range act.Bindings.Sources {
		if _, ok := sources[source]; !ok {
			return nil, false
		}
		name := c.getOrGeneratePluginName(source)
		sourcesGenerated = append(sourcesGenerated, name)
	}
	executorsGenerated := make([]string, 0, len(act.Bindings.Executors))
	for _, executor := range act.Bindings.Executors {
		if _, ok := executors[executor]; !ok {
			return nil, false
		}
		name := c.getOrGeneratePluginName(executor)
		executorsGenerated = append(executorsGenerated, name)
	}

	return &gqlModel.ActionCreateUpdateInputBindings{
		Sources:   sourcesGenerated,
		Executors: executorsGenerated,
	}, true
}

// ConvertAliases converts Aliases.
func (c *Converter) ConvertAliases(aliases bkconfig.Aliases, instanceID string) []*gqlModel.AliasCreateInput {
	var out []*gqlModel.AliasCreateInput
	for name, alias := range aliases {
		out = append(out, &gqlModel.AliasCreateInput{
			Name:          name,
			DisplayName:   alias.DisplayName,
			Command:       alias.Command,
			DeploymentIds: []string{instanceID},
		})
	}
	return out
}

// ConvertPlugins converts all plugins.
func (c *Converter) ConvertPlugins(exec map[string]bkconfig.Executors, sources map[string]bkconfig.Sources) ([]*gqlModel.PluginsCreateInput, error) {
	createSources, err := c.convertSources(sources)
	if err != nil {
		return nil, err
	}
	createExecutors, err := c.convertExecutors(exec)
	if err != nil {
		return nil, nil
	}

	return []*gqlModel.PluginsCreateInput{
		{
			Groups: append(createSources, createExecutors...),
		},
	}, nil
}

func (c *Converter) convertExecutors(executors map[string]bkconfig.Executors) ([]*gqlModel.PluginConfigurationGroupInput, error) {
	var out []*gqlModel.PluginConfigurationGroupInput

	errs := multierror.New()
	for cfgName, conf := range executors {
		for name, p := range conf.Plugins {
			if !p.Enabled || !strings.HasPrefix(name, "botkube") { // skip all 3rd party plugins
				continue
			}

			rawCfg, err := json.Marshal(p.Config)
			if err != nil {
				return nil, err
			}
			out = append(out, &gqlModel.PluginConfigurationGroupInput{
				Name:        name,
				DisplayName: name,
				Type:        gqlModel.PluginTypeExecutor,
				Configurations: []*gqlModel.PluginConfigurationInput{
					{
						Name:          c.getOrGeneratePluginName(cfgName),
						Configuration: string(rawCfg),
						Rbac:          c.convertRbac(p.Context),
					},
				},
			})
		}
	}

	return out, errs.ErrorOrNil()
}

func (c *Converter) convertSources(sources map[string]bkconfig.Sources) ([]*gqlModel.PluginConfigurationGroupInput, error) {
	var out []*gqlModel.PluginConfigurationGroupInput

	errs := multierror.New()
	for cfgName, conf := range sources {
		for name, p := range conf.Plugins {
			if !p.Enabled || !strings.HasPrefix(name, "botkube") { // skip all 3rd party plugins
				continue
			}
			rawCfg, err := json.Marshal(p.Config)
			if err != nil {
				return nil, err
			}
			out = append(out, &gqlModel.PluginConfigurationGroupInput{
				Name:        name,
				DisplayName: conf.DisplayName,
				Type:        gqlModel.PluginTypeSource,
				Configurations: []*gqlModel.PluginConfigurationInput{
					{
						Name:          c.getOrGeneratePluginName(cfgName),
						Configuration: string(rawCfg),
						Rbac:          c.convertRbac(p.Context),
					},
				},
			})
		}
	}

	return out, errs.ErrorOrNil()
}

func (c *Converter) convertRbac(ctx bkconfig.PluginContext) *gqlModel.RBACInput {
	return &gqlModel.RBACInput{
		User: &gqlModel.UserPolicySubjectInput{
			Type:   graphqlPolicySubjectType(ctx.RBAC.User.Type),
			Static: &gqlModel.UserStaticSubjectInput{Value: ctx.RBAC.User.Static.Value},
			Prefix: &ctx.RBAC.User.Prefix,
		},
		Group: &gqlModel.GroupPolicySubjectInput{
			Type:   graphqlPolicySubjectType(ctx.RBAC.Group.Type),
			Static: &gqlModel.GroupStaticSubjectInput{Values: ctx.RBAC.Group.Static.Values},
			Prefix: &ctx.RBAC.Group.Prefix,
		},
	}
}

// ConvertPlatforms converts cloud supported platforms.
func (c *Converter) ConvertPlatforms(platforms map[string]bkconfig.Communications) *gqlModel.PlatformsCreateInput {
	out := gqlModel.PlatformsCreateInput{}

	for name, comm := range platforms {
		if comm.SocketSlack.Enabled {
			out.SocketSlacks = append(out.SocketSlacks, c.convertSlackPlatform(name, comm.SocketSlack))
		}

		if comm.Mattermost.Enabled {
			out.Mattermosts = append(out.Mattermosts, c.convertMattermostPlatform(name, comm.Mattermost))
		}

		if comm.Discord.Enabled {
			out.Discords = append(out.Discords, c.convertDiscordPlatform(name, comm.Discord))
		}
	}
	return &out
}

func (c *Converter) convertSlackPlatform(name string, slack bkconfig.SocketSlack) *gqlModel.SocketSlackCreateInput {
	var channels []*gqlModel.ChannelBindingsByNameCreateInput
	for _, ch := range slack.Channels {
		channels = append(channels, &gqlModel.ChannelBindingsByNameCreateInput{
			Name: ch.Name,
			Bindings: &gqlModel.BotBindingsCreateInput{
				Sources:   c.toGeneratedNamesSlice(ch.Bindings.Sources),
				Executors: c.toGeneratedNamesSlice(ch.Bindings.Executors),
			},
			NotificationsDisabled: ptr.FromType(ch.Notification.Disabled),
		})
	}

	return &gqlModel.SocketSlackCreateInput{
		Name:     fmt.Sprintf("Slack %s", strings.ToLower(name)),
		AppToken: slack.AppToken,
		BotToken: slack.BotToken,
		Channels: channels,
	}
}

func (c *Converter) convertDiscordPlatform(name string, discord bkconfig.Discord) *gqlModel.DiscordCreateInput {
	var channels []*gqlModel.ChannelBindingsByIDCreateInput
	for _, ch := range discord.Channels {
		channels = append(channels, &gqlModel.ChannelBindingsByIDCreateInput{
			ID: ch.ID,
			Bindings: &gqlModel.BotBindingsCreateInput{
				Sources:   c.toGeneratedNamesSlice(ch.Bindings.Sources),
				Executors: c.toGeneratedNamesSlice(ch.Bindings.Executors),
			},
			NotificationsDisabled: ptr.FromType(ch.Notification.Disabled),
		})
	}

	return &gqlModel.DiscordCreateInput{
		Name:     fmt.Sprintf("Discord %s", strings.ToLower(name)),
		Token:    discord.Token,
		BotID:    discord.BotID,
		Channels: channels,
	}
}

func (c *Converter) convertMattermostPlatform(name string, matt bkconfig.Mattermost) *gqlModel.MattermostCreateInput {
	var channels []*gqlModel.ChannelBindingsByNameCreateInput
	for _, ch := range matt.Channels {
		channels = append(channels, &gqlModel.ChannelBindingsByNameCreateInput{
			Name: ch.Name,
			Bindings: &gqlModel.BotBindingsCreateInput{
				Sources:   c.toGeneratedNamesSlice(ch.Bindings.Sources),
				Executors: c.toGeneratedNamesSlice(ch.Bindings.Executors),
			},
			NotificationsDisabled: ptr.FromType(ch.Notification.Disabled),
		})
	}

	return &gqlModel.MattermostCreateInput{
		Name:     fmt.Sprintf("Mattermost %s", strings.ToLower(name)),
		BotName:  matt.BotName,
		URL:      matt.URL,
		Token:    matt.Token,
		Team:     matt.Team,
		Channels: channels,
	}
}

func (c *Converter) getOrGeneratePluginName(plugin string) string {
	if name, ok := c.pluginNames[plugin]; ok {
		return name
	}
	name := fmt.Sprintf("%s_%s", plugin, randomstring.CookieFriendlyString(5))
	c.pluginNames[plugin] = name
	return name
}

func (c *Converter) toGeneratedNamesSlice(in []string) []*string {
	out := make([]*string, 0, len(in))
	for _, name := range in {
		generated := c.getOrGeneratePluginName(name)
		out = append(out, &generated)
	}
	return out
}

func graphqlPolicySubjectType(sub bkconfig.PolicySubjectType) gqlModel.PolicySubjectType {
	switch sub {
	case bkconfig.StaticPolicySubjectType:
		return gqlModel.PolicySubjectTypeStatic
	case bkconfig.ChannelNamePolicySubjectType:
		return gqlModel.PolicySubjectTypeChannelName
	default:
		return gqlModel.PolicySubjectTypeEmpty
	}
}
