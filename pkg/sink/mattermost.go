package sink

import (
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
)

// MattermostBot contains server URL and token
type MattermostBot struct {
	log logrus.FieldLogger

	Client       *model.Client4
	Channel      string
	Notification config.Notification
}

// NewMattermost returns new MattermostBot object
func NewMattermost(log logrus.FieldLogger, c config.Mattermost) (*MattermostBot, error) {
	// Set configurations for MattermostBot server
	client := model.NewAPIv4Client(c.URL)
	client.SetOAuthToken(c.Token)
	botTeam, resp := client.GetTeamByName(c.Team, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	botChannel, resp := client.GetChannelByName(c.Channels.GetFirst().Name, botTeam.Id, "")
	if resp.Error != nil {
		return nil, resp.Error
	}

	return &MattermostBot{
		log:          log,
		Client:       client,
		Channel:      botChannel.Id,
		Notification: c.Notification,
	}, nil
}
