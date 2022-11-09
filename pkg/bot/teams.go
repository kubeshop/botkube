package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"github.com/gorilla/mux"
	"github.com/infracloudio/msbotbuilder-go/core"
	coreActivity "github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/httpsrv"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

// TODO: Refactor this file as a part of https://github.com/kubeshop/botkube/issues/667
//  - see if we cannot set conversation ref without waiting for `@Botkube notify start` message.
//  - support source and executor bindings per channel
//  - see if a public endpoint can be avoided to handle Teams messages.
//  - see if we can use different library
//  - split to multiple files in a separate package,
//  - review all the methods and see if they can be simplified.

const (
	defaultPort        = "3978"
	longRespNotice     = "Response is too long. Sending last few lines. Please send DM to Botkube to get complete response."
	convTypePersonal   = "personal"
	contentTypeCard    = "application/vnd.microsoft.card.adaptive"
	contentTypeFile    = "application/vnd.microsoft.teams.card.file.consent"
	responseFileName   = "response.txt"
	activityFileUpload = "fileUpload"
	activityAccept     = "accept"
	activityUploadInfo = "uploadInfo"

	// teamsMaxMessageSize max size before a message should be uploaded as a file.
	teamsMaxMessageSize = 15700
)

var _ Bot = &Teams{}

const teamsBotMentionPrefixFmt = "^<at>%s</at>"

// mdEmojiTag finds the emoji tags
var mdEmojiTag = regexp.MustCompile(`:(\w+):`)

type conversation struct {
	ref    schema.ConversationReference
	notify bool
}

// Teams listens for user's message, execute commands and sends back the response.
type Teams struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory
	reporter        AnalyticsReporter
	// TODO: Be consistent with other communicators when Teams supports multiple channels
	//channels map[string][ChannelBindingsByName]
	bindings           config.BotBindings
	conversationsMutex sync.RWMutex
	commGroupName      string
	conversations      map[string]conversation
	notifyMutex        sync.Mutex
	botMentionRegex    *regexp.Regexp
	longFormatter      interactive.MDFormatter
	shortFormatter     interactive.MDFormatter

	botName      string
	AppID        string
	AppPassword  string
	MessagePath  string
	Port         string
	ClusterName  string
	Notification config.Notification
	Adapter      core.Adapter
}

type consentContext struct {
	Command string
}

// NewTeams creates a new Teams instance.
func NewTeams(log logrus.FieldLogger, commGroupName string, cfg config.Teams, clusterName string, executorFactory ExecutorFactory, reporter AnalyticsReporter) (*Teams, error) {
	botMentionRegex, err := teamsBotMentionRegex(cfg.BotName)
	if err != nil {
		return nil, err
	}

	port := cfg.Port
	if port == "" {
		port = defaultPort
	}
	msgPath := cfg.MessagePath
	if msgPath == "" {
		msgPath = "/"
	}

	longFormatter := interactive.NewMDFormatter(longLineFormatter, interactive.MdHeaderFormatter)
	shortFormatter := interactive.NewMDFormatter(shortLineFormatter, interactive.MdHeaderFormatter)

	return &Teams{
		log:             log,
		executorFactory: executorFactory,
		reporter:        reporter,
		botName:         cfg.BotName,
		ClusterName:     clusterName,
		AppID:           cfg.AppID,
		AppPassword:     cfg.AppPassword,
		Notification:    cfg.Notification,
		bindings:        cfg.Bindings,
		commGroupName:   commGroupName,
		MessagePath:     msgPath,
		Port:            port,
		conversations:   make(map[string]conversation),
		botMentionRegex: botMentionRegex,
		longFormatter:   longFormatter,
		shortFormatter:  shortFormatter,
	}, nil
}

// Start MS Teams server to serve messages from Teams client
func (b *Teams) Start(ctx context.Context) error {
	b.log.Info("Starting bot")
	var err error
	setting := core.AdapterSetting{
		AppID:       b.AppID,
		AppPassword: b.AppPassword,
	}
	b.Adapter, err = core.NewBotAdapter(setting)
	if err != nil {
		return fmt.Errorf("while starting Teams bot: %w", err)
	}

	addr := fmt.Sprintf(":%s", b.Port)

	router := mux.NewRouter()
	router.PathPrefix(b.MessagePath).HandlerFunc(b.processActivity)

	err = b.reporter.ReportBotEnabled(b.IntegrationName())
	if err != nil {
		return fmt.Errorf("while reporting analytics: %w", err)
	}

	srv := httpsrv.New(b.log, addr, router)
	err = srv.Serve(ctx)
	if err != nil {
		return fmt.Errorf("while running MS Teams server: %w", err)
	}

	return nil
}

func (b *Teams) deleteConsent(ctx context.Context, ID string, convRef schema.ConversationReference) {
	b.log.Debugf("Deleting activity %s\n", ID)
	if err := b.Adapter.DeleteActivity(ctx, ID, convRef); err != nil {
		b.log.Errorf("Failed to delete activity. %s", err.Error())
	}
}

func (b *Teams) processActivity(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	b.log.Debugf("Received activity %v\n", req)
	activity, err := b.Adapter.ParseRequest(ctx, req)
	if err != nil {
		b.log.Errorf("Failed to parse Teams request. %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = b.Adapter.ProcessActivity(ctx, activity, coreActivity.HandlerFuncs{
		OnMessageFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
			n, resp := b.processMessage(ctx, turn.Activity)
			if n >= teamsMaxMessageSize {
				if turn.Activity.Conversation.ConversationType == convTypePersonal {
					// send file upload request
					attachments := []schema.Attachment{
						{
							ContentType: contentTypeFile,
							Name:        responseFileName,
							Content: map[string]interface{}{
								"description": turn.Activity.Text,
								"sizeInBytes": len(resp),
								"acceptContext": map[string]interface{}{
									"command": activity.Text,
								},
							},
						},
					}
					return turn.SendActivity(coreActivity.MsgOptionAttachments(attachments))
				}
				resp = fmt.Sprintf("%s\n```\nCluster: %s\n%s", longRespNotice, b.ClusterName, resp[len(resp)-teamsMaxMessageSize:])
			}
			return turn.SendActivity(coreActivity.MsgOptionText(resp))
		},

		// handle invoke events
		// https://developer.microsoft.com/en-us/microsoft-teams/blogs/working-with-files-in-your-microsoft-teams-bot/
		OnInvokeFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
			b.deleteConsent(ctx, turn.Activity.ReplyToID, coreActivity.GetCoversationReference(turn.Activity))
			if err != nil {
				return schema.Activity{}, fmt.Errorf("while reading file: %w", err)
			}
			if turn.Activity.Value["type"] != activityFileUpload {
				return schema.Activity{}, nil
			}
			if turn.Activity.Value["action"] != activityAccept {
				return schema.Activity{}, nil
			}
			if turn.Activity.Value["context"] == nil {
				return schema.Activity{}, nil
			}

			// Parse upload info from invoke accept response
			uploadInfo := schema.UploadInfo{}
			infoJSON, err := json.Marshal(turn.Activity.Value[activityUploadInfo])
			if err != nil {
				return schema.Activity{}, err
			}
			if err := json.Unmarshal(infoJSON, &uploadInfo); err != nil {
				return schema.Activity{}, fmt.Errorf("while unmarshalling activity: %w", err)
			}

			// Parse context
			consentCtx := consentContext{}
			ctxJSON, err := json.Marshal(turn.Activity.Value["context"])
			if err != nil {
				return schema.Activity{}, fmt.Errorf("while marshalling activity context: %w", err)
			}
			if err := json.Unmarshal(ctxJSON, &consentCtx); err != nil {
				return schema.Activity{}, fmt.Errorf("while unmarshalling activity context: %w", err)
			}

			activity.Text = consentCtx.Command
			_, resp := b.processMessage(ctx, activity)

			actJSON, err := json.MarshalIndent(turn.Activity, "", "  ")
			if err != nil {
				return schema.Activity{}, fmt.Errorf("while marshalling activity: %w", err)
			}
			b.log.Debugf("Incoming MSTeams Activity: %s", actJSON)

			// upload file
			err = b.putRequest(uploadInfo.UploadURL, []byte(resp))
			if err != nil {
				return schema.Activity{}, fmt.Errorf("while uploading file: %w", err)
			}

			// notify user about uploaded file
			fileAttach := []schema.Attachment{
				{
					ContentType: contentTypeFile,
					ContentURL:  uploadInfo.ContentURL,
					Name:        uploadInfo.Name,
					Content: map[string]interface{}{
						"uniqueId": uploadInfo.UniqueID,
						"fileType": uploadInfo.FileType,
					},
				},
			}
			return turn.SendActivity(coreActivity.MsgOptionAttachments(fileAttach))
		},
	})
	if err != nil {
		b.log.Errorf("Failed to process request. %s", err.Error())
	}
}

func (b *Teams) processMessage(ctx context.Context, activity schema.Activity) (int, string) {
	trimmedMsg := b.trimBotMention(activity.Text)

	// Multicluster is not supported for Teams

	ref, err := b.getConversationReferenceFrom(activity)
	if err != nil {
		b.log.Errorf("while getting conversation reference: %s", err.Error())
		return 0, ""
	}

	e := b.executorFactory.NewDefault(execute.NewDefaultInput{
		CommGroupName:   b.commGroupName,
		Platform:        b.IntegrationName(),
		NotifierHandler: newTeamsNotifMgrForActivity(b, ref),
		Conversation: execute.Conversation{
			Alias:            "",
			IsAuthenticated:  true,
			ID:               ref.ChannelID,
			ExecutorBindings: b.bindings.Executors,
			CommandOrigin:    command.TypedOrigin,
		},
		Message: trimmedMsg,
	})
	return b.convertInteractiveMessage(e.Execute(ctx), false)
}

func (b *Teams) convertInteractiveMessage(in interactive.Message, forceMarkdown bool) (int, string) {
	var out string

	if in.HasSections() {
		// MS Teams doesn't respect multiple new lines, so it needs to be rendered
		// with `<br>` tags instead  ¯\_(ツ)_/¯
		out = interactive.RenderMessage(b.longFormatter, in)
	} else {
		out = interactive.RenderMessage(b.shortFormatter, in)
	}

	actualLength := len(out)

	if !forceMarkdown && actualLength >= teamsMaxMessageSize {
		return actualLength, interactive.MessageToPlaintext(in, interactive.NewlineFormatter)
	}

	return actualLength, out
}

func (b *Teams) putRequest(u string, data []byte) (err error) {
	client := &http.Client{}
	dec, err := url.QueryUnescape(u)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, dec, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	size := fmt.Sprintf("%d", len(data))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Length", size)
	req.Header.Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(data)-1, len(data)))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		deferredErr := resp.Body.Close()
		if deferredErr != nil {
			err = multierror.Append(err, deferredErr)
		}
	}()
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return fmt.Errorf("failed to upload file with status %d", resp.StatusCode)
	}
	return nil
}

// SendEvent sends event message via Bot interface
func (b *Teams) SendEvent(ctx context.Context, event events.Event, eventSources []string) error {
	b.log.Debugf("Sending to Teams: %+v", event)
	card := b.formatMessage(event, b.Notification)

	if !sliceutil.Intersect(eventSources, b.bindings.Sources) {
		b.log.Debugf(
			"Event was not sent as bot source bindings: %+v do not overlap with the event's sources: %+v",
			b.bindings.Sources,
			eventSources,
		)
		return nil
	}

	errs := multierror.New()
	for _, convRef := range b.getConversationRefsToNotify(eventSources) {
		err := b.sendProactiveMessage(ctx, convRef, card)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while posting message to channel %q: %w", convRef.ChannelID, err))
			continue
		}

		b.log.Debugf("Event successfully sent to channel %q at %b", convRef.ChannelID)
	}

	return errs.ErrorOrNil()
}

// SendGenericMessage sends message to MS Teams to selected conversations.
func (b *Teams) SendGenericMessage(ctx context.Context, genericMsg interactive.GenericMessage, sourceBindings []string) error {
	msg := genericMsg.ForBot(b.BotName())

	errs := multierror.New()
	for _, ref := range b.getConversationRefsToNotify(sourceBindings) {
		channelID := ref.ChannelID

		_, converted := b.convertInteractiveMessage(msg, true)
		b.log.Debugf("Sending message to channel %q: %+v", channelID, converted)
		err := b.Adapter.ProactiveMessage(ctx, ref, coreActivity.HandlerFuncs{
			OnMessageFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
				return turn.SendActivity(coreActivity.MsgOptionText(converted))
			},
		})
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Teams message to channel %q: %w", channelID, err))
			continue
		}
		b.log.Debugf("Message successfully sent to channel %q", channelID)
	}

	return errs.ErrorOrNil()
}

// SendMessageToAll sends message to MS Teams to all conversations.
func (b *Teams) SendMessageToAll(ctx context.Context, msg interactive.Message) error {
	errs := multierror.New()
	for _, convCfg := range b.getConversations() {
		channelID := convCfg.ref.ChannelID

		_, converted := b.convertInteractiveMessage(msg, true)
		b.log.Debugf("Sending message to channel %q: %+v", channelID, converted)
		err := b.Adapter.ProactiveMessage(ctx, convCfg.ref, coreActivity.HandlerFuncs{
			OnMessageFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
				return turn.SendActivity(coreActivity.MsgOptionText(converted))
			},
		})
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending Teams message to channel %q: %w", channelID, err))
			continue
		}
		b.log.Debugf("Message successfully sent to channel %q", channelID)
	}

	return errs.ErrorOrNil()
}

// IntegrationName describes the integration name.
func (b *Teams) IntegrationName() config.CommPlatformIntegration {
	return config.TeamsCommPlatformIntegration
}

// Type describes the integration type.
func (b *Teams) Type() config.IntegrationType {
	return config.BotIntegrationType
}

// NotificationsEnabled returns current notification status for a given channel ID.
func (b *Teams) NotificationsEnabled(channelID string) bool {
	channel, exists := b.getConversations()[channelID]
	if !exists {
		return false
	}

	return channel.notify
}

// SetNotificationsEnabled sets a new notification status for a given channel ID.
func (b *Teams) SetNotificationsEnabled(enabled bool, ref schema.ConversationReference) error {
	// avoid race conditions with using the setter concurrently, as we set whole map
	b.notifyMutex.Lock()
	defer b.notifyMutex.Unlock()

	conversations := b.getConversations()
	conv, exists := conversations[ref.ChannelID]
	if !exists {
		// not returning execute.ErrNotificationsNotConfigured error, as MS Teams channels are configured dynamically.
		// In such case this shouldn't be considered as an error.

		conv = conversation{
			ref: ref,
		}
	}

	conv.notify = enabled
	conversations[ref.ChannelID] = conv
	b.setConversations(conversations)

	return nil
}

// BotName returns the Bot name.
func (b *Teams) BotName() string {
	return fmt.Sprintf("@%s", b.botName)
}

func (b *Teams) sendProactiveMessage(ctx context.Context, convRef schema.ConversationReference, card map[string]interface{}) error {
	err := b.Adapter.ProactiveMessage(ctx, convRef, coreActivity.HandlerFuncs{
		OnMessageFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
			attachments := []schema.Attachment{
				{
					ContentType: contentTypeCard,
					Content:     card,
				},
			}
			return turn.SendActivity(coreActivity.MsgOptionAttachments(attachments))
		},
	})
	return err
}

func (b *Teams) getConversationRefsToNotify(sourceBindings []string) []schema.ConversationReference {
	var convRefsToNotify []schema.ConversationReference
	for _, convConfig := range b.getConversations() {
		if !convConfig.notify {
			b.log.Infof("Skipping notification for channel %q as notifications are disabled.", convConfig.ref.ChannelID)
			continue
		}

		if !sliceutil.Intersect(sourceBindings, b.bindings.Sources) {
			continue
		}

		convRefsToNotify = append(convRefsToNotify, convConfig.ref)
	}
	return convRefsToNotify
}

func (b *Teams) getConversations() map[string]conversation {
	b.conversationsMutex.RLock()
	defer b.conversationsMutex.RUnlock()
	return b.conversations
}

func (b *Teams) setConversations(conversations map[string]conversation) {
	b.conversationsMutex.Lock()
	defer b.conversationsMutex.Unlock()
	b.conversations = conversations
}

// The whole integration should be rewritten using a different library. See the TODO on the top of the file.
func (b *Teams) getConversationReferenceFrom(activity schema.Activity) (schema.ConversationReference, error) {
	// Such ref has the ChannelID property always set to `msteams`. Why? ¯\_(ツ)_/¯
	ref := coreActivity.GetCoversationReference(activity)

	// Set proper IDs as seen in previous implementation. Why both activity and channel IDs are needed? ¯\_(ツ)_/¯
	rawChannelID, exists := activity.ChannelData["teamsChannelId"]
	if !exists {
		// Apparently `msteams` ID is sometimes OK, for example in private conversation.
		// Why? Is there a separation for two users? I guess the Activity ID also matters... ¯\_(ツ)_/¯
		b.log.Info("Teams Channel ID not found. Using default ID...`")
		return ref, nil
	}

	channelID, ok := rawChannelID.(string)
	if !ok {
		return schema.ConversationReference{}, fmt.Errorf("couldn't convert channelID from channel data to string")
	}

	ref.ChannelID = channelID
	ref.Conversation.ID = channelID
	return ref, nil
}

func (b *Teams) trimBotMention(msg string) string {
	return b.botMentionRegex.ReplaceAllString(msg, "")
}

type teamsNotificationManager struct {
	b   *Teams
	ref schema.ConversationReference
}

func newTeamsNotifMgrForActivity(b *Teams, ref schema.ConversationReference) *teamsNotificationManager {
	return &teamsNotificationManager{b: b, ref: ref}
}

// NotificationsEnabled returns current notification status for a given channel ID.
func (n *teamsNotificationManager) NotificationsEnabled(channelID string) bool {
	return n.b.NotificationsEnabled(channelID)
}

// SetNotificationsEnabled sets a new notification status for a given channel ID.
func (n *teamsNotificationManager) SetNotificationsEnabled(_ string, enabled bool) error {
	return n.b.SetNotificationsEnabled(enabled, n.ref)
}

// BotName returns the Bot name.
func (n *teamsNotificationManager) BotName() string {
	return n.b.BotName()
}

func teamsBotMentionRegex(botName string) (*regexp.Regexp, error) {
	botMentionRegex, err := regexp.Compile(fmt.Sprintf(teamsBotMentionPrefixFmt, botName))
	if err != nil {
		return nil, fmt.Errorf("while compiling bot mention regex: %w", err)
	}

	return botMentionRegex, nil
}

// longLineFormatter represents new line formatting for MS Teams where message has multiple sections.
// Unfortunately, it's different from all others integrations.
func longLineFormatter(msg string) string {
	// e.g. `:rocket:` is not supported by MS Teams, so we need to replace it with actual emoji
	msg = replaceEmojiTagsWithActualOne(msg)
	return fmt.Sprintf("%s<br>", msg)
}

func shortLineFormatter(msg string) string {
	// e.g. `:rocket:` is not supported by MS Teams, so we need to replace it with actual emoji
	msg = replaceEmojiTagsWithActualOne(msg)
	return fmt.Sprintf("%s\n", msg)
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
	":white_check_mark:":        "✅",
	":arrows_counterclockwise:": "🔄",
	":exclamation:":             "❗",
	":cricket:":                 "🦗",
}
