package interactive

import (
	"fmt"
	"os"

	"golang.org/x/exp/slices"

	"github.com/kubeshop/botkube/internal/config/remote"
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
//
// You can see how the help message looks like without starting the Agent - navigate to `test/msg-layouts/help_test.go`.
func (h *HelpMessage) Build(init bool) CoreMessage {
	msg := CoreMessage{}

	if init {
		msg.Header = fmt.Sprintf("🚀 Botkube instance %q is now active.", h.clusterName)
	}

	type getter func() []api.Section
	var sections = []getter{
		h.botkubeCloud,
		h.aiPlugin,
		h.basicCommands,
		h.notificationSections,
		h.pluginHelpSections,
		h.cluster,
		h.advancedFeatures,
		h.footer,
	}
	for _, add := range sections {
		msg.Sections = append(msg.Sections, add()...)
	}

	return msg
}

func (h *HelpMessage) cluster() []api.Section {
	switch h.platform {
	case config.DiscordCommPlatformIntegration, config.MattermostCommPlatformIntegration:
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
	case config.CloudSlackCommPlatformIntegration, config.CloudTeamsCommPlatformIntegration:
		return []api.Section{
			{
				Base: api.Base{
					Header: "🏁 Multi-Cluster flags",
					Description: fmt.Sprintf("`--cluster-name=%q` flag to run a command on this cluster\n", h.clusterName) +
						"`--all-clusters` flag to run commands on all clusters",
				},
			},
		}
	default:
		return nil
	}
}

func (h *HelpMessage) basicCommands() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "🛠️ Basic commands",
				Description: fmt.Sprintf("`%s ping` - ping your cluster and check its status\n", api.MessageBotNamePlaceholder) +
					fmt.Sprintf("`%s list [source|executor|action|alias]` - list available plugins and features", api.MessageBotNamePlaceholder),
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithoutDesc("Ping cluster", "ping"),
				h.btnBuilder.ForCommandWithoutDesc("List source plugins", "list sources"),
				h.btnBuilder.ForCommandWithoutDesc("List executor plugins", "list executors"),
			},
		},
	}
}

func (h *HelpMessage) footer() []api.Section {
	btns := api.Buttons{
		h.btnBuilder.ForURL("Give feedback", "https://feedback.botkube.io", api.ButtonStylePrimary),
		h.btnBuilder.ForURL("Read our docs", "https://docs.botkube.io"),
	}

	if h.platform == config.CloudSlackCommPlatformIntegration || h.platform == config.CloudTeamsCommPlatformIntegration {
		btns = append(btns, h.btnBuilder.ForURL("Get support", "https://botkube.io/support"))
	} else {
		btns = append(btns, h.btnBuilder.ForURL("Join our Slack", "https://join.botkube.io"))
	}

	btns = append(btns, h.btnBuilder.ForURL("Follow us on Twitter/X", "https://twitter.com/botkube_io"))

	if !remote.IsEnabled() {
		return []api.Section{
			{
				Buttons: btns,
			},
		}
	}

	return []api.Section{
		{
			Context: api.ContextItems{
				{Text: fmt.Sprintf("👀 _All %s mentions and events are visible to your Botkube Cloud organisation’s administrators._", api.MessageBotNamePlaceholder)},
			},
		},
		{
			Style: api.SectionStyle{
				Divider: api.DividerStyleTopNone,
			},
			Buttons: btns,
		},
	}
}

func (h *HelpMessage) notificationSections() []api.Section {
	btns := api.Buttons{
		h.btnBuilder.ForCommandWithoutDesc("Enable", "enable notifications"),
		h.btnBuilder.ForCommandWithoutDesc("Disable", "disable notifications"),
		h.btnBuilder.ForCommandWithoutDesc("Get status", "status notifications"),
	}
	instanceID := os.Getenv(remote.ProviderIdentifierEnvKey)
	if instanceID != "" {
		instanceViewURL := fmt.Sprintf("https://app.botkube.io/instances/%s", instanceID)
		btns = append(btns, h.btnBuilder.ForURL("Change notification on Cloud", instanceViewURL, api.ButtonStylePrimary))
	}
	return []api.Section{
		{
			Base: api.Base{
				Header: "📣 Notifications",
				Description: fmt.Sprintf("`%s [enable|disable|status] notifications` - set or query your notification status\n", api.MessageBotNamePlaceholder) +
					fmt.Sprintf("`%s edit sourcebindings` - select notification sources for this channel", api.MessageBotNamePlaceholder),
			},
			Buttons: btns,
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

func (h *HelpMessage) botkubeCloud() []api.Section {
	if !remote.IsEnabled() {
		return nil
	}
	return []api.Section{
		{
			Base: api.Base{
				Header: "☁️ Botkube Cloud",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithDescCmd("List connected instances", "cloud list instances"),
				h.btnBuilder.ForCommandWithDescCmd("Set channel default cluster", "cloud set default-instance"),
				h.btnBuilder.ForURL("Open Botkube Cloud", "https://app.botkube.io", api.ButtonStylePrimary),
			},
		},
	}
}

func (h *HelpMessage) aiPlugin() []api.Section {
	if !remote.IsEnabled() {
		return nil
	}
	return []api.Section{
		{
			Base: api.Base{
				Header:      "🤖 AI powered Kubernetes assistant",
				Description: fmt.Sprintf("`%s ai` use natural language to ask any questions", api.MessageBotNamePlaceholder),
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithoutDesc("Ask a question", "ai hi!", api.ButtonStylePrimary),
			},
		},
	}
}

func (h *HelpMessage) advancedFeatures() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Other features",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForURLWithTextDesc("Automation", "Automate your workflows by executing custom commands based on specific events", "https://docs.botkube.io/usage/automated-actions", api.ButtonStylePrimary),
			},
		},
	}
}
