package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/botkube/internal/health"
	"github.com/kubeshop/botkube/pkg/api"
	pb "github.com/kubeshop/botkube/pkg/api/cloudteams"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

const (
	originKeyName            = "originName"
	teamsBotMentionPrefixFmt = "^<at>%s</at>"
)

// mdEmojiTag finds the emoji tags
var mdEmojiTag = regexp.MustCompile(`:(\w+):`)

var _ Bot = &CloudTeams{}

// CloudTeams listens for user's messages, execute commands and sends back the response.
// It sends also source notifications.
//
// User message (executors) flow:
//
//	+-------------+       +-------------+      +-------------+
//	|   MS Teams' | REST  |   Cloud     | REST |  Pub/Sub    |
//	|   message   +------>|   router    +----->|  #bot       |
//	+-------------+       +-------------+      +-------+-----+
//	                                                   v
//	+-------------+       +-------------+      +-------------+
//	|  Pub/Sub    | gRPC  |  Agent      | gRPC |  Cloud      |
//	|  #agent     |<------+  CloudTeams |<-----+  processor  |
//	+------+------+       +-------------+      +-------------+
//	       |
//	+------v------+       +-------------+
//	|  Cloud      | REST  |   MS Teams' |
//	|  processor  |+----->|   channel   |
//	+-------------+       +-------------+
//
// Notification (sources) flow:
//
//	+-------------+       +-------------+      +-------------+
//	|   Source    | gRPC  |  Agent      | gRPC |  Cloud      |
//	|   message   +------>|  CloudTeams +----->|  router     |
//	+-------------+       +-------------+      +-------+-----+
//	                                                   v
//	+-------------+       +-------------+      +-------------+
//	|  MS Teams'  | REST  |  Cloud      | REST |  Pub/Sub    |
//	|  channel    |<------+  processor  |<-----+  #agent     |
//	+------+------+       +-------------+      +-------------+
type CloudTeams struct {
	log                  logrus.FieldLogger
	cfg                  config.CloudTeams
	executorFactory      ExecutorFactory
	reporter             AnalyticsCommandReporter
	commGroupMetadata    CommGroupMetadata
	notifyMutex          sync.Mutex
	clusterName          string
	status               health.PlatformStatusMsg
	failuresNo           int
	failureReason        health.FailureReasonMsg
	errorMsg             string
	reportOnce           sync.Once
	botMentionRegex      *regexp.Regexp
	botName              string
	agentActivityMessage chan *pb.AgentActivity
	channelsMutex        sync.RWMutex
	channels             map[string]teamsCloudChannelConfigByID
}

// NewCloudTeams returns a new CloudTeams instance.
func NewCloudTeams(
	log logrus.FieldLogger,
	commGroupMetadata CommGroupMetadata,
	cfg config.CloudTeams,
	clusterName string,
	executorFactory ExecutorFactory,
	reporter AnalyticsCommandReporter) (*CloudTeams, error) {
	botMentionRegex, err := teamsBotMentionRegex(cfg.BotName)
	if err != nil {
		return nil, err
	}
	return &CloudTeams{
		log:                  log,
		executorFactory:      executorFactory,
		reporter:             reporter,
		cfg:                  cfg,
		botName:              cfg.BotName,
		channels:             teamsCloudChannelsConfig(cfg.Teams),
		commGroupMetadata:    commGroupMetadata,
		clusterName:          clusterName,
		botMentionRegex:      botMentionRegex,
		status:               health.StatusUnknown,
		agentActivityMessage: make(chan *pb.AgentActivity, platformMessageChannelSize),
	}, nil
}

// Start MS Teams server to serve messages from Teams client
func (b *CloudTeams) Start(ctx context.Context) error {
	return b.withRetries(ctx, b.log, maxRetries, func() error {
		return b.start(ctx)
	})
}

// SendMessageToAll sends the message to MS CloudTeams to all conversations even if notifications are disabled.
func (b *CloudTeams) SendMessageToAll(ctx context.Context, msg interactive.CoreMessage) error {
	return b.sendAgentActivity(ctx, msg, maps.Values(b.getChannels()))
}

// SendMessage sends the message to MS CloudTeams to selected conversations.
func (b *CloudTeams) SendMessage(ctx context.Context, msg interactive.CoreMessage, sourceBindings []string) error {
	return b.sendAgentActivity(ctx, msg, b.getChannelsToNotify(sourceBindings))
}

// IntegrationName describes the integration name.
func (b *CloudTeams) IntegrationName() config.CommPlatformIntegration {
	return config.CloudTeamsCommPlatformIntegration
}

// NotificationsEnabled returns current notification status for a given channel ID.
func (b *CloudTeams) NotificationsEnabled(channelID string) bool {
	channel, exists := b.getChannels()[channelID]
	if !exists {
		return false
	}

	return channel.notify
}

// SetNotificationsEnabled sets a new notification status for a given channel ID.
func (b *CloudTeams) SetNotificationsEnabled(channelID string, enabled bool) error {
	// avoid race conditions with using the setter concurrently, as we set a whole map
	b.notifyMutex.Lock()
	defer b.notifyMutex.Unlock()

	channels := b.getChannels()
	channel, exists := channels[channelID]
	if !exists {
		return execute.ErrNotificationsNotConfigured
	}

	channel.notify = enabled
	channels[channelID] = channel
	b.setChannels(channels)

	return nil
}

// Type describes the integration type.
func (b *CloudTeams) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// BotName returns the Bot name.
func (b *CloudTeams) BotName() string {
	return fmt.Sprintf("<at>%s</at>", b.botName)
}

// GetStatus gets bot status.
func (b *CloudTeams) GetStatus() health.PlatformStatus {
	return health.PlatformStatus{
		Status:   b.status,
		Restarts: fmt.Sprintf("%d/%d", b.failuresNo, maxRetries),
		Reason:   b.failureReason,
		ErrorMsg: b.errorMsg,
	}
}

func (b *CloudTeams) start(ctx context.Context) error {
	svc, err := newGrpcCloudTeamsConnector(b.log, b.cfg.Server)
	if err != nil {
		return fmt.Errorf("while creating gRPC connector: %w", err)
	}
	defer svc.Shutdown()

	err = svc.Start(ctx)
	if err != nil {
		return err
	}

	b.reportOnce.Do(func() {
		if err := b.reporter.ReportBotEnabled(b.IntegrationName(), b.commGroupMetadata.Index); err != nil {
			b.log.Errorf("report analytics error: %s", err.Error())
		}
	})
	b.failuresNo = 0 // Reset the failures to start exponential back-off from the beginning
	b.setFailureReason("", "")
	b.log.Info("Botkube connected to Cloud Teams!")

	parallel, ctx := errgroup.WithContext(ctx)
	parallel.Go(func() error {
		return svc.ProcessCloudActivity(ctx, b.handleStreamMessage)
	})
	parallel.Go(func() error {
		return svc.ProcessAgentActivity(ctx, b.agentActivityMessage)
	})

	return parallel.Wait()
}

func (b *CloudTeams) withRetries(ctx context.Context, log logrus.FieldLogger, maxRetries int, fn func() error) error {
	b.failuresNo = 0
	var lastFailureTimestamp time.Time
	resettableBackoff := func(n uint, err error, cfg *retry.Config) time.Duration {
		if !lastFailureTimestamp.IsZero() && time.Since(lastFailureTimestamp) >= successIntervalDuration {
			// if the last run was long enough, we treat is as success, so we reset failures
			log.Infof("Resetting failures counter as last failure was more than %s ago", successIntervalDuration)
			b.failuresNo = 0
			b.setFailureReason("", "")
		}
		lastFailureTimestamp = time.Now()
		b.failuresNo++
		b.setFailureReason(health.FailureReasonConnectionError, err.Error())

		return retry.BackOffDelay(uint(b.failuresNo), err, cfg)
	}
	return retry.Do(
		func() error {
			err := fn()
			if b.failuresNo >= maxRetries {
				b.setFailureReason(health.FailureReasonMaxRetriesExceeded, fmt.Sprintf("Reached max number of %d retries", maxRetries))
				log.Debugf("Reached max number of %d retries: %s", maxRetries, err)
				return retry.Unrecoverable(err)
			}
			return err
		},
		retry.OnRetry(func(_ uint, err error) {
			log.Warnf("Retrying Cloud Teams startup (attempt no %d/%d): %s", b.failuresNo, maxRetries, err)
		}),
		retry.DelayType(resettableBackoff),
		retry.Attempts(0), // infinite, we cancel that by our own
		retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
}

func (b *CloudTeams) handleStreamMessage(ctx context.Context, data *pb.CloudActivity) (*pb.AgentActivity, error) {
	b.setFailureReason("", "")
	var act schema.Activity
	err := json.Unmarshal(data.Event, &act)
	if err != nil {
		return nil, fmt.Errorf("while unmarshaling activity event: %w", err)
	}
	switch act.Type {
	case schema.Message, schema.Invoke:
		b.log.WithFields(logrus.Fields{
			"message":                 formatx.StructDumper().Sdump(act),
			"conversationDisplayName": data.ConversationDisplayName,
		}).Debug("Processing Cloud message...")
		channel, exists, err := b.getChannelForActivity(act)
		if err != nil {
			b.log.WithError(err).Error("cannot extract message channel id, processing with empty...")
		}
		msg := b.processMessage(ctx, act, channel, data.ConversationDisplayName, exists)
		if msg.IsEmpty() {
			b.log.WithField("activityID", act.ID).Debug("Empty message... Skipping sending response")
			return nil, nil
		}

		msg.ReplaceBotNamePlaceholder(b.BotName(), api.BotNameWithClusterName(b.clusterName))
		raw, err := json.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("while marshaling message to trasfer it via gRPC: %w", err)
		}

		conversationRef := activity.GetCoversationReference(act)
		return &pb.AgentActivity{
			Message: &pb.Message{
				MessageType:    pb.MessageType_MESSAGE_EXECUTOR,
				TeamId:         channel.teamID,
				ConversationId: conversationRef.Conversation.ID,
				Data:           raw,
			},
		}, nil
	default:
		return nil, fmt.Errorf("activity type %s not supported yet", act.Type)
	}
}

func (b *CloudTeams) processMessage(ctx context.Context, act schema.Activity, channel teamsCloudChannelConfigByID, channelDisplayName string, exists bool) interactive.CoreMessage {
	trimmedMsg := b.trimBotMention(act.Text)

	e := b.executorFactory.NewDefault(execute.NewDefaultInput{
		CommGroupName:   b.commGroupMetadata.Name,
		Platform:        b.IntegrationName(),
		NotifierHandler: b,
		Conversation: execute.Conversation{
			Alias:            channel.alias,
			IsKnown:          exists,
			ID:               channel.Identifier(),
			ExecutorBindings: channel.Bindings.Executors,
			SourceBindings:   channel.Bindings.Sources,
			CommandOrigin:    b.mapToCommandOrigin(act),
			DisplayName:      channelDisplayName,
			ParentActivityID: act.Conversation.ID,
		},
		Message: trimmedMsg,
		User: execute.UserInput{
			// Note: this is a plain text mention, a native mentions will be provided as a part of:
			// https://github.com/kubeshop/botkube/issues/1331
			Mention:     act.From.Name,
			DisplayName: act.From.Name,
		},
		AuditContext: act.ChannelData,
	})
	return e.Execute(ctx)
}

// generate a function code tab based on in type will return proper command origin.
func (b *CloudTeams) mapToCommandOrigin(act schema.Activity) command.Origin {
	// in the newer Botkube Cloud version, the origin is explicitly set
	c, found := b.extractExplicitOrigin(act)
	if found {
		return c
	}

	// fallback to default strategy
	switch act.Type {
	case schema.Message:
		return command.TypedOrigin
	case schema.Invoke:
		return command.ButtonClickOrigin
	default:
		return command.UnknownOrigin
	}
}

type activityMetadata struct {
	// OriginName is inlined for e.g. 'Action.Submit'. We are not able to force the unified approach.
	OriginName string `mapstructure:"originName"`
	Action     struct {
		Data struct {
			// OriginName is nested for e.g. 'Action.Execute'.
			OriginName string `mapstructure:"originName"`
		} `mapstructure:"data"`
	} `mapstructure:"action"`
}

func (b *CloudTeams) extractExplicitOrigin(act schema.Activity) (command.Origin, bool) {
	if act.Value == nil {
		return "", false
	}
	var data activityMetadata
	err := mapstructure.Decode(act.Value, &data)
	if err != nil {
		b.log.WithError(err).Debug("Cannot decode activity value to extract command origin")
		return "", false
	}

	origin := data.OriginName
	if origin == "" {
		origin = data.Action.Data.OriginName
	}
	if command.IsValidOrigin(origin) {
		return command.Origin(origin), true
	}
	return "", false
}

func (b *CloudTeams) sendAgentActivity(ctx context.Context, msg interactive.CoreMessage, channels []teamsCloudChannelConfigByID) error {
	errs := multierror.New()
	for _, channel := range channels {
		b.log.Debugf("Sending message to channel %q: %+v", channel.ID, msg)

		msg.ReplaceBotNamePlaceholder(b.BotName(), api.BotNameWithClusterName(b.clusterName))
		raw, err := json.Marshal(msg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while proxing message via agent for channel id %q: %w", channel.ID, err))
			continue
		}

		act := &pb.AgentActivity{
			Message: &pb.Message{
				MessageType:    pb.MessageType_MESSAGE_SOURCE,
				TeamId:         channel.teamID,
				ConversationId: channel.ID,
				Data:           raw,
			},
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case b.agentActivityMessage <- act:
		}
	}
	return errs.ErrorOrNil()
}

type channelData struct {
	Channel struct {
		ID string `mapstructure:"id"`
	} `mapstructure:"channel"`
	TeamsChannelID string `mapstructure:"teamsChannelId"`
}

func (b *CloudTeams) getChannelForActivity(act schema.Activity) (teamsCloudChannelConfigByID, bool, error) {
	var data channelData
	err := mapstructure.Decode(act.ChannelData, &data)
	if err != nil {
		return teamsCloudChannelConfigByID{}, false, fmt.Errorf("while decoding data: %w", err)
	}

	if data.Channel.ID == "" && data.TeamsChannelID == "" {
		return teamsCloudChannelConfigByID{}, false, fmt.Errorf("cannot find channel id in: %s", formatx.StructDumper().Sdump(act.ChannelData))
	}

	id := data.TeamsChannelID
	if id == "" {
		id = data.Channel.ID
	}

	channel, exists := b.getChannels()[id]
	return channel, exists, nil
}

func (b *CloudTeams) getChannelsToNotify(sourceBindings []string) []teamsCloudChannelConfigByID {
	var out []teamsCloudChannelConfigByID
	for _, cfg := range b.getChannels() {
		if !cfg.notify {
			b.log.Infof("Skipping notification for channel %q as notifications are disabled.", cfg.Identifier())
			continue
		}

		if sourceBindings != nil && !sliceutil.Intersect(sourceBindings, cfg.Bindings.Sources) {
			continue
		}

		out = append(out, cfg)
	}
	return out
}

func (b *CloudTeams) getChannels() map[string]teamsCloudChannelConfigByID {
	b.channelsMutex.RLock()
	defer b.channelsMutex.RUnlock()
	return b.channels
}

func (b *CloudTeams) setChannels(channels map[string]teamsCloudChannelConfigByID) {
	b.channelsMutex.Lock()
	defer b.channelsMutex.Unlock()
	b.channels = channels
}

func (b *CloudTeams) trimBotMention(msg string) string {
	msg = strings.TrimSpace(msg)
	return b.botMentionRegex.ReplaceAllString(msg, "")
}

func (b *CloudTeams) setFailureReason(reason health.FailureReasonMsg, errorMsg string) {
	if reason == "" {
		b.status = health.StatusHealthy
	} else {
		b.status = health.StatusUnHealthy
	}
	b.failureReason = reason
	b.errorMsg = errorMsg
}

type teamsCloudChannelConfigByID struct {
	config.ChannelBindingsByID
	alias  string
	notify bool
	teamID string
}

func teamsCloudChannelsConfig(teams []config.TeamsBindings) map[string]teamsCloudChannelConfigByID {
	out := make(map[string]teamsCloudChannelConfigByID)
	for _, team := range teams {
		for alias, channel := range team.Channels {
			out[channel.Identifier()] = teamsCloudChannelConfigByID{
				ChannelBindingsByID: channel,
				alias:               alias,
				notify:              !channel.Notification.Disabled,
				teamID:              team.ID,
			}
		}
	}
	return out
}

func teamsBotMentionRegex(botName string) (*regexp.Regexp, error) {
	botMentionRegex, err := regexp.Compile(fmt.Sprintf(teamsBotMentionPrefixFmt, botName))
	if err != nil {
		return nil, fmt.Errorf("while compiling bot mention regex: %w", err)
	}

	return botMentionRegex, nil
}

// replaceEmojiTagsWithActualOne replaces the emoji tag with actual emoji.
func replaceEmojiTagsWithActualOne(content string) string {
	return mdEmojiTag.ReplaceAllStringFunc(content, func(s string) string {
		return emojiMapping[s]
	})
}

// emojiMapping holds mapping between emoji tags and actual ones.
var emojiMapping = map[string]string{
	":rocket:":                  "🚀",
	":warning:":                 "⚠️",
	":white_check_mark:":        "✅",
	":arrows_counterclockwise:": "🔄",
	":exclamation:":             "❗",
	":cricket:":                 "🦗",
	":no_entry_sign:":           "🚫",
	":large_green_circle:":      "🟢",
	":new:":                     "🆕",
}
