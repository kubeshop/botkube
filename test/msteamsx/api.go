package msteamsx

import (
	"context"
	"fmt"
	"github.com/infracloudio/msbotbuilder-go/core"
	"github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/kubeshop/botkube/internal/ptr"
	msgraphsdkgo "github.com/microsoftgraph/msgraph-sdk-go"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/teams"
	"github.com/pkg/errors"
	"strings"
)

const (
	serviceURL = "https://smba.trafficmanager.net/teams/"
)

type Client struct {
	cli   *msgraphsdkgo.GraphServiceClient
	bot   *core.BotFrameworkAdapter
	appID string
}

func New(appID, appPassword, tenantID string) (*Client, error) {
	msGraphAPICli := NewGraphAPIClientGetter(appID, appPassword)

	cli, err := msGraphAPICli.GetForTenant(tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "while creating MS Graph API Client")
	}

	msBot, err := core.NewBotAdapter(core.AdapterSetting{
		AppID:       appID,
		AppPassword: appPassword,
	})
	if err != nil {
		return nil, errors.Wrap(err, "while creating Bot Adapter Client")
	}
	bot, ok := msBot.(*core.BotFrameworkAdapter)
	if !ok {
		return nil, errors.New("invalid bot type")
	}
	return &Client{cli: cli, appID: appID, bot: bot}, nil
}

func (c *Client) CreateChannel(ctx context.Context, teamID, channelName string) (string, error) {
	requestBody := graphmodels.NewChannel()
	displayName := channelName
	requestBody.SetDisplayName(&displayName)
	description := "Temp channel for Botkube CI Testing"
	requestBody.SetDescription(&description)
	membershipType := graphmodels.STANDARD_CHANNELMEMBERSHIPTYPE
	requestBody.SetMembershipType(&membershipType)

	resp, err := c.cli.Teams().ByTeamId(teamID).Channels().Post(ctx, requestBody, nil)
	if err != nil {
		return "", err
	}
	return ptr.ToValue(resp.GetId()), nil
}

func (c *Client) DeleteChannel(ctx context.Context, teamID, channelID string) error {
	return c.cli.Teams().ByTeamId(teamID).Channels().ByChannelId(channelID).Delete(ctx, nil)
}

func (c *Client) SendMessage(ctx context.Context, convID, msg string) error {
	var msgOpts []activity.MsgOption
	msgOpts = append(msgOpts, activity.MsgOptionText(msg))
	ref := c.getConvReference(serviceURL, convID)
	return c.bot.ProactiveMessage(ctx, ref, activity.HandlerFuncs{
		OnMessageFunc: func(turn *activity.TurnContext) (schema.Activity, error) {
			return turn.SendActivity(msgOpts...)
		},
	})
}

func (c *Client) GetMessages(ctx context.Context, teamID, channelID string, pageSize int) ([]MsTeamsMessage, error) {
	limit := int32(pageSize)
	query := teams.ItemChannelsItemMessagesRequestBuilderGetQueryParameters{
		Top: &limit,
	}
	options := teams.ItemChannelsItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &query,
	}
	msgs, err := c.cli.Teams().ByTeamId(teamID).Channels().ByChannelId(channelID).Messages().Get(ctx, &options)
	if err != nil {
		return nil, err
	}
	var messages []MsTeamsMessage
	for _, t := range msgs.GetValue() {
		messages = append(messages, MsTeamsMessage{Raw: t, Rendered: c.MessageFrom(t)})
	}
	return messages, nil
}

func (c *Client) MessageFrom(msg graphmodels.ChatMessageable) string {
	var msgTexts []string
	for _, a := range msg.GetAttachments() {
		msgTexts = append(msgTexts, *a.GetContent())
	}
	return strings.Join(msgTexts, "\n")
}

func (c *Client) getConvReference(url, conversationID string) schema.ConversationReference {
	return schema.ConversationReference{
		Bot: schema.ChannelAccount{
			ID:   fmt.Sprintf("%s%s", "28:", c.appID),
			Name: "BotkubeCloud",
		},
		Conversation: schema.ConversationAccount{
			ID: conversationID,
		},
		ChannelID:  "msteams",
		ServiceURL: url,
	}
}

type MsTeamsMessage struct {
	Raw      graphmodels.ChatMessageable
	Rendered string
}
