package remote

import (
	"strings"

	"github.com/kubeshop/botkube/internal/remote/graphql"
)

// NewBotPlatform creates new BotPlatform from string
func NewBotPlatform(s string) *graphql.BotPlatform {
	var platform graphql.BotPlatform
	switch strings.ToUpper(s) {
	case "SLACK", "CLOUDSLACK", "SOCKETSLACK":
		platform = graphql.BotPlatformSLACk
	case "DISCORD":
		platform = graphql.BotPlatformDiscord
	case "MATTERMOST":
		platform = graphql.BotPlatformMattermost
	case "TEAMS":
		fallthrough
	case "MS_TEAMS", "CLOUDTEAMS":
		platform = graphql.BotPlatformMsTeams
	default:
		platform = graphql.BotPlatformUnknown
	}

	return &platform
}
