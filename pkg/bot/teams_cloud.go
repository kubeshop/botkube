package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/sirupsen/logrus"
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

const channelIDKeyName = "teamsChannelId"

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

// SendMessageToAll sends the message to MS CloudTeams to all conversations.
func (b *CloudTeams) SendMessageToAll(ctx context.Context, msg interactive.CoreMessage) error {
	return b.sendAgentActivity(ctx, msg, b.getChannelsToNotify(nil))
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
	}
}

func (b *CloudTeams) start(ctx context.Context) error {
	svc, err := newGrpcCloudTeamsConnector(b.log, b.cfg.Server.URL)
	if err != nil {
		return fmt.Errorf("while creating gRPC connector")
	}
	defer svc.Shutdown()

	err = svc.Start(ctx)
	if err != nil {
		return err
	}

	b.setFailureReason("")
	b.reportOnce.Do(func() {
		if err := b.reporter.ReportBotEnabled(b.IntegrationName(), b.commGroupMetadata.Index); err != nil {
			b.log.Errorf("report analytics error: %s", err.Error())
		}
		b.log.Info("Botkube connected to Cloud Teams!")
	})

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
	return retry.Do(
		func() error {
			err := fn()
			if err != nil {
				if !lastFailureTimestamp.IsZero() && time.Since(lastFailureTimestamp) >= successIntervalDuration {
					// if the last run was long enough, we treat is as success, so we reset failures
					log.Infof("Resetting failures counter as last failure was more than %s ago", successIntervalDuration)
					b.failuresNo = 0
				}

				if b.failuresNo >= maxRetries {
					b.setFailureReason(health.FailureReasonMaxRetriesExceeded)
					log.Debugf("Reached max number of %d retries: %s", maxRetries, err)
					return retry.Unrecoverable(err)
				}

				lastFailureTimestamp = time.Now()
				b.failuresNo++
				b.setFailureReason(health.FailureReasonConnectionError)
				return err
			}
			b.setFailureReason("")
			return nil
		},
		retry.OnRetry(func(_ uint, err error) {
			log.Warnf("Retrying Cloud Teams startup (attempt no %d/%d): %s", b.failuresNo, maxRetries, err)
		}),
		retry.Delay(retryDelay),
		retry.Attempts(0), // infinite, we cancel that by our own
		retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
}

func (b *CloudTeams) handleStreamMessage(ctx context.Context, data *pb.CloudActivity) (*pb.AgentActivity, error) {
	b.setFailureReason("")
	var act schema.Activity
	err := json.Unmarshal(data.Event, &act)
	if err != nil {
		return nil, fmt.Errorf("while unmarshaling activity event: %w", err)
	}
	switch act.Type {
	case schema.Message:
		b.log.WithField("message", formatx.StructDumper().Sdump(act)).Debug("Processing Cloud message...")
		channel, exists := b.getChannelForActivity(act)

		msg := b.processMessage(ctx, act, channel, exists)
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
				TeamId:         channel.teamID,
				ConversationId: conversationRef.Conversation.ID,
				ActivityId:     conversationRef.ActivityID, // activity ID allows us to send it as a thread message
				Data:           raw,
			},
		}, nil
	default:
		return nil, fmt.Errorf("activity type %s not supported yet", act.Type)
	}
}

func (b *CloudTeams) processMessage(ctx context.Context, act schema.Activity, channel teamsCloudChannelConfigByID, exists bool) interactive.CoreMessage {
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
			CommandOrigin:    command.TypedOrigin,
		},
		Message: trimmedMsg,
		User: execute.UserInput{
			//Mention:     "", // TODO(https://github.com/kubeshop/botkube-cloud/issues/677): set when adding interactivity support.
			DisplayName: act.From.Name,
		},
	})
	return e.Execute(ctx)
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
				TeamId:         channel.teamID,
				ActivityId:     "", // empty so it will be sent on root instead of sending as a thread message
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

func (b *CloudTeams) getChannelForActivity(act schema.Activity) (teamsCloudChannelConfigByID, bool) {
	rawChannelID, exists := act.ChannelData[channelIDKeyName]
	if !exists {
		return teamsCloudChannelConfigByID{}, false
	}

	channelID, ok := rawChannelID.(string)
	if !ok {
		return teamsCloudChannelConfigByID{}, false
	}

	channel, exists := b.getChannels()[channelID]
	return channel, exists
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
	return b.botMentionRegex.ReplaceAllString(msg, "")
}

func (b *CloudTeams) setFailureReason(reason health.FailureReasonMsg) {
	if reason == "" {
		b.status = health.StatusHealthy
	} else {
		b.status = health.StatusUnHealthy
	}
	b.failureReason = reason
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
