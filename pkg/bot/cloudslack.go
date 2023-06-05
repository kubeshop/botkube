package bot

import (
	"context"
	"errors"
	"fmt"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/slackeventsx"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/slack-go/slack"
	"regexp"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/kubeshop/botkube/pkg/api/cloudslack"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
)

var _ Bot = &CloudSlack{}

// CloudSlack listens for user's message, execute commands and sends back the response.
type CloudSlack struct {
	log             logrus.FieldLogger
	cfg             config.CloudSlack
	client          *slack.Client
	executorFactory ExecutorFactory
	reporter        cloudSlackAnalyticsReporter
	commGroupName   string
	realNamesForID  map[string]string
	botMentionRegex *regexp.Regexp
	botID           string
	channelsMutex   sync.RWMutex
	renderer        *SlackRenderer
	channels        map[string]channelConfigByName
	notifyMutex     sync.Mutex
}

// cloudSlackAnalyticsReporter defines a reporter that collects analytics data.
type cloudSlackAnalyticsReporter interface {
	FatalErrorAnalyticsReporter
	ReportCommand(platform config.CommPlatformIntegration, command string, origin command.Origin, withFilter bool) error
}

func NewCloudSlack(log logrus.FieldLogger,
	commGroupName string,
	cfg config.CloudSlack,
	executorFactory ExecutorFactory,
	reporter cloudSlackAnalyticsReporter) (*CloudSlack, error) {

	client := slack.New(cfg.BotToken, slack.OptionAppLevelToken(cfg.AppToken))

	authResp, err := client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("while testing the ability to do auth Slack request: %w", err)
	}
	botID := authResp.UserID

	botMentionRegex, err := slackBotMentionRegex(botID)
	if err != nil {
		return nil, err
	}

	channels := slackChannelsConfigFrom(cfg.Channels)
	if err != nil {
		return nil, fmt.Errorf("while producing channels configuration map by ID: %w", err)
	}

	return &CloudSlack{
		log:             log.WithField("integration", config.CloudSlackCommPlatformIntegration),
		cfg:             cfg,
		executorFactory: executorFactory,
		reporter:        reporter,
		commGroupName:   commGroupName,
		botMentionRegex: botMentionRegex,
		botID:           botID,
		renderer:        NewSlackRenderer(),
		channels:        channels,
		client:          client,
		realNamesForID:  map[string]string{},
	}, nil
}

func (b *CloudSlack) Start(ctx context.Context) error {
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	opts := []grpc.DialOption{creds}

	conn, err := grpc.Dial(b.cfg.Server, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	req := &pb.ConnectRequest{
		InstanceId: "test",
	}
	var conOpts []grpc.CallOption
	c, err := pb.NewCloudSlackClient(conn).Connect(ctx, conOpts...)
	defer c.CloseSend()
	if err != nil {
		return err
	}

	err = c.Send(req)
	if err != nil {
		return err
	}

	for {
		data, err := c.Recv()
		if err != nil {
			return err
		}
		event, err := slackeventsx.ParseEvent(data.Event, slackeventsx.OptionNoVerifyToken())
		switch event.Type {
		case slackeventsx.CallbackEvent:
			b.log.Debugf("Got callback event %s", formatx.StructDumper().Sdump(event))
			innerEvent := event.InnerEvent
			switch ev := innerEvent.Data.(type) {
			case *slackeventsx.AppMentionEvent:
				b.log.Debugf("Got app mention %s", formatx.StructDumper().Sdump(innerEvent))
				userName := b.getRealNameWithFallbackToUserID(ctx, ev.User)
				msg := socketSlackMessage{
					Text:            ev.Text,
					Channel:         ev.Channel,
					ThreadTimeStamp: ev.ThreadTimeStamp,
					UserID:          ev.User,
					UserName:        userName,
					CommandOrigin:   command.TypedOrigin,
				}

				if err := b.handleMessage(ctx, msg); err != nil {
					b.log.Errorf("while handling message: %s", err.Error())
				}
			}
		}
		fmt.Printf("received: %q\n", event)
	}
}

func (b *CloudSlack) SendMessage(ctx context.Context, msg interactive.CoreMessage, sourceBindings []string) error {
	return nil
}

func (b *CloudSlack) SendMessageToAll(ctx context.Context, msg interactive.CoreMessage) error {
	return nil
}

func (b *CloudSlack) Type() config.IntegrationType {
	return config.BotIntegrationType
}

func (b *CloudSlack) IntegrationName() config.CommPlatformIntegration {
	return config.CloudSlackCommPlatformIntegration
}

func (b *CloudSlack) getRealNameWithFallbackToUserID(ctx context.Context, userID string) string {
	realName, exists := b.realNamesForID[userID]
	if exists {
		return realName
	}

	user, err := b.client.GetUserInfoContext(ctx, userID)
	if err != nil {
		b.log.Errorf("while getting user info: %s", err.Error())
		return userID
	}

	if user == nil || user.RealName == "" {
		return userID
	}

	b.realNamesForID[userID] = user.RealName
	return user.RealName
}

func (b *CloudSlack) handleMessage(ctx context.Context, event socketSlackMessage) error {
	// Handle message only if starts with mention
	request, found := b.findAndTrimBotMention(event.Text)
	if !found {
		b.log.Debugf("Ignoring message as it doesn't contain %q mention", b.botID)
		return nil
	}

	b.log.Debugf("Slack incoming Request: %s", request)

	// Unfortunately we need to do a call for channel name based on ID every time a message arrives.
	// I wanted to query for channel IDs based on names and prepare a map in the `slackChannelsConfigFrom`,
	// but unfortunately Botkube would need another scope (get all conversations).
	// Keeping current way of doing this until we come up with a better idea.
	info, err := b.client.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID:     event.Channel,
		IncludeLocale: true,
	})
	if err != nil {
		return fmt.Errorf("while getting conversation info: %w", err)
	}

	channel, exists := b.getChannels()[info.Name]

	e := b.executorFactory.NewDefault(execute.NewDefaultInput{
		CommGroupName:   b.commGroupName,
		Platform:        b.IntegrationName(),
		NotifierHandler: b,
		Conversation: execute.Conversation{
			Alias:            channel.alias,
			ID:               channel.Identifier(),
			DisplayName:      info.Name,
			ExecutorBindings: channel.Bindings.Executors,
			SourceBindings:   channel.Bindings.Sources,
			IsKnown:          exists,
			CommandOrigin:    event.CommandOrigin,
			SlackState:       event.State,
		},
		Message: request,
		User: execute.UserInput{
			Mention:     fmt.Sprintf("<@%s>", event.UserID),
			DisplayName: event.UserName,
		},
	})
	response := e.Execute(ctx)
	err = b.send(ctx, event, response)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	return nil
}

func (b *CloudSlack) send(ctx context.Context, event socketSlackMessage, resp interactive.CoreMessage) error {
	b.log.Debugf("Sending message to channel %q: %+v", event.Channel, resp)

	resp.ReplaceBotNamePlaceholder(b.BotName())
	markdown := b.renderer.MessageToMarkdown(resp)

	if len(markdown) == 0 {
		return errors.New("while reading Slack response: empty response")
	}

	// Upload message as a file if too long
	var file *slack.File
	var err error
	if len(markdown) >= slackMaxMessageSize {
		file, err = uploadFileToSlack(ctx, event.Channel, resp, b.client, event.ThreadTimeStamp)
		if err != nil {
			return err
		}
		resp = interactive.CoreMessage{
			Message: api.Message{
				PlaintextInputs: resp.PlaintextInputs,
			},
		}
	}

	// we can open modal only if we have a TriggerID (it's available when user clicks a button)
	if resp.Type == api.PopupMessage && event.TriggerID != "" {
		modalView := b.renderer.RenderModal(resp)
		modalView.PrivateMetadata = event.Channel
		_, err := b.client.OpenViewContext(ctx, event.TriggerID, modalView)
		if err != nil {
			return fmt.Errorf("while opening modal: %w", err)
		}
		return nil
	}

	options := []slack.MsgOption{
		b.renderer.RenderInteractiveMessage(resp),
	}

	if ts := b.getThreadOptionIfNeeded(event, file); ts != nil {
		options = append(options, ts)
	}

	if resp.ReplaceOriginal && event.ResponseURL != "" {
		options = append(options, slack.MsgOptionReplaceOriginal(event.ResponseURL))
	}

	if resp.OnlyVisibleForYou {
		if _, err := b.client.PostEphemeralContext(ctx, event.Channel, event.UserID, options...); err != nil {
			return fmt.Errorf("while posting Slack message visible only to user: %w", err)
		}
	} else {
		if _, _, err := b.client.PostMessageContext(ctx, event.Channel, options...); err != nil {
			return fmt.Errorf("while posting Slack message: %w", err)
		}
	}

	b.log.Debugf("Message successfully sent to channel %q", event.Channel)
	return nil
}

func (b *CloudSlack) findAndTrimBotMention(msg string) (string, bool) {
	if !b.botMentionRegex.MatchString(msg) {
		return "", false
	}

	return b.botMentionRegex.ReplaceAllString(msg, ""), true
}

func (b *CloudSlack) getChannels() map[string]channelConfigByName {
	b.channelsMutex.RLock()
	defer b.channelsMutex.RUnlock()
	return b.channels
}

func (b *CloudSlack) BotName() string {
	return fmt.Sprintf("<@%s>", b.botID)
}

func (b *CloudSlack) getThreadOptionIfNeeded(event socketSlackMessage, file *slack.File) slack.MsgOption {
	//if the message is from thread then add an option to return the response to the thread
	if event.ThreadTimeStamp != "" {
		return slack.MsgOptionTS(event.ThreadTimeStamp)
	}

	if file == nil {
		return nil
	}

	// If the message was already as a file attachment, reply it a given thread
	for _, share := range file.Shares.Public {
		if len(share) >= 1 && share[0].Ts != "" {
			return slack.MsgOptionTS(share[0].Ts)
		}
	}

	return nil
}

// NotificationsEnabled returns current notification status for a given channel name.
func (b *CloudSlack) NotificationsEnabled(channelName string) bool {
	channel, exists := b.getChannels()[channelName]
	if !exists {
		return false
	}

	return channel.notify
}

// SetNotificationsEnabled sets a new notification status for a given channel name.
func (b *CloudSlack) SetNotificationsEnabled(channelName string, enabled bool) error {
	// avoid race conditions with using the setter concurrently, as we set whole map
	b.notifyMutex.Lock()
	defer b.notifyMutex.Unlock()

	channels := b.getChannels()
	channel, exists := channels[channelName]
	if !exists {
		return execute.ErrNotificationsNotConfigured
	}

	channel.notify = enabled
	channels[channelName] = channel
	b.setChannels(channels)

	return nil
}

func (b *CloudSlack) setChannels(channels map[string]channelConfigByName) {
	b.channelsMutex.Lock()
	defer b.channelsMutex.Unlock()
	b.channels = channels
}
