package conversation

import (
	"strings"
)

// NormalizeChannelIdentifier removes leading and trailing spaces and # from the channel name.
// this is platform-agnostic, as different platforms use different rules:
// Slack - channel name: https://api.slack.com/methods/conversations.rename#naming
// Mattermost - channel name: https://docs.mattermost.com/channels/channel-naming-conventions.html
// Discord - channel ID: https://support.discord.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID-
func NormalizeChannelIdentifier(in string) (string, bool) {
	out := strings.TrimLeft(strings.TrimSpace(in), "#")
	return out, out != in
}
