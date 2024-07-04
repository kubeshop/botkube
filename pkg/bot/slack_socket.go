package bot

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/sourcegraph/conc/pool"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/internal/health"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

// TODO: Refactor this file as a part of https://github.com/kubeshop/botkube/issues/667
//    - split to multiple files in a separate package,
//    - review all the methods and see if they can be simplified.

// slackMaxMessageSize max size before a message should be uploaded as a file.
//
// "The text for the block, in the form of a text object.
//
//	Maximum length for the text in this field is 3000 characters.  (..)"
//
// source: https://api.slack.com/reference/block-kit/blocks#section
const (
	slackMaxMessageSize = 3001
)

var _ Bot = &SocketSlack{}

// SocketSlack listens for user's message, execute commands and sends back the response.
type SocketSlack struct {
	log               logrus.FieldLogger
	executorFactory   ExecutorFactory
	reporter          socketSlackAnalyticsReporter
	botID             string
	client            *slack.Client
	channelsMutex     sync.RWMutex
	channels          map[string]channelConfigByName
	notifyMutex       sync.Mutex
	botMentionRegex   *regexp.Regexp
	commGroupMetadata CommGroupMetadata
	renderer          *SlackRenderer
	realNamesForID    map[string]string
	msgStatusTracker  *SlackMessageStatusTracker
	messages          chan slackMessage
	messageWorkers    *pool.Pool
	shutdownOnce      sync.Once
	status            health.PlatformStatusMsg
	failureReason     health.FailureReasonMsg
	errorMsg          string
}

// socketSlackAnalyticsReporter defines a reporter that collects analytics data.
type socketSlackAnalyticsReporter interface {
	FatalErrorAnalyticsReporter
	ReportCommand(in analytics.ReportCommandInput) error
}

// NewSocketSlack creates a new SocketSlack instance.
func NewSocketSlack(log logrus.FieldLogger, commGroupMetadata CommGroupMetadata, cfg config.SocketSlack, executorFactory ExecutorFactory, reporter socketSlackAnalyticsReporter) (*SocketSlack, error) {
	client := slack.New(cfg.BotToken, slack.OptionAppLevelToken(cfg.AppToken))

	authResp, err := client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("while testing the ability to do authenticated Slack request: %w", slackError(err, ""))
	}
	botID := authResp.UserID

	botMentionRegex, err := slackBotMentionRegex(botID)
	if err != nil {
		return nil, err
	}

	channels := slackChannelsConfigFrom(log, cfg.Channels)
	if err != nil {
		return nil, fmt.Errorf("while producing channels configuration map by ID: %w", err)
	}

	return &SocketSlack{
		log:               log,
		executorFactory:   executorFactory,
		reporter:          reporter,
		botID:             botID,
		client:            client,
		channels:          channels,
		commGroupMetadata: commGroupMetadata,
		renderer:          NewSlackRenderer(),
		botMentionRegex:   botMentionRegex,
		realNamesForID:    map[string]string{},
		msgStatusTracker:  NewSlackMessageStatusTracker(log, client),
		messages:          make(chan slackMessage, platformMessageChannelSize),
		messageWorkers:    pool.New().WithMaxGoroutines(platformMessageWorkersCount),
		status:            health.StatusUnknown,
		failureReason:     "",
	}, nil
}

// Start starts the Slack WebSocket connection and listens for messages
func (b *SocketSlack) Start(ctx context.Context) error {
	b.log.Info("Starting bot")

	websocketClient := socketmode.New(b.client)

	go func() {
		defer analytics.ReportPanicIfOccurs(b.log, b.reporter)
		socketRunErr := websocketClient.RunContext(ctx)
		if socketRunErr != nil {
			reportErr := b.reporter.ReportFatalError(socketRunErr)
			if reportErr != nil {
				b.log.Errorf("while reporting socket error: %s", reportErr.Error())
			}
		}
	}()

	b.setFailureReason("", "")
	go b.startMessageProcessor(ctx)

	for {
		select {
		case <-ctx.Done():
			b.log.Info("Shutdown requested. Finishing...")
			b.shutdown()
			return nil
		case event := <-websocketClient.Events:
			switch event.Type {
			case socketmode.EventTypeConnecting:
				b.log.Info("Botkube is connecting to Slack...")
			case socketmode.EventTypeConnected:
				if err := b.reporter.ReportBotEnabled(b.IntegrationName(), b.commGroupMetadata.Index); err != nil {
					b.log.Errorf("report analytics error: %s", err.Error())
				}
				b.log.Info("Botkube connected to Slack!")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := event.Data.(slackevents.EventsAPIEvent)
				if !ok {
					b.log.Errorf("Invalid event %+v\n", event.Data)
					continue
				}
				websocketClient.Ack(*event.Request)
				if eventsAPIEvent.Type == slackevents.CallbackEvent {
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.MessageEvent:
						b.log.Debugf("Got message %s", formatx.StructDumper().Sdump(ev))

						// For now, we are interested only in channel messages.
						// More info: https://github.com/slack-go/slack/blob/4c00dbda0cd36b3ecb9d64ed6adc79d1cb4c0ca5/slackevents/inner_events.go#L231-L238
						if ev.ChannelType != "channel" {
							b.log.WithField("type", ev.SubType).Debug("Ignoring non-channel message...")
							continue
						}

						// For now, we are interested only in the "root" message.
						// More info: https://api.slack.com/events/message#subtypes
						if ev.SubType != "" {
							b.log.WithField("sub-type", ev.SubType).Debug("Ignoring sub-type message...")
							continue
						}

						// For now, we don't follow thread messages.
						if ev.ThreadTimeStamp != "" {
							b.log.Debug("Ignoring thread message...")
							continue
						}

						_, hasBotMention := b.findAndTrimBotMention(ev.Text)
						if hasBotMention {
							// we will get the same event on slackevents.AppMentionEvent, so to avoid duplication, let's skip this one
							continue
						}

						msg := slackMessage{
							Text:                 ev.Text,
							Channel:              ev.Channel,
							RootMessageTimeStamp: ev.TimeStamp,
							ThreadTimeStamp:      ev.ThreadTimeStamp,
							EventTimeStamp:       ev.EventTimeStamp,
							UserID:               ev.User,
							UserName:             b.getRealNameWithFallbackToUserID(ctx, ev.User),
						}

						b.messages <- msg
					case *slackevents.AppMentionEvent:
						if ev.BotID != "" {
							b.log.Infof("Not reacting to own messages, or anything from other bots")
							continue
						}
						b.log.Debugf("Got app mention %s", formatx.StructDumper().Sdump(innerEvent))
						msg := slackMessage{
							Text:                 ev.Text,
							Channel:              ev.Channel,
							RootMessageTimeStamp: ev.TimeStamp,
							ThreadTimeStamp:      ev.ThreadTimeStamp,
							EventTimeStamp:       ev.EventTimeStamp,
							UserID:               ev.User,
							UserName:             b.getRealNameWithFallbackToUserID(ctx, ev.User),
							CommandOrigin:        command.TypedOrigin,
						}

						b.messages <- msg
					default:
						b.log.Debugf("Got callback event that we don't watch %s", formatx.StructDumper().Sdump(eventsAPIEvent))
					}
				}
			case socketmode.EventTypeInteractive:
				callback, ok := event.Data.(slack.InteractionCallback)
				if !ok {
					b.log.Errorf("Invalid event %+v\n", event.Data)
					continue
				}

				websocketClient.Ack(*event.Request)

				switch callback.Type {
				case slack.InteractionTypeBlockActions:
					b.log.Debugf("Got block action %s", formatx.StructDumper().Sdump(callback))

					if len(callback.ActionCallback.BlockActions) != 1 {
						b.log.Debug("Ignoring callback as the number of actions is different from 1")
						continue
					}

					act := callback.ActionCallback.BlockActions[0]
					if act == nil || strings.HasPrefix(act.ActionID, urlButtonActionIDPrefix) {
						reportErr := b.reporter.ReportCommand(analytics.ReportCommandInput{Platform: b.IntegrationName(), Command: act.ActionID, Origin: command.ButtonClickOrigin})
						if reportErr != nil {
							b.log.Errorf("while reporting URL command, error: %s", reportErr.Error())
						}
						continue // skip the url actions
					}

					channelID := callback.Channel.ID
					if channelID == "" && callback.View.ID != "" {
						// TODO: add support when we will need to handle button clicks from active modal.
						//
						// The request is coming from active modal, currently we don't support that.
						// We process that only when the modal is submitted (see slack.InteractionTypeViewSubmission action type).
						b.log.Debug("Ignoring callback as its source is an active modal")
						continue
					}

					cmd, cmdOrigin := resolveBlockActionCommand(*act)
					// Use thread's TS if interactive call triggered within thread.
					threadTs := callback.MessageTs
					if callback.Message.Msg.ThreadTimestamp != "" {
						threadTs = callback.Message.Msg.ThreadTimestamp
					}

					state := removeBotNameFromIDs(b.BotName(), callback.BlockActionState)

					userName := b.getRealNameWithFallbackToUserID(ctx, callback.User.ID)
					msg := slackMessage{
						Text:            cmd,
						Channel:         channelID,
						ThreadTimeStamp: threadTs,
						TriggerID:       callback.TriggerID,
						UserID:          callback.User.ID,
						UserName:        userName,
						CommandOrigin:   cmdOrigin,
						State:           state,
						EventTimeStamp:  callback.Message.Timestamp,
						ResponseURL:     callback.ResponseURL,
						BlockID:         act.BlockID,
					}
					b.messages <- msg
				case slack.InteractionTypeViewSubmission: // this event is received when modal is submitted

					// the map key is the ID of the input block, for us, it's autogenerated
					for _, item := range callback.View.State.Values {
						for actID, act := range item {
							act.ActionID = actID // normalize event

							cmd, cmdOrigin := resolveBlockActionCommand(act)
							userName := b.getRealNameWithFallbackToUserID(ctx, callback.User.ID)
							msg := slackMessage{
								Text:           cmd,
								Channel:        callback.View.PrivateMetadata,
								UserID:         callback.User.ID,
								UserName:       userName,
								EventTimeStamp: "", // there is no timestamp for interactive modals
								CommandOrigin:  cmdOrigin,
							}

							b.messages <- msg
						}
					}
				default:
					b.log.Debugf("get unhandled event %s", callback.Type)
				}

			case socketmode.EventTypeErrorBadMessage:
				b.log.Errorf("Bad message: %+v\n", event.Data)
			case socketmode.EventTypeIncomingError:
				b.log.Errorf("Incoming error: %+v\n", event.Data)
			case socketmode.EventTypeConnectionError:
				b.log.Errorf("Slack connection error: %+v\n", event.Data)
			}
		}
	}
}

func removeBotNameFromIDs(botName string, state *slack.BlockActionStates) *slack.BlockActionStates {
	if state == nil {
		return nil
	}

	for blockID, blocks := range state.Values {
		updateBlocks := map[string]slack.BlockAction{}
		for oldID, value := range blocks {
			newID := strings.TrimPrefix(oldID, botName)
			newID = strings.TrimSpace(newID)
			updateBlocks[newID] = value
		}

		// replace with normalized blocks
		state.Values[blockID] = updateBlocks
	}

	return state
}

// Type describes the notifier type.
func (b *SocketSlack) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// IntegrationName describes the notifier integration name.
func (b *SocketSlack) IntegrationName() config.CommPlatformIntegration {
	return config.SocketSlackCommPlatformIntegration
}

// NotificationsEnabled returns current notification status for a given channel name.
func (b *SocketSlack) NotificationsEnabled(channelName string) bool {
	channel, exists := b.getChannels()[channelName]
	if !exists {
		return false
	}

	return channel.notify
}

// SetNotificationsEnabled sets a new notification status for a given channel name.
func (b *SocketSlack) SetNotificationsEnabled(channelName string, enabled bool) error {
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

func (b *SocketSlack) startMessageProcessor(ctx context.Context) {
	b.log.Info("Starting socket slack message processor...")
	defer b.log.Info("Stopped socket slack message processor...")

	for msg := range b.messages {
		b.messageWorkers.Go(func() {
			err := b.handleMessage(ctx, msg)
			if err != nil {
				b.log.WithError(err).Error("Failed to handle Socket Slack message")
			}
		})
	}
}

func (b *SocketSlack) shutdown() {
	b.shutdownOnce.Do(func() {
		b.log.Info("Shutting down socket slack message processor...")
		close(b.messages)
		b.messageWorkers.Wait()
	})
}

func (b *SocketSlack) handleMessage(ctx context.Context, event slackMessage) error {
	// Handle message only if starts with mention
	request, hasBotMention := b.findAndTrimBotMention(event.Text)

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
	bindings := channel.Bindings
	processedEmoji := msgProcessedEmoji
	if !hasBotMention { // there wasn't botkube mentions, trying to match against messages
		messageTrigger, matched := b.hasMatchingTextMessageTrigger(channel, request, event.UserID)
		if !matched {
			b.log.WithField("matchers", formatx.StructDumper().Sdump(channel.MessageTriggers)).Debugf("Ignoring message as it doesn't contain %q mention nor text matchers", b.botID)
			return nil
		}
		bindings = config.BotBindings{Executors: messageTrigger.Executors}
		request = messageTrigger.Command
		if messageTrigger.ProcessedEmojiIndicator != nil {
			processedEmoji = *messageTrigger.ProcessedEmojiIndicator
		}
	}

	permalink, err := b.client.GetPermalink(&slack.PermalinkParameters{
		Channel: event.Channel,
		Ts:      event.EventTimeStamp,
	})
	if err != nil {
		b.log.WithError(err).Error("Cannot get permalink")
	}

	e := b.executorFactory.NewDefault(execute.NewDefaultInput{
		CommGroupName:   b.commGroupMetadata.Name,
		Platform:        b.IntegrationName(),
		NotifierHandler: b,
		Conversation: execute.Conversation{
			Alias:            channel.alias,
			ID:               channel.Identifier(),
			DisplayName:      info.Name,
			ExecutorBindings: bindings.Executors,
			SourceBindings:   bindings.Sources,
			IsKnown:          exists,
			CommandOrigin:    event.CommandOrigin,
			SlackState:       event.State,
			URL:              permalink,
			Text:             event.Text,
			ParentActivityID: event.GetTimestamp(),
		},
		Message: request,
		User: execute.UserInput{
			Mention:     fmt.Sprintf("<@%s>", event.UserID),
			DisplayName: event.UserName,
		},
	})

	msgRef := b.msgStatusTracker.GetMsgRef(event)
	if exists {
		b.msgStatusTracker.MarkAsReceived(msgRef)
	}

	response := e.Execute(ctx)
	err = b.send(ctx, event, response)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	if exists {
		b.msgStatusTracker.MarkAsProcessedWithCustomEmoji(msgRef, processedEmoji)
	}
	return nil
}

func (b *SocketSlack) hasMatchingTextMessageTrigger(channel channelConfigByName, request string, id string) (config.TextMessageTriggers, bool) {
	for _, binding := range channel.MessageTriggers {
		allowed, err := binding.Text.IsAllowed(request)
		if err != nil {
			b.log.WithError(err).Error("Cannot validate text message constraint")
			continue
		}

		if !allowed {
			continue
		}

		if binding.IsUserExcluded(id) {
			continue
		}

		return binding, true
	}
	return config.TextMessageTriggers{}, false
}

func (b *SocketSlack) send(ctx context.Context, event slackMessage, in interactive.CoreMessage) error {
	b.log.Debugf("Sending message to channel %q: %+v", event.Channel, in)

	var msgs []api.Message
	if !in.Message.IsEmpty() {
		msgs = append(msgs, in.Message)
	}

	msgs = append(msgs, in.Messages...)

	for idx := range msgs {
		if msgs[idx].IsEmpty() {
			continue
		}
		msgs[idx].ReplaceBotNamePlaceholder(b.BotName())

		resp := interactive.CoreMessage{
			Header:      in.Header,
			Description: in.Description,
			Metadata:    in.Metadata,
			Message:     msgs[idx],
		}

		markdown := b.renderer.MessageToMarkdown(resp)

		if len(markdown) == 0 {
			return errors.New("while reading Slack response: empty response")
		}

		// Upload message as a file if too long
		var err error
		var file *slack.File
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
		if resp.Message.Type == api.PopupMessage && event.TriggerID != "" {
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

		if resp.Message.Type == api.ThreadMessage && event.ThreadTimeStamp == "" {
			// if the message should be sent in thread, but thread is not yet started, then use the root message timestamp
			event.ThreadTimeStamp = event.RootMessageTimeStamp
		}
		if ts := b.getThreadOptionIfNeeded(event, file); ts != nil {
			options = append(options, ts)
		}

		if resp.Message.ReplaceOriginal && event.ResponseURL != "" {
			options = append(options, slack.MsgOptionReplaceOriginal(event.ResponseURL))
		}

		if resp.Message.OnlyVisibleForYou {
			if _, err := b.client.PostEphemeralContext(ctx, event.Channel, event.UserID, options...); err != nil {
				return fmt.Errorf("while posting Slack message visible only to user: %w", err)
			}
		} else {
			id := event.Channel
			if resp.Message.UserHandle != "" {
				id = resp.Message.UserHandle
			}

			if resp.Message.ParentActivityID != "" {
				options = append(options, slack.MsgOptionTS(resp.Message.ParentActivityID))
			}

			_, _, err := b.client.PostMessageContext(ctx, id, options...)
			if err != nil {
				return fmt.Errorf("while posting Slack message: %w", slackError(err, event.Channel))
			}
		}

		b.log.Debugf("Message successfully sent to channel %q", event.Channel)
	}

	return nil
}

func (b *SocketSlack) getChannelsToNotify(sourceBindings []string) []string {
	var out []string
	for _, cfg := range b.getChannels() {
		if !cfg.notify {
			b.log.Infof("Skipping notification for channel %q as notifications are disabled.", cfg.Identifier())
			continue
		}

		if !sliceutil.Intersect(sourceBindings, cfg.Bindings.Sources) {
			continue
		}

		out = append(out, cfg.Identifier())
	}
	return out
}

// SendMessage sends message with interactive sections to selected Slack channels.
func (b *SocketSlack) SendMessage(ctx context.Context, msg interactive.CoreMessage, sourceBindings []string) error {
	errs := multierror.New()
	for _, channelName := range b.getChannelsToNotify(sourceBindings) {
		msgMetadata := slackMessage{
			Channel:         channelName,
			ThreadTimeStamp: "",
			BlockID:         uuid.New().String(),
		}
		err := b.send(ctx, msgMetadata, msg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Slack message to channel %q: %w", channelName, err))
			continue
		}
	}

	return errs.ErrorOrNil()
}

// SendMessageToAll sends message with interactive sections to all Slack channels.
func (b *SocketSlack) SendMessageToAll(ctx context.Context, msg interactive.CoreMessage) error {
	errs := multierror.New()
	for _, channel := range b.getChannels() {
		channelName := channel.Name
		msgMetadata := slackMessage{
			Channel: channelName,
			BlockID: uuid.New().String(),
		}
		err := b.send(ctx, msgMetadata, msg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Slack message to channel %q (alias: %q): %w", channelName, channel.alias, err))
			continue
		}
	}

	return errs.ErrorOrNil()
}

// BotName returns the Bot name.
func (b *SocketSlack) BotName() string {
	return fmt.Sprintf("<@%s>", b.botID)
}

func (b *SocketSlack) getChannels() map[string]channelConfigByName {
	b.channelsMutex.RLock()
	defer b.channelsMutex.RUnlock()
	return b.channels
}

func (b *SocketSlack) setChannels(channels map[string]channelConfigByName) {
	b.channelsMutex.Lock()
	defer b.channelsMutex.Unlock()
	b.channels = channels
}

func (b *SocketSlack) findAndTrimBotMention(msg string) (string, bool) {
	if !b.botMentionRegex.MatchString(msg) {
		return msg, false
	}

	return b.botMentionRegex.ReplaceAllString(msg, ""), true
}

func resolveBlockActionCommand(act slack.BlockAction) (string, command.Origin) {
	cmd := act.Value
	cmdOrigin := command.UnknownOrigin

	switch act.Type {
	// currently we support only interactive.MultiSelect option
	case "multi_static_select":
		var items []string
		for _, item := range act.SelectedOptions {
			items = append(items, item.Value)
		}
		cmd = fmt.Sprintf("%s %s", act.ActionID, strings.Join(items, ","))
		cmdOrigin = command.MultiSelectValueChangeOrigin
	case "static_select":
		// Example of commands that are handled here:
		//   @Botkube kcc --verbs get
		//   @Botkube kcc --resource-type
		cmd = fmt.Sprintf("%s %s", act.ActionID, act.SelectedOption.Value)
		cmdOrigin = command.SelectValueChangeOrigin
	case "button":
		cmdOrigin = command.ButtonClickOrigin
	case "plain_text_input":
		cmd = fmt.Sprintf("%s%q", act.BlockID, strings.TrimSpace(act.Value))
		cmdOrigin = command.PlainTextInputOrigin
	}

	return cmd, cmdOrigin
}

func (b *SocketSlack) getThreadOptionIfNeeded(event slackMessage, file *slack.File) slack.MsgOption {
	// if the message is from thread then add an option to return the response to the thread
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

func (b *SocketSlack) getRealNameWithFallbackToUserID(ctx context.Context, userID string) string {
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

func (b *SocketSlack) setFailureReason(reason health.FailureReasonMsg, errorMsg string) {
	if reason == "" {
		b.status = health.StatusHealthy
	} else {
		b.status = health.StatusUnHealthy
	}
	b.failureReason = reason
	b.errorMsg = errorMsg
}

func (b *SocketSlack) GetStatus() health.PlatformStatus {
	return health.PlatformStatus{
		Status:   b.status,
		Restarts: "0/0",
		Reason:   b.failureReason,
		ErrorMsg: b.errorMsg,
	}
}

func uploadFileToSlack(ctx context.Context, channel string, resp interactive.CoreMessage, client *slack.Client, ts string) (*slack.File, error) {
	content := interactive.MessageToPlaintext(resp, interactive.NewlineFormatter)
	r := strings.NewReader(content)

	params := slack.UploadFileV2Parameters{
		Filename:        "Response.txt",
		FileSize:        len(content),
		Title:           "Response.txt",
		InitialComment:  resp.Description,
		Reader:          r,
		Channel:         channel,
		ThreadTimestamp: ts,
	}

	file, err := client.UploadFileV2Context(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("while uploading file: %w", err)
	}

	return file, nil
}
