package commplatform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"botkube.io/botube/test/diff"
	"botkube.io/botube/test/msteamsx"
	gcppubsub "cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/pkg/pubsub"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/pkg/teamsx"
	"github.com/kubeshop/botkube/pkg/api"
	pb "github.com/kubeshop/botkube/pkg/api/cloudteams"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/kubeshop/botkube/pkg/ptr"
	"github.com/markbates/errx"
	"github.com/nsf/jsondiff"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	platformMessageWorkersCount = 10
	platformMessageChannelSize  = 100
	devTeamsBotEventsTopicName  = "dev.teams_router.teams_events"
	// serviceURL is a generic URL used when you don't yet have the ServiceURL from the received Activity. For more information, see:
	// https://learn.microsoft.com/en-us/azure/bot-service/rest-api/bot-framework-rest-connector-api-reference?view=azure-bot-service-4.0#base-uri
	serviceURL = "https://smba.trafficmanager.net/teams/"
	// channelID serves as namespaces for other platforms, for example, 'slack'.
	// more info: https://learn.microsoft.com/en-us/azure/bot-service/bot-service-resources-identifiers-guide?view=azure-bot-service-4.0#channel-id
	channelID             = "msteams"
	lineLimitToShowFilter = 16
)

type TeamsConfig struct {
	AdditionalContextMessage string        `envconfig:"optional"`
	RecentMessagesLimit      int           `envconfig:"default=6"`
	MessageWaitTimeout       time.Duration `envconfig:"default=50s"`

	BotDevName string `envconfig:"default=BotkubeDev"`

	BotTesterName        string `envconfig:"default=BotkubeTester"`
	BotTesterAppID       string
	BotTesterAppPassword string

	OrganizationTenantID string
	OrganizationTeamID   string
}

type TeamsChannel struct {
	id   string
	name string
}

func (s *TeamsChannel) ID() string {
	return s.id
}
func (s *TeamsChannel) Name() string {
	return s.name
}
func (s *TeamsChannel) Identifier() string {
	return s.id
}

type TeamsTester struct {
	cli                  *msteamsx.Client
	cfg                  TeamsConfig
	firstChannel         Channel
	secondChannel        Channel
	thirdChannel         Channel
	configProviderApiKey string
	agentActivityMessage chan *pb.AgentActivity
	renderer             *teamsx.MessageRendererAdapter
	pubSubClient         *pubsub.Client
}

func (s *TeamsTester) ReplaceBotNamePlaceholder(msg *interactive.CoreMessage, clusterName string) {
	msg.ReplaceBotNamePlaceholder(s.cfg.BotDevName, api.BotNameWithClusterName(clusterName))
}

func NewTeamsTester(teamsCfg TeamsConfig, apiKey *string) (*TeamsTester, error) {
	teamsCli, err := msteamsx.New(teamsCfg.BotTesterAppID, teamsCfg.BotTesterAppPassword, teamsCfg.OrganizationTenantID)
	if err != nil {
		return nil, err
	}

	pubSubClient, err := pubsub.New(context.Background(), false)

	return &TeamsTester{
		cli:                  teamsCli,
		cfg:                  teamsCfg,
		pubSubClient:         pubSubClient,
		configProviderApiKey: ptr.ToValue(apiKey),
		renderer:             teamsx.NewMessageRendererAdapter(loggerx.NewNoop(), teamsCfg.BotTesterAppID, teamsCfg.BotDevName),
		agentActivityMessage: make(chan *pb.AgentActivity, platformMessageChannelSize),
	}, nil
}

// Shutdown performs the shutdown of the dispatcher.
func (s *TeamsTester) Shutdown() error {
	//s.log.Info("Shutting down event dispatcher...")
	err := s.pubSubClient.Instance.Close()
	if err != nil {
		return errx.Wrap(err, "while closing pub/sub instance")
	}
	return nil
}

// AgentEvent is the event being sent by Agent either as a new notification or executor response.
// This is used in processor, passed through Pub/Sub.
type AgentEvent struct {
	Message    *pb.Message `json:"message,omitempty"`
	InstanceID string      `json:"instanceID,omitempty"`
}

// publishBotActivityIntoPubSub puts a given event into queue.
func (s *TeamsTester) publishBotActivityIntoPubSub(t *testing.T, ctx context.Context, event schema.Activity) error {
	t.Helper()

	out, err := json.Marshal(event)
	if err != nil {
		return err
	}

	t.Logf("Publishing event: %s", string(out))

	res := s.pubSubClient.Instance.Topic(devTeamsBotEventsTopicName).Publish(ctx, &gcppubsub.Message{
		Data: out,
	})
	if _, err := res.Get(ctx); err != nil {
		return errx.Wrap(err, "failed to publish bot Teams' event")
	}
	return nil
}

func (s *TeamsTester) InitUsers(t *testing.T) {
	t.Helper()
	t.Log("No need to init users for Teams, skipping...")
}

func (s *TeamsTester) InitChannels(t *testing.T) []func() {
	channels, err := s.cli.GetChannels(context.Background(), s.cfg.OrganizationTeamID)
	require.NoError(t, err)
	for _, i := range channels {
		err := s.cli.DeleteChannel(context.Background(), s.cfg.OrganizationTeamID, i)
		require.NoError(t, err)
	}

	firstChannel, cleanupFirstChannelFn := s.CreateChannel(t, "first")
	s.firstChannel = firstChannel

	secondChannel, cleanupSecondChannelFn := s.CreateChannel(t, "second")
	s.secondChannel = secondChannel

	thirdChannel, cleanupThirdChannelFn := s.CreateChannel(t, "rbac")
	s.thirdChannel = thirdChannel

	return []func(){
		func() { cleanupFirstChannelFn(t) },
		func() { cleanupSecondChannelFn(t) },
		func() { cleanupThirdChannelFn(t) },
	}
}

func (s *TeamsTester) Type() DriverType {
	return TeamsBot
}

func (s *TeamsTester) BotName() string {
	return fmt.Sprintf("<@%s>", s.cfg.BotDevName)
}

func (s *TeamsTester) BotUserID() string {
	return s.cfg.BotDevName
}

func (s *TeamsTester) TesterUserID() string {
	return s.cfg.BotTesterName
}

func (s *TeamsTester) Channel() Channel {
	return s.firstChannel
}

func (s *TeamsTester) SecondChannel() Channel {
	return s.secondChannel
}

func (s *TeamsTester) ThirdChannel() Channel {
	return s.thirdChannel
}

func (s *TeamsTester) MDFormatter() interactive.MDFormatter {
	return s.renderer.MDFormatter()
}

func (s *TeamsTester) PostInitialMessage(t *testing.T, channelName string) {
	t.Helper()
	t.Log("Posting welcome message...")

	var additionalContextMsg string
	if s.cfg.AdditionalContextMessage != "" {
		additionalContextMsg = fmt.Sprintf("%s\n", s.cfg.AdditionalContextMessage)
	}
	message := fmt.Sprintf("Hello!\n%s%s", additionalContextMsg, welcomeText)
	err := s.cli.SendMessage(context.Background(), channelName, message)
	require.NoError(t, err)
}

func (s *TeamsTester) PostMessageToBot(t *testing.T, channel, command string) {
	ctx := context.Background()
	msgText := fmt.Sprintf("<at>%s</at> %s", s.cfg.BotDevName, command)
	activity := schema.Activity{
		Type:       schema.Message,
		ServiceURL: serviceURL,
		ChannelID:  channelID,
		Conversation: schema.ConversationAccount{
			ID: channel,
		},
		Text: msgText,
		ChannelData: map[string]any{
			"channel": map[string]string{
				"id": channel,
			},
			"team": map[string]string{
				"id": "19:R0qJu_rs0Ib3ceRjQ_UkwUXXOXcVfQmf5ZlV21v8L741@thread.tacv2",
			},
		},
		From: schema.ChannelAccount{
			ID:   fmt.Sprintf("28:%s", s.cfg.BotTesterAppID),
			Name: s.cfg.BotTesterName,
		},
	}

	//  message
	err := s.cli.SendMessage(ctx, channel, msgText)
	assert.NoError(t, err)
	err = s.publishBotActivityIntoPubSub(t, ctx, activity)
	assert.NoError(t, err)
}

func (s *TeamsTester) InviteBotToChannel(t *testing.T, channelID string) {
	t.Logf("No need to invite bot for channel %q since bot is added in Team level...", channelID)
}

// FIXME: Valid ones

func (s *TeamsTester) WaitForMessagePostedRecentlyEqual(userID, channelID, expectedMsg string) error {
	// TODO: unify with InteractivePosted
	msg := api.NewPlaintextMessage(expectedMsg, false)
	_, card, _ := s.renderer.RenderCoreMessageCardAndOptions(interactive.CoreMessage{Message: msg}, s.cfg.BotDevName)
	card.MsTeams.Entities = nil

	expMsg, err := json.Marshal(card)
	if err != nil {
		return err
	}
	opts := jsondiff.DefaultConsoleOptions()
	opts.SkipMatches = true
	return s.WaitForInteractiveMessagePosted(userID, channelID, s.cfg.RecentMessagesLimit, func(msg string) (bool, int, string) {
		gotMsg := strings.NewReplacer(`<at id=\"0\">`, "", "<at>", "", "</at>", "").Replace(msg)
		ok, msgDiff := jsondiff.Compare(expMsg, []byte(gotMsg), ptr.FromType(opts))
		if ok != jsondiff.FullMatch {
			return false, 1, msgDiff
		}

		return true, 0, ""
	})
}

func (s *TeamsTester) WaitForLastMessageContains(userID, channelID, expectedMsgSubstring string) error {
	return s.WaitForInteractiveMessagePosted(userID, channelID, 1, func(msg string) (bool, int, string) {
		msg, expectedMsgSubstring = NormalizeTeamsWhitespacesInMessages(msg, expectedMsgSubstring)
		return strings.Contains(msg, expectedMsgSubstring), 0, ""
	})
}

// NormalizeTeamsWhitespacesInMessages normalizes messages, as the Teams renderer uses different line breaks in order to make the message
// more readable. It's hard to come up with a single message that matches all our communication platforms so
// this makes sure that we're normalizing the message to a single line break.
//
// We can consider enchantment in the future, and replace the expectedMsg string with api.Message to allow using dedicated MD renderers in each platform.
func NormalizeTeamsWhitespacesInMessages(got, exp string) (string, string) {
	got = strings.ReplaceAll(got, "\n\n", "\n")
	got = strings.ReplaceAll(got, "\n\n\n", "\n")

	exp = strings.ReplaceAll(exp, "\n\n", "\n")
	exp = strings.ReplaceAll(exp, "\n\n\n", "\n")
	return got, exp
}

func (s *TeamsTester) WaitForLastMessageEqual(userID, channelID, expectedMsg string) error {
	limitMessages := 1
	if len(strings.Split(expectedMsg, "\n")) > lineLimitToShowFilter {
		limitMessages = 2 // messages with filter are split into 2, so we need to fetch one more message to get body
	}

	return s.WaitForInteractiveMessagePosted(userID, channelID, limitMessages, func(msg string) (bool, int, string) {
		msg, expectedMsg = NormalizeTeamsWhitespacesInMessages(msg, expectedMsg)

		if msg != expectedMsg {
			count := diff.CountMatchBlock(expectedMsg, msg)
			msgDiff := diff.Diff(expectedMsg, msg)
			return false, count, msgDiff
		}
		return true, 0, ""
	})
}

func (s *TeamsTester) WaitForMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	return s.WaitForInteractiveMessagePosted(userID, channelID, limitMessages, assertFn)
}

func (s *TeamsTester) WaitForInteractiveMessagePosted(userID, channelID string, limitMessages int, assertFn MessageAssertion) error {
	var (
		fetchedMessages []msteamsx.MsTeamsMessage
		diffMessage     string
		lastErr         error
	)
	var highestCommonBlockCount int
	if limitMessages == 1 {
		highestCommonBlockCount = -1 // a single message is fetched, always print diff
	}

	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, s.cfg.MessageWaitTimeout, true, func(ctx context.Context) (done bool, err error) {
		fetchedMessages, err = s.cli.GetMessages(ctx, s.cfg.OrganizationTeamID, channelID, limitMessages)
		if err != nil {
			lastErr = err
			return false, nil
		}

		for _, msg := range fetchedMessages {
			if !strings.EqualFold(ptr.ToValue(msg.Raw.GetFrom().GetApplication().GetDisplayName()), userID) {
				continue
			}

			equal, commonCount, diffStr := assertFn(msg.Rendered)
			if !equal {
				// different message; update the diff if it's more similar than the previous one or initial value
				if commonCount > highestCommonBlockCount {
					highestCommonBlockCount = commonCount
					diffMessage = diffStr
				}
				continue
			}

			return true, nil
		}

		return false, nil
	})
	if lastErr == nil {
		lastErr = fmt.Errorf("message assertion function returned false%s", diffMessage)
	}
	if err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

// FIXME: Valid ones  -- end

func (s *TeamsTester) WaitForMessagePostedWithFileUpload(userID, channelID string, assertFn FileUploadAssertion) error {
	var fetchedMessages []slack.Message
	var lastErr error
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, s.cfg.MessageWaitTimeout, false, func(ctx context.Context) (done bool, err error) {
		/*	historyRes, err := s.cli.GetConversationHistory(&slack.GetConversationHistoryParameters{
				ChannelID: channelID, Limit: 1,
			})
			if err != nil {
				lastErr = err
				return false, nil
			}

			fetchedMessages = historyRes.Messages
			for _, msg := range historyRes.Messages {
				if msg.User != userID {
					continue
				}

				if len(msg.Files) != 1 {
					return false, nil
				}

				upload := msg.Files[0]
				if !assertFn(upload.Title, upload.Mimetype) {
					// different message
					return false, nil
				}

				return true, nil
			}*/

		return false, nil
	})
	if lastErr == nil {
		lastErr = errors.New("message assertion function returned false")
	}
	if err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("while waiting for condition: last error: %w; fetched messages: %s", lastErr, structDumper.Sdump(fetchedMessages))
		}
		return err
	}

	return nil
}

func (s *TeamsTester) WaitForMessagePostedWithAttachment(userID, channelID string, limitMessages int, expAttachment ExpAttachmentInput) error {
	// for now we don't compare times
	expAttachment.Message.Timestamp = time.Time{}

	_, card, _ := s.renderer.RenderCoreMessageCardAndOptions(interactive.CoreMessage{Message: expAttachment.Message}, s.cfg.BotDevName)
	card.MsTeams.Entities = nil

	expMsg, err := json.Marshal(card)
	if err != nil {
		return err
	}
	opts := jsondiff.DefaultConsoleOptions()
	opts.SkipMatches = true
	return s.WaitForInteractiveMessagePosted(userID, channelID, limitMessages, func(msg string) (bool, int, string) {
		gotMsg := strings.NewReplacer(`<at id=\"0\">`, "", "</at>", "", "<at>", "").Replace(msg)
		gotMsg, err := filterDatesObjects(gotMsg)
		if err != nil {
			return false, 1, err.Error()
		}
		ok, msgDiff := jsondiff.Compare(expMsg, []byte(gotMsg), ptr.FromType(opts))
		switch ok {
		// SupersetMatch is used as sometimes we sent more details than is returned by Teams API, e.g.:
		// we sent:
		// 				{
		//					"type": "TableColumnDefinition",
		//					"width": 1,
		//					"horizontalCellContentAlignment": "left",
		//					"verticalCellContentAlignment": "bottom"
		//				}
		// while API returns:
		// 				{
		//					"verticalCellContentAlignment": "bottom",
		//					"width": 1
		//				}
		case jsondiff.FullMatch, jsondiff.SupersetMatch:
			return true, 0, ""
		default:
			return false, 1, msgDiff
		}
	})
}

func (s *TeamsTester) WaitForInteractiveMessagePostedRecentlyEqual(userID, channelID string, msg interactive.CoreMessage) error {
	return s.waitForAdaptiveCardMessage(userID, channelID, s.cfg.RecentMessagesLimit, msg)
}

func (s *TeamsTester) WaitForLastInteractiveMessagePostedEqual(userID, channelID string, msg interactive.CoreMessage) error {
	return s.waitForAdaptiveCardMessage(userID, channelID, 1, msg)
}

func (s *TeamsTester) WaitForLastInteractiveMessagePostedEqualWithCustomRender(_, _, _ string) error {
	return errors.New("not implemented")
}

func (s *TeamsTester) SetTimeout(timeout time.Duration) {
	s.cfg.MessageWaitTimeout = timeout
}

func (s *TeamsTester) Timeout() time.Duration {
	return s.cfg.MessageWaitTimeout
}

func (s *TeamsTester) CreateChannel(t *testing.T, prefix string) (Channel, func(t *testing.T)) {
	t.Helper()
	randomID := uuid.New()
	channelName := fmt.Sprintf("%s-%s-%s", channelNamePrefix, prefix, randomID.String())

	t.Logf("Creating channel %q...", channelName)
	ctx := context.Background()
	channelID, err := s.cli.CreateChannel(ctx, s.cfg.OrganizationTeamID, channelName)
	require.NoError(t, err)

	t.Logf("Channel %q (ID: %q) created", channelName, channelID)

	cleanupFn := func(t *testing.T) {
		t.Helper()
		t.Logf("Archiving channel %q...", channelName)
		err = s.cli.DeleteChannel(ctx, s.cfg.OrganizationTeamID, channelID)
		assert.NoError(t, err)
	}

	return &TeamsChannel{id: channelID, name: channelName}, cleanupFn
}

// private

func (s *TeamsTester) waitForAdaptiveCardMessage(userID, channelID string, limitMessages int, msg interactive.CoreMessage) error {
	_, card, _ := s.renderer.RenderCoreMessageCardAndOptions(msg, s.cfg.BotDevName)
	card.MsTeams.Entities = nil

	expMsg, err := json.Marshal(card)
	if err != nil {
		return err
	}
	opts := jsondiff.DefaultConsoleOptions()
	opts.SkipMatches = true
	return s.WaitForInteractiveMessagePosted(userID, channelID, limitMessages, func(msg string) (bool, int, string) {
		gotMsg := strings.NewReplacer(`<at id=\"0\">`, "", "</at>", "", "<at>", "").Replace(msg)

		ok, msgDiff := jsondiff.Compare(expMsg, []byte(gotMsg), ptr.FromType(opts))
		switch ok {
		// SupersetMatch is used as sometimes we sent more details than is returned by Teams API, e.g.:
		// we send:
		// 				{
		//					"type": "TableColumnDefinition",
		//					"width": 1,
		//					"horizontalCellContentAlignment": "left",
		//					"verticalCellContentAlignment": "bottom"
		//				}
		// while API returns:
		// 				{
		//					"verticalCellContentAlignment": "bottom",
		//					"width": 1
		//				}
		case jsondiff.FullMatch, jsondiff.SupersetMatch:
			return true, 0, ""
		default:
			return false, 1, msgDiff
		}
	})
}

func filterDatesObjects(adaptiveCard string) (string, error) {
	var event map[string]any
	err := json.Unmarshal([]byte(adaptiveCard), &event)
	if err != nil {
		return adaptiveCard, err
	}

	body := event["body"].([]any)
	var keep []any
	for _, item := range body {
		if isDateOrActions(item) {
			continue
		}
		keep = append(keep, item)
	}

	event["body"] = keep
	out, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func isDateOrActions(in any) bool {
	obj, ok := in.(map[string]any)
	if !ok {
		return false
	}
	objType, objTypeFound := obj["type"]
	objText, objTextFound := obj["text"]

	hasDate := objTextFound && objText.(string) != "" && strings.HasPrefix(objText.(string), "_{{DATE(")
	isActionSet := objTypeFound && objType.(string) == "ActionSet"

	return hasDate || isActionSet
}
