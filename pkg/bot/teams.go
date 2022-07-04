package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-multierror"
	"github.com/infracloudio/msbotbuilder-go/core"
	coreActivity "github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/sirupsen/logrus"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/httpsrv"
)

const (
	defaultPort      = "3978"
	longRespNotice   = "Response is too long. Sending last few lines. Please send DM to BotKube to get complete response."
	convTypePersonal = "personal"
	maxMessageSize   = 15700
	contentTypeCard  = "application/vnd.microsoft.card.adaptive"
	contentTypeFile  = "application/vnd.microsoft.teams.card.file.consent"
	responseFileName = "response.txt"

	activityFileUpload = "fileUpload"
	activityAccept     = "accept"
	activityUploadInfo = "uploadInfo"
)

var _ Bot = (*Teams)(nil)

// Teams contains credentials to start Teams backend server
type Teams struct {
	log             logrus.FieldLogger
	executorFactory ExecutorFactory

	AppID            string
	AppPassword      string
	MessagePath      string
	Port             string
	AllowKubectl     bool
	RestrictAccess   bool
	ClusterName      string
	NotifType        config.NotifType
	Adapter          core.Adapter
	DefaultNamespace string

	ConversationRef *schema.ConversationReference
}

type consentContext struct {
	Command string
}

// NewTeamsBot returns Teams instance
func NewTeamsBot(log logrus.FieldLogger, c *config.Config, executorFactory ExecutorFactory) *Teams {
	// Set notifier off by default
	config.Notify = false
	port := c.Communications.Teams.Port
	if port == "" {
		port = defaultPort
	}
	msgPath := c.Communications.Teams.MessagePath
	if msgPath == "" {
		msgPath = "/"
	}
	return &Teams{
		log:              log,
		executorFactory:  executorFactory,
		AppID:            c.Communications.Teams.AppID,
		AppPassword:      c.Communications.Teams.AppPassword,
		NotifType:        c.Communications.Teams.NotifType,
		MessagePath:      msgPath,
		Port:             port,
		AllowKubectl:     c.Settings.Kubectl.Enabled,
		RestrictAccess:   c.Settings.Kubectl.RestrictAccess,
		DefaultNamespace: c.Settings.Kubectl.DefaultNamespace,
		ClusterName:      c.Settings.ClusterName,
	}
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
			resp := b.processMessage(turn.Activity)
			if len(resp) >= maxMessageSize {
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
				resp = fmt.Sprintf("%s\n```\nCluster: %s\n%s", longRespNotice, b.ClusterName, resp[len(resp)-maxMessageSize:])
			}
			return turn.SendActivity(coreActivity.MsgOptionText(resp))
		},

		// handle invoke events
		// https://developer.microsoft.com/en-us/microsoft-teams/blogs/working-with-files-in-your-microsoft-teams-bot/
		OnInvokeFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
			b.deleteConsent(ctx, turn.Activity.ReplyToID, coreActivity.GetCoversationReference(turn.Activity))
			if err != nil {
				return schema.Activity{}, fmt.Errorf("failed to read file: %s", err.Error())
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
				return schema.Activity{}, err
			}

			// Parse context
			consentCtx := consentContext{}
			ctxJSON, err := json.Marshal(turn.Activity.Value["context"])
			if err != nil {
				return schema.Activity{}, err
			}
			if err := json.Unmarshal(ctxJSON, &consentCtx); err != nil {
				return schema.Activity{}, err
			}

			msg := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(consentCtx.Command), "<at>BotKube</at>"))
			e := b.executorFactory.NewDefault(config.TeamsBot, true, msg)
			out := e.Execute()

			actJSON, _ := json.MarshalIndent(turn.Activity, "", "  ")
			b.log.Debugf("Incoming MSTeams Activity: %s", actJSON)

			// upload file
			err = b.putRequest(uploadInfo.UploadURL, []byte(out))
			if err != nil {
				return schema.Activity{}, fmt.Errorf("failed to upload file: %s", err.Error())
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

func (b *Teams) processMessage(activity schema.Activity) string {
	// Trim @BotKube prefix
	msg := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(activity.Text), "<at>BotKube</at>"))

	// User needs to execute "notifier start" cmd to enable notifications
	// Parse "notifier" command and set conversation reference
	args := strings.Fields(msg)
	if activity.Conversation.ConversationType != convTypePersonal && len(args) > 0 && execute.ValidNotifierCommand[args[0]] {
		if len(args) < 2 {
			return execute.IncompleteCmdMsg
		}
		if execute.Start.String() == args[1] {
			config.Notify = true
			ref := coreActivity.GetCoversationReference(activity)
			b.ConversationRef = &ref
			// Remove messageID from the ChannelID
			if ID, ok := activity.ChannelData["teamsChannelId"]; ok {
				b.ConversationRef.ChannelID = ID.(string)
				b.ConversationRef.Conversation.ID = ID.(string)
			}
			return fmt.Sprintf(execute.NotifierStartMsg, b.ClusterName)
		}
	}

	// Multicluster is not supported for Teams
	e := b.executorFactory.NewDefault(config.TeamsBot, true, msg)
	return formatCodeBlock(e.Execute())
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
func (b *Teams) SendEvent(ctx context.Context, event events.Event) error {
	card := formatTeamsMessage(event, b.NotifType)
	if err := b.sendProactiveMessage(ctx, card); err != nil {
		b.log.Errorf("Failed to send notification. %s", err.Error())
	}
	b.log.Debugf("Event successfully sent to MS Teams >> %+v", event)
	return nil
}

// SendMessage sends message to MsTeams
func (b *Teams) SendMessage(ctx context.Context, msg string) error {
	if b.ConversationRef == nil {
		b.log.Infof("Skipping SendMessage since conversation ref not set")
		return nil
	}
	err := b.Adapter.ProactiveMessage(ctx, *b.ConversationRef, coreActivity.HandlerFuncs{
		OnMessageFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
			return turn.SendActivity(coreActivity.MsgOptionText(msg))
		},
	})
	if err != nil {
		return err
	}
	b.log.Debug("Message successfully sent to MS Teams")
	return nil
}

func (b *Teams) sendProactiveMessage(ctx context.Context, card map[string]interface{}) error {
	if b.ConversationRef == nil {
		b.log.Infof("Skipping SendMessage since conversation ref not set")
		return nil
	}
	err := b.Adapter.ProactiveMessage(ctx, *b.ConversationRef, coreActivity.HandlerFuncs{
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
