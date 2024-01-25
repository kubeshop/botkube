package msteamsx

import (
	"context"
	"fmt"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/infracloudio/msbotbuilder-go/core"
	"github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/kubeshop/botkube/pkg/ptr"
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
	Cli   *msgraphsdkgo.GraphServiceClient
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
	return &Client{Cli: cli, appID: appID, bot: bot}, nil
}

func (c *Client) CreateChannel(ctx context.Context, teamID, channelName string) (string, error) {
	requestBody := graphmodels.NewChannel()
	displayName := channelName
	requestBody.SetDisplayName(&displayName)
	description := "Temp channel for Botkube CI Testing"
	requestBody.SetDescription(&description)
	membershipType := graphmodels.STANDARD_CHANNELMEMBERSHIPTYPE
	requestBody.SetMembershipType(&membershipType)

	resp, err := c.Cli.Teams().ByTeamId(teamID).Channels().Post(ctx, requestBody, nil)
	if err != nil {
		return "", err
	}
	return ptr.ToValue(resp.GetId()), nil
}

func (c *Client) DeleteChannel(ctx context.Context, teamID, channelID string) error {
	return c.Cli.Teams().ByTeamId(teamID).Channels().ByChannelId(channelID).Delete(ctx, nil)
}

func (c *Client) GetChannels(ctx context.Context, teamID string) ([]string, error) {
	channels, err := c.Cli.Teams().ByTeamId(teamID).Channels().Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, c := range channels.GetValue() {
		if c.GetId() == nil || strings.EqualFold(ptr.ToValue(c.GetDisplayName()), "general") {
			continue
		}
		result = append(result, ptr.ToValue(c.GetId()))
	}
	return result, nil
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

func (c *Client) SendMessageV1(ctx context.Context, convID string, msg activity.MsgOption) error {
	ref := c.getConvReference(serviceURL, convID)
	return c.bot.ProactiveMessage(ctx, ref, activity.HandlerFuncs{
		OnMessageFunc: func(turn *activity.TurnContext) (schema.Activity, error) {
			return turn.SendActivity(msg)
		},
	})
}

func (c *Client) SendMessageV2(ctx context.Context, convID string, msg schema.Activity) error {
	cli := &activity.DefaultResponse{Client: c.bot.Client}
	ref := c.getConvReference(serviceURL, convID)

	msg = c.applyConversationReference(msg, ref)

	return cli.SendActivity(ctx, msg)
	//
	//return c.bot.ProactiveMessage(ctx, ref, activity.HandlerFuncs{
	//	OnMessageFunc: func(turn *activity.TurnContext) (schema.Activity, error) {
	//		return turn.SendActivity(msg)
	//	},
	//})
}

const channelID = "msteams"

func (c *Client) applyConversationReference(activity schema.Activity, reference schema.ConversationReference) schema.Activity {
	activity.ID = reference.ActivityID
	activity.Conversation = reference.Conversation
	activity.ChannelID = channelID

	activity.ServiceURL = reference.ServiceURL
	if activity.ServiceURL == "" {
		activity.ServiceURL = serviceURL
	}

	//activity.Recipient = reference.User
	//activity.From = schema.ChannelAccount{
	//	ID:   fmt.Sprintf("%s%s", botPrefix, p.appID),
	//	Name: p.botName,
	//}

	return activity
}
func (c *Client) GetMessages(ctx context.Context, teamID, channelID string, pageSize int) ([]MsTeamsMessage, error) {
	query := teams.ItemChannelsItemMessagesRequestBuilderGetQueryParameters{
		Top: ptr.FromType(int32(pageSize)),
	}
	options := teams.ItemChannelsItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &query,
	}
	msgs, err := c.Cli.Teams().ByTeamId(teamID).Channels().ByChannelId(channelID).Messages().Get(ctx, &options)
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

	if len(msgTexts) == 0 {
		plaintext := msg.GetBody().GetContent()

		if plaintext != nil && *plaintext != "" {
			converter := md.NewConverter("", true, nil)

			markdown, err := converter.ConvertString(*plaintext)
			if err != nil {
				fmt.Println(err)
			}
			markdown = strings.ReplaceAll(markdown, "\n\n\n", "\n")
			fmt.Println("md ->", markdown)
			msgTexts = append(msgTexts, markdown)
		}
	}
	return strings.Join(msgTexts, "\n")
}

func (c *Client) getConvReference(url, conversationID string) schema.ConversationReference {
	return schema.ConversationReference{
		User: schema.ChannelAccount{
			ID:   fmt.Sprintf("%s%s", "28:", "82ecdc81-8380-420e-9df5-7d6f85196631"),
			Name: "BotkubeDev",
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
