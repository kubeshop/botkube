package interactive

import (
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
)

// RunCommandName defines the button name for the run commands.
const RunCommandName = "Run command"

// HelpMessage provides an option to build the Help message depending on a given platform.
type HelpMessage struct {
	btnBuilder             *api.ButtonBuilder
	platform               config.CommPlatformIntegration
	clusterName            string
	enabledPluginExecutors []string
}

// NewHelpMessage return a new instance of HelpMessage.
func NewHelpMessage(platform config.CommPlatformIntegration, clusterName string, executors []string) *HelpMessage {
	return &HelpMessage{
		btnBuilder:             api.NewMessageButtonBuilder(),
		platform:               platform,
		clusterName:            clusterName,
		enabledPluginExecutors: executors,
	}
}

// Build returns help message with interactive sections.
func (h *HelpMessage) Build() CoreMessage {
	msg := CoreMessage{
		Header: fmt.Sprintf(":rocket: Botkube instance %q is now active.", h.clusterName),
	}

	type getter func() []api.Section
	var sections = []getter{
		h.cluster,
		h.ping,
		h.notificationSections,
		h.actionSections,
		h.executorSections,
		h.pluginHelpSections,
		h.footer,
	}
	for _, add := range sections {
		msg.Sections = append(msg.Sections, add()...)
	}

	return msg
}

func (h *HelpMessage) cluster() []api.Section {
	switch h.platform {
	case config.SlackCommPlatformIntegration, config.DiscordCommPlatformIntegration, config.MattermostCommPlatformIntegration:
		return []api.Section{
			{
				Base: api.Base{
					Header:      "Multi-cluster mode",
					Description: "If you have multiple clusters configured for this channel, specify the cluster name when typing commands.",
					Body: api.Body{
						CodeBlock: fmt.Sprintf("--cluster-name=%s\n", h.clusterName),
					},
				},
			},
		}
	case config.CloudSlackCommPlatformIntegration:
		return []api.Section{
			{
				BulletLists: []api.BulletList{
					{
						Title: "Multi-Cluster",
						Items: []string{
							fmt.Sprintf("Specify `--cluster-name=%s` flag to run a command on this cluster.", h.clusterName),
							"Use `--all-clusters` flag to run commands on all clusters.",
							"Use Cloud commands to manage connected instances and set per-channel defaults.",
						},
					},
				},
				Buttons: []api.Button{
					h.btnBuilder.ForCommandWithDescCmd("List Cloud commands", "cloud help"),
				},
			},
		}
	default:
		return nil
	}
}

func (h *HelpMessage) ping() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Ping your cluster to check its status",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithDescCmd("Ping cluster", "ping"),
			},
		},
	}
}

func (h *HelpMessage) footer() []api.Section {
	return []api.Section{
		{
			Buttons: []api.Button{
				h.btnBuilder.ForURL("Give feedback", "https://feedback.botkube.io", api.ButtonStylePrimary),
				h.btnBuilder.ForURL("Read our docs", "https://docs.botkube.io"),
				h.btnBuilder.ForURL("Join our Slack", "https://join.botkube.io"),
				h.btnBuilder.ForURL("Follow us on Twitter", "https://twitter.com/botkube_io"),
			},
		},
	}
}

func (h *HelpMessage) notificationSections() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Manage incoming notifications",
				Body: api.Body{
					CodeBlock: fmt.Sprintf("%s [enable|disable|status] notifications", api.MessageBotNamePlaceholder),
				},
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommand("Enable", "enable notifications"),
				h.btnBuilder.ForCommand("Disable", "disable notifications"),
				h.btnBuilder.ForCommand("Get status", "status notifications"),
			},
		},
		{
			Base: api.Base{
				Header: "Fine-tune your notifications for this channel",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithDescCmd("Adjust notifications", "edit SourceBindings"),
			},
		},
	}
}

func (h *HelpMessage) actionSections() []api.Section {
	return []api.Section{
		{
			Buttons: []api.Button{
				h.btnBuilder.ForURLWithBoldDesc("Automation help", "Automatically execute commands upon receiving events", "https://docs.botkube.io/usage/automated-actions"),
			},
		},
	}
}

func (h *HelpMessage) executorSections() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Manage executors and aliases",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommand("List executors", "list executors"),
				h.btnBuilder.ForCommand("List aliases", "list aliases"),
				h.btnBuilder.ForURL("Executors and aliases help", "https://docs.botkube.io/usage/executor"),
			},
		},
	}
}

func (h *HelpMessage) pluginHelpSections() []api.Section {
	var out []api.Section

	slices.Sort(h.enabledPluginExecutors) // to make the order predictable for testing

	for _, name := range h.enabledPluginExecutors {
		helpFn, found := pluginHelpProvider[name]
		if !found {
			continue
		}

		helpSection := helpFn(h.platform, h.btnBuilder)
		out = append(out, helpSection)
	}
	return out
}
