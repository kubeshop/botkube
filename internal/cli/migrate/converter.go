package migrate

import (
	"encoding/json"
	"fmt"
	"strings"

	bkconfig "github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"

	gqlModel "github.com/kubeshop/botkube/internal/graphql"
	"github.com/kubeshop/botkube/pkg/ptr"
)

// Converter converts OS config into GraphQL create input.
type Converter struct{}

// NewConverter returns a new Converter instance.
func NewConverter() *Converter {
	return &Converter{}
}

// ConvertActions converts Actions.
func (c *Converter) ConvertActions(actions bkconfig.Actions) []*gqlModel.ActionCreateUpdateInput {
	var out []*gqlModel.ActionCreateUpdateInput
	for name, act := range actions {
		out = append(out, &gqlModel.ActionCreateUpdateInput{
			Name:        name,
			DisplayName: act.DisplayName,
			Enabled:     act.Enabled,
			Command:     act.Command,
			Bindings: &gqlModel.ActionCreateUpdateInputBindings{
				Sources:   act.Bindings.Sources,
				Executors: act.Bindings.Executors,
			},
		})
	}
	return out
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
						Name:          cfgName,
						Configuration: string(rawCfg),
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
						Name:          cfgName,
						Configuration: string(rawCfg),
					},
				},
			})
		}
	}

	return out, errs.ErrorOrNil()
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
				Sources:   toSlicePointers(ch.Bindings.Sources),
				Executors: toSlicePointers(ch.Bindings.Executors),
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
				Sources:   toSlicePointers(ch.Bindings.Sources),
				Executors: toSlicePointers(ch.Bindings.Executors),
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
				Sources:   toSlicePointers(ch.Bindings.Sources),
				Executors: toSlicePointers(ch.Bindings.Executors),
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

func toSlicePointers[T any](in []T) []*T {
	var out []*T
	for idx := range in {
		out = append(out, &in[idx])
	}
	return out
}
