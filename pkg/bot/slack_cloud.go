package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/botkube/internal/config/remote"
	"github.com/kubeshop/botkube/pkg/api"
	pb "github.com/kubeshop/botkube/pkg/api/cloudslack"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

const (
	APIKeyContextKey        = "X-Api-Key"       // #nosec
	DeploymentIDContextKey  = "X-Deployment-Id" // #nosec
	retryDelay              = time.Second
	maxRetries              = 30
	successIntervalDuration = 3 * time.Minute
	quotaExceededMsg        = "Quota exceeded detected. Stopping reconnecting to Botkube Cloud gRPC API..."
)

var _ Bot = &CloudSlack{}

// CloudSlack listens for user's message, execute commands and sends back the response.
type CloudSlack struct {
	log              logrus.FieldLogger
	cfg              config.CloudSlack
	client           *slack.Client
	executorFactory  ExecutorFactory
	reporter         cloudSlackAnalyticsReporter
	commGroupName    string
	realNamesForID   map[string]string
	botMentionRegex  *regexp.Regexp
	botID            string
	channelsMutex    sync.RWMutex
	renderer         *SlackRenderer
	channels         map[string]channelConfigByName
	notifyMutex      sync.Mutex
	clusterName      string
	msgStatusTracker *SlackMessageStatusTracker
	status           StatusMsg
	maxRetries       int
	failuresNo       int
	failureReason    FailureReasonMsg
}

// cloudSlackAnalyticsReporter defines a reporter that collects analytics data.
type cloudSlackAnalyticsReporter interface {
	FatalErrorAnalyticsReporter
	ReportCommand(platform config.CommPlatformIntegration, command string, origin command.Origin, withFilter bool) error
}

func NewCloudSlack(log logrus.FieldLogger,
	commGroupName string,
	cfg config.CloudSlack,
	clusterName string,
	executorFactory ExecutorFactory,
	reporter cloudSlackAnalyticsReporter) (*CloudSlack, error) {
	client := slack.New(cfg.Token)

	_, err := client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("while testing the ability to do auth Slack request: %w", err)
	}

	botMentionRegex, err := slackBotMentionRegex(cfg.BotID)
	if err != nil {
		return nil, err
	}

	channels := slackChannelsConfigFrom(log, cfg.Channels)
	if err != nil {
		return nil, fmt.Errorf("while producing channels configuration map by ID: %w", err)
	}

	return &CloudSlack{
		log:              log,
		cfg:              cfg,
		executorFactory:  executorFactory,
		reporter:         reporter,
		commGroupName:    commGroupName,
		botMentionRegex:  botMentionRegex,
		renderer:         NewSlackRenderer(),
		channels:         channels,
		client:           client,
		botID:            cfg.BotID,
		clusterName:      clusterName,
		realNamesForID:   map[string]string{},
		msgStatusTracker: NewSlackMessageStatusTracker(log, client),
		status:           StatusUnknown,
		maxRetries:       maxRetries,
		failuresNo:       0,
		failureReason:    "",
	}, nil
}

func (b *CloudSlack) Start(ctx context.Context) error {
	if b.cfg.ExecutionEventStreamingDisabled {
		b.setFailureReason(FailureReasonQuotaExceeded)
		b.log.Warn(quotaExceededMsg)
		return nil
	}
	return b.withRetries(ctx, b.log, func() error {
		return b.start(ctx)
	})
}

func (b *CloudSlack) withRetries(ctx context.Context, log logrus.FieldLogger, fn func() error) error {
	b.failuresNo = 0
	var lastFailureTimestamp time.Time
	return retry.Do(
		func() error {
			err := fn()
			if err != nil {
				if !lastFailureTimestamp.IsZero() && time.Since(lastFailureTimestamp) >= successIntervalDuration {
					// if the last run was long enough, we treat is as success, so we reset failures
					log.Infof("Resetting failures counter as last failure was more than %s ago", successIntervalDuration)
					b.failuresNo = 0
				}

				if b.failuresNo >= b.maxRetries {
					b.setFailureReason(FailureReasonMaxRetriesExceeded)
					log.Debugf("Reached max number of %d retries: %s", b.maxRetries, err)
					return retry.Unrecoverable(err)
				}

				lastFailureTimestamp = time.Now()
				b.failuresNo++
				return err
			}
			b.setFailureReason("")
			return nil
		},
		retry.OnRetry(func(_ uint, err error) {
			log.Warnf("Retrying Cloud Slack startup (attempt no %d/%d): %s", b.failuresNo, b.maxRetries, err)
		}),
		retry.Delay(retryDelay),
		retry.Attempts(0), // infinite, we cancel that by our own
		retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
}

func (b *CloudSlack) start(ctx context.Context) error {
	messageWorkers := pool.New().WithMaxGoroutines(platformMessageWorkersCount)
	messages := make(chan *pb.ConnectResponse, platformMessageChannelSize)
	defer b.shutdown(messageWorkers, messages)

	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	opts := []grpc.DialOption{creds,
		grpc.WithStreamInterceptor(b.addStreamingClientCredentials()),
		grpc.WithUnaryInterceptor(b.addUnaryClientCredentials()),
	}

	conn, err := grpc.Dial(b.cfg.Server.URL, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	remoteConfig, ok := remote.GetConfig()
	if !ok {
		return fmt.Errorf("while getting remote config for %s", config.CloudSlackCommPlatformIntegration)
	}

	req := &pb.ConnectRequest{
		InstanceId: remoteConfig.Identifier,
		BotId:      b.botID,
	}
	c, err := pb.NewCloudSlackClient(conn).Connect(ctx)
	if err != nil {
		return fmt.Errorf("while initializing gRPC cloud client: %w", err)
	}
	defer func(c pb.CloudSlack_ConnectClient) {
		err := c.CloseSend()
		if err != nil {
			b.log.Errorf("while closing connection: %s", err.Error())
		}
	}(c)

	err = c.Send(req)
	if err != nil {
		return fmt.Errorf("while sending gRPC connection request. %w", err)
	}

	go b.startMessageProcessor(ctx, messageWorkers, messages)

	for {
		data, err := c.Recv()
		if err != nil {
			if err == io.EOF {
				b.log.Warn("gRPC connection was closed by server")
				return errors.New("gRPC connection closed")
			}
			errStatus, ok := status.FromError(err)
			if ok && errStatus.Code() == codes.Canceled && errStatus.Message() == context.Canceled.Error() {
				b.log.Debugf("Context was cancelled. Skipping returning error...")
				return fmt.Errorf("while resolving error from gRPC response %s", errStatus.Err())
			}
			return fmt.Errorf("while receiving cloud slack events: %w", err)
		}
		messages <- data
	}
}

func (b *CloudSlack) startMessageProcessor(ctx context.Context, messageWorkers *pool.Pool, messages chan *pb.ConnectResponse) {
	b.log.Info("Starting cloud slack message processor...")
	defer b.log.Info("Stopped cloud slack message processor...")

	for msg := range messages {
		messageWorkers.Go(func() {
			err, _ := b.handleStreamMessage(ctx, msg)
			if err != nil {
				b.log.WithError(err).Error("Failed to handle Cloud Slack message")
			}
		})
	}
}

func (b *CloudSlack) shutdown(messageWorkers *pool.Pool, messages chan *pb.ConnectResponse) {
	b.log.Info("Shutting down cloud slack message processor...")
	close(messages)
	messageWorkers.Wait()
}

func (b *CloudSlack) handleStreamMessage(ctx context.Context, data *pb.ConnectResponse) (error, bool) {
	b.setFailureReason("")
	if streamingError := b.checkStreamingError(data.Event); pb.IsQuotaExceededErr(streamingError) {
		b.setFailureReason(FailureReasonQuotaExceeded)
		b.log.Warn(quotaExceededMsg)
		return nil, true
	}
	if len(data.Event) == 0 {
		return nil, false
	}
	event, err := slackevents.ParseEvent(data.Event, slackevents.OptionNoVerifyToken())
	if err != nil {
		return fmt.Errorf("while parsing event: %w", err), true
	}
	switch event.Type {
	case slackevents.CallbackEvent:
		b.log.Debugf("Got callback event %s", formatx.StructDumper().Sdump(event))
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			b.log.Debugf("Got app mention %s", formatx.StructDumper().Sdump(innerEvent))
			userName := b.getRealNameWithFallbackToUserID(ctx, ev.User)
			msg := slackMessage{
				Text:            ev.Text,
				Channel:         ev.Channel,
				ThreadTimeStamp: ev.ThreadTimeStamp,
				UserID:          ev.User,
				EventTimeStamp:  ev.EventTimeStamp,
				UserName:        userName,
				CommandOrigin:   command.TypedOrigin,
			}

			if err := b.handleMessage(ctx, msg); err != nil {
				b.log.Errorf("while handling message: %s", err.Error())
			}
		case *slackevents.MessageEvent:
			b.log.Debugf("Got generic message event %s", formatx.StructDumper().Sdump(innerEvent))
			msg := slackMessage{
				Text:           ev.Text,
				Channel:        ev.Channel,
				UserID:         ev.User,
				EventTimeStamp: ev.EventTimeStamp,
			}
			b.setFailureReason(FailureReasonQuotaExceeded)
			response := quotaExceeded()

			if err := b.send(ctx, msg, response); err != nil {
				return fmt.Errorf("while sending message: %w", err), true
			}
		}
	case string(slack.InteractionTypeBlockActions), string(slack.InteractionTypeViewSubmission):
		var callback slack.InteractionCallback
		err = json.Unmarshal(data.Event, &callback)
		if err != nil {
			b.log.Errorf("Invalid event %+v\n", data.Event)
			return fmt.Errorf("Invalid event %+v\n", data.Event), false
		}

		switch callback.Type {
		case slack.InteractionTypeBlockActions:
			b.log.Debugf("Got block action %s", formatx.StructDumper().Sdump(callback))

			if len(callback.ActionCallback.BlockActions) != 1 {
				b.log.Debug("Ignoring callback as the number of actions is different from 1")
				return nil, false
			}

			act := callback.ActionCallback.BlockActions[0]
			if act == nil || strings.HasPrefix(act.ActionID, urlButtonActionIDPrefix) {
				reportErr := b.reporter.ReportCommand(b.IntegrationName(), act.ActionID, command.ButtonClickOrigin, false)
				if reportErr != nil {
					b.log.Errorf("while reporting URL command, error: %s", reportErr.Error())
				}
				return nil, false // skip the url actions
			}

			channelID := callback.Channel.ID
			if channelID == "" && callback.View.ID != "" {
				// TODO: add support when we will need to handle button clicks from active modal.
				//
				// The request is coming from active modal, currently we don't support that.
				// We process that only when the modal is submitted (see slack.InteractionTypeViewSubmission action type).
				b.log.Debug("Ignoring callback as its source is an active modal")
				return nil, false
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
			if err := b.handleMessage(ctx, msg); err != nil {
				b.log.Errorf("Message handling error: %s", err.Error())
			}
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
						EventTimeStamp: "", // there is no timestamp for interactive callbacks
						CommandOrigin:  cmdOrigin,
					}

					if err := b.handleMessage(ctx, msg); err != nil {
						b.log.Errorf("Message handling error: %s", err.Error())
					}
				}
			}
		default:
			b.log.Debugf("get unhandled event %s", callback.Type)
		}
	}
	b.log.Debugf("received: %q\n", event)
	return nil, false
}

func (b *CloudSlack) SendMessage(ctx context.Context, msg interactive.CoreMessage, sourceBindings []string) error {
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

func (b *CloudSlack) SendMessageToAll(ctx context.Context, msg interactive.CoreMessage) error {
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

func (b *CloudSlack) handleMessage(ctx context.Context, event slackMessage) error {
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

	msgRef := b.msgStatusTracker.GetMsgRef(event)
	b.msgStatusTracker.MarkAsReceived(msgRef)

	response := e.Execute(ctx)
	err = b.send(ctx, event, response)
	if err != nil {
		return fmt.Errorf("while sending message: %w", err)
	}

	b.msgStatusTracker.MarkAsProcessed(msgRef)

	return nil
}

func (b *CloudSlack) send(ctx context.Context, event slackMessage, resp interactive.CoreMessage) error {
	b.log.Debugf("Sending message to channel %q: %+v", event.Channel, resp)

	resp.ReplaceBotNamePlaceholder(b.BotName(), api.BotNameWithClusterName(b.clusterName))
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

	// TODO: Currently, we don't get the channel ID once we use modal. This needs to be investigated and fixed.
	//
	// we can open modal only if we have a TriggerID (it's available when user clicks a button)
	//if resp.Type == api.PopupMessage && event.TriggerID != "" {
	//	modalView := b.renderer.RenderModal(resp)
	//	modalView.PrivateMetadata = event.Channel
	//	_, err := b.client.OpenViewContext(ctx, event.TriggerID, modalView)
	//	if err != nil {
	//		return fmt.Errorf("while opening modal: %w", err)
	//	}
	//	return nil
	//}

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

func (b *CloudSlack) getThreadOptionIfNeeded(event slackMessage, file *slack.File) slack.MsgOption {
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

func (b *CloudSlack) getChannelsToNotify(sourceBindings []string) []string {
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

func (b *CloudSlack) addStreamingClientCredentials() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		remoteCfg, ok := remote.GetConfig()
		if !ok {
			return nil, errors.New("empty remote configuration")
		}
		md := metadata.New(map[string]string{
			APIKeyContextKey:       remoteCfg.APIKey,
			DeploymentIDContextKey: remoteCfg.Identifier,
		})

		ctx = metadata.NewOutgoingContext(ctx, md)

		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}

		return clientStream, nil
	}
}

func (b *CloudSlack) addUnaryClientCredentials() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		remoteCfg, ok := remote.GetConfig()
		if !ok {
			return errors.New("empty remote configuration")
		}
		md := metadata.New(map[string]string{
			APIKeyContextKey:       remoteCfg.APIKey,
			DeploymentIDContextKey: remoteCfg.Identifier,
		})

		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (b *CloudSlack) checkStreamingError(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	cloudSlackErr := &pb.CloudSlackError{}
	if err := json.Unmarshal(data, cloudSlackErr); err != nil {
		return fmt.Errorf("while unmarshaling error: %w", err)
	}
	return cloudSlackErr
}

func quotaExceeded() interactive.CoreMessage {
	return interactive.CoreMessage{
		Header: "Quota exceeded",
		Message: api.Message{
			Sections: []api.Section{
				{
					Base: api.Base{
						Description: "You cannot use the Botkube Cloud Slack application within your plan. The command executions are blocked.",
					},
				},
			},
		},
	}
}

func (b *CloudSlack) setFailureReason(reason FailureReasonMsg) {
	if reason == "" {
		b.status = StatusHealthy
	} else {
		b.status = StatusUnHealthy
	}
	b.failureReason = reason
}

func (b *CloudSlack) GetStatus() Status {
	return Status{
		Status:   b.status,
		Restarts: fmt.Sprintf("%d/%d", b.failuresNo, b.maxRetries),
		Reason:   b.failureReason,
	}
}
