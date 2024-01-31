package msteamsx

import (
	"context"
	"fmt"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	msgraphsdkgo "github.com/microsoftgraph/msgraph-sdk-go"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/teams"

	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/pkg/teamsx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/kubeshop/botkube/pkg/ptr"
)

const (
	serviceURL = "https://smba.trafficmanager.net/teams/"
)

type Message struct {
	Raw      graphmodels.ChatMessageable
	Rendered string
}

// Client is a client for bot and MS Graph API.
type Client struct {
	graphCLI *msgraphsdkgo.GraphServiceClient
	bot      *teamsx.Bot
}

func NewClient(appName, appID, appPassword, tenantID string) (*Client, error) {
	bot, err := teamsx.NewBot(loggerx.NewNoop(), teamsx.BotConfig{
		AppID:       appID,
		AppPassword: appPassword,
		AppName:     appName,
	})
	if err != nil {
		return nil, err
	}
	msGraphAPICli := teamsx.NewGraphAPIClientGetter(appID, appPassword)

	cli, err := msGraphAPICli.GetForTenant(tenantID)
	if err != nil {
		return nil, fmt.Errorf("while creating MS Graph API Client: %w", err)
	}

	return &Client{graphCLI: cli, bot: bot}, nil
}

func (c *Client) CreateChannel(ctx context.Context, teamID, channelName string) (string, error) {
	requestBody := graphmodels.NewChannel()
	requestBody.SetDisplayName(ptr.FromType(channelName))
	requestBody.SetDescription(ptr.FromType("Temp channel for Botkube CI Testing"))
	requestBody.SetMembershipType(ptr.FromType(graphmodels.STANDARD_CHANNELMEMBERSHIPTYPE))

	resp, err := c.graphCLI.Teams().ByTeamId(teamID).Channels().Post(ctx, requestBody, nil)
	if err != nil {
		return "", err
	}
	return ptr.ToValue(resp.GetId()), nil
}

func (c *Client) DeleteChannel(ctx context.Context, teamID, channelID string) error {
	return c.graphCLI.Teams().ByTeamId(teamID).Channels().ByChannelId(channelID).Delete(ctx, nil)
}

func (c *Client) GetChannels(ctx context.Context, teamID string) ([]string, error) {
	channels, err := c.graphCLI.Teams().ByTeamId(teamID).Channels().Get(ctx, nil)
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
	plain := api.NewPlaintextMessage(msg, false)
	return c.bot.PostMessage(ctx, teamsx.WithMessage(plain), teamsx.WithServiceURLAndConvID(serviceURL, convID))
}

func (c *Client) GetMessages(ctx context.Context, teamID, channelID string, pageSize int) ([]Message, error) {
	query := teams.ItemChannelsItemMessagesRequestBuilderGetQueryParameters{
		Top: ptr.FromType(int32(pageSize)),
	}
	options := teams.ItemChannelsItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &query,
	}
	msgs, err := c.graphCLI.Teams().ByTeamId(teamID).Channels().ByChannelId(channelID).Messages().Get(ctx, &options)
	if err != nil {
		return nil, err
	}
	var messages []Message
	for _, t := range msgs.GetValue() {
		messages = append(messages, Message{Raw: t, Rendered: c.messageFrom(t)})
	}

	return messages, nil
}

func (c *Client) messageFrom(msg graphmodels.ChatMessageable) string {
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
			msgTexts = append(msgTexts, markdown)
		}
	}
	return strings.Join(msgTexts, "\n")
}
