package bot

import (
	"context"
	"strings"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/notifier"
)

// Bot connects to communication channels and reads/sends messages. It is a two-way integration.
type Bot interface {
	Start(ctx context.Context) error
	notifier.Bot
}

// ExecutorFactory facilitates creation of execute.Executor instances.
type ExecutorFactory interface {
	NewDefault(cfg execute.NewDefaultInput) execute.Executor
}

// AnalyticsReporter defines a reporter that collects analytics data.
type AnalyticsReporter interface {
	// ReportBotEnabled reports an enabled bot.
	ReportBotEnabled(platform config.CommPlatformIntegration) error
}

// FatalErrorAnalyticsReporter reports a fatal errors.
type FatalErrorAnalyticsReporter interface {
	AnalyticsReporter

	// ReportFatalError reports a fatal app error.
	ReportFatalError(err error) error

	// Close cleans up the reporter resources.
	Close() error
}

type channelConfigByID struct {
	config.ChannelBindingsByID

	alias  string
	notify bool
	name   string
}

type channelConfigByName struct {
	config.ChannelBindingsByName

	alias  string
	notify bool
}

func AsNotifiers(bots map[string]Bot) []notifier.Bot {
	notifiers := make([]notifier.Bot, 0, len(bots))
	for _, bot := range bots {
		notifiers = append(notifiers, bot)
	}
	return notifiers
}

// normalizeChannelName removes leading and trailing spaces and # from the channel name.
// this is platform-agnostic, as different platforms use different rules:
// Slack - channel name: https://api.slack.com/methods/conversations.rename#naming
// Mattermost - channel name: https://docs.mattermost.com/channels/channel-naming-conventions.html
// Discord - channel ID: https://docs.statbot.net/docs/faq/general/how-find-id/
func normalizeChannelName(in string) (string, bool) {
	out := strings.TrimLeft(strings.TrimSpace(in), "#")
	return out, out != in
}
