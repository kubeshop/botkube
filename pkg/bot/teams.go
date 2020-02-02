package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/logging"
	"github.com/infracloudio/msbotbuilder-go/core"
	coreActivity "github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
)

const (
	defaultMsgPath = "/api/messages"
	defaultPort    = "3978"
)

// Teams contains credentials to start Teams backend server
type Teams struct {
	AppID          string
	AppPassword    string
	MessagePath    string
	Port           string
	AllowKubectl   bool
	RestrictAccess bool
	ClusterName    string
	NotifType      config.NotifType
	Adapter        core.Adapter

	ConversationRef schema.ConversationReference
}

// NewTeamsBot returns Teams instance
func NewTeamsBot(c *config.Config) *Teams {
	logging.Logger.Infof("Config:: %+v", c.Communications.Teams)
	return &Teams{
		AppID:          c.Communications.Teams.AppID,
		AppPassword:    c.Communications.Teams.AppPassword,
		NotifType:      c.Communications.Teams.NotifType,
		MessagePath:    defaultMsgPath,
		Port:           defaultPort,
		AllowKubectl:   c.Settings.AllowKubectl,
		RestrictAccess: c.Settings.RestrictAccess,
		ClusterName:    c.Settings.ClusterName,
	}
}

// Start MS Teams server to serve messages from Teams client
func (t *Teams) Start() {
	var err error
	setting := core.AdapterSetting{
		AppID:       t.AppID,
		AppPassword: t.AppPassword,
	}
	t.Adapter, err = core.NewBotAdapter(setting)
	if err != nil {
		logging.Logger.Errorf("Failed Start teams bot. %+v", err)
		return
	}
	http.HandleFunc(t.MessagePath, t.processActivity)
	logging.Logger.Infof("Started MS Teams server on port %s", defaultPort)
	logging.Logger.Errorf("Error in MS Teams server. %v", http.ListenAndServe(fmt.Sprintf(":%s", t.Port), nil))
}

func (t *Teams) processActivity(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	activity, err := t.Adapter.ParseRequest(ctx, req)
	if err != nil {
		logging.Logger.Errorf("Failed to parse Teams request. %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = t.Adapter.ProcessActivity(ctx, activity, coreActivity.HandlerFuncs{
		OnMessageFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
			actjson, _ := json.MarshalIndent(turn.Activity, "", "  ")
			logging.Logger.Debugf("Received activity: %s", actjson)
			return turn.SendActivity(coreActivity.MsgOptionText(t.processMessage(turn.Activity)))
		},
	})
	if err != nil {
		logging.Logger.Errorf("Failed to process request. %s", err.Error())
	}
}

func (t *Teams) processMessage(activity schema.Activity) string {
	// Trim @BotKube prefix
	msg := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(activity.Text), "<at>BotKube</at>"))

	// Parse "set default channel" command and set conversation reference
	if msg == "set default channel" {
		t.ConversationRef = coreActivity.GetCoversationReference(activity)
		// Remove messageID from the ChannelID
		if ID, ok := activity.ChannelData["teamsChannelId"]; ok {
			t.ConversationRef.ChannelID = ID.(string)
			t.ConversationRef.Conversation.ID = ID.(string)
		}
		return "Okay. I'll send notifications to this channel"
	}

	// Multicluster is not supported for Teams
	e := execute.NewDefaultExecutor(msg, t.AllowKubectl, t.RestrictAccess, t.ClusterName, true)
	return fmt.Sprintf("```%s\n%s```", t.ClusterName, e.Execute())
}

func (t *Teams) SendEvent(event events.Event) error {
	card := formatTeamsMessage(event, t.NotifType)
	if err := t.sendProactiveMessage(card); err != nil {
		logging.Logger.Errorf("Failed to send notification. %s", err.Error())
	}
	logging.Logger.Debugf("Event successfully sent to MS Teams >> %+v", event)
	return nil
}

// SendMessage sends message to MsTeams
func (t *Teams) SendMessage(msg string) error {
	err := t.Adapter.ProactiveMessage(context.TODO(), t.ConversationRef, coreActivity.HandlerFuncs{
		OnMessageFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
			return turn.SendActivity(coreActivity.MsgOptionText(msg))
		},
	})
	if err != nil {
		return err
	}
	logging.Logger.Debug("Message successfully sent to MS Teams")
	return nil
}

func (t *Teams) sendProactiveMessage(card map[string]interface{}) error {
	err := t.Adapter.ProactiveMessage(context.TODO(), t.ConversationRef, coreActivity.HandlerFuncs{
		OnMessageFunc: func(turn *coreActivity.TurnContext) (schema.Activity, error) {
			attachments := []schema.Attachment{
				{
					ContentType: "application/vnd.microsoft.card.adaptive",
					Content:     card,
				},
			}
			return turn.SendActivity(coreActivity.MsgOptionAttachments(attachments))
		},
	})
	return err
}
