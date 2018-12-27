package slack

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/infracloudio/kubeops/pkg/config"
	"github.com/infracloudio/kubeops/pkg/logging"
	"github.com/nlopes/slack"
)

type SlackBot struct {
	Token string
}

func NewSlackBot() *SlackBot {
	c, err := config.New()
	if err != nil {
		logging.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}
	return &SlackBot{
		Token: c.Communications.Slack.Token,
	}
}

func (s *SlackBot) Start() {
	api := slack.New(s.Token)
	authResp, err := api.AuthTest()
	if err != nil {
		logging.Logger.Fatal(err)
	}
	botID := authResp.UserID

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			logging.Logger.Debug("Connection Info: ", ev.Info)

		case *slack.MessageEvent:
			// Serve only if mentioned
			if !strings.HasPrefix(ev.Text, "<@"+botID+">") {
				continue
			}
			logging.Logger.Debugf("Slack incoming message: %+v", ev)
			msg := strings.TrimPrefix(ev.Text, "<@"+botID+"> ")
			handleMessage(rtm, ev.Channel, msg)

		case *slack.RTMError:
			logging.Logger.Errorf("Slack RMT error: %+v", ev.Error())

		case *slack.InvalidAuthEvent:
			logging.Logger.Error("Invalid Credentials")
			return
		default:
		}
	}
}

func handleMessage(rtm *slack.RTM, channelID, msg string) {
	args := strings.Split(msg, " ")
	isLog := false
	out, err := runCommand(args, &isLog)
	if err != nil {
		if !strings.Contains(out, "Forbidden") {
			out = "Sorry, I don't understand"
		}
		formatAndSendMsg(rtm, channelID, out)
		return
	}
	if isLog {
		formatAndSendLogs(rtm, channelID, out, msg)
		return
	}
	if strings.ToLower(args[0]) == "help" {
		formatAndSendHelp(rtm, channelID)
		return
	}
	formatAndSendMsg(rtm, channelID, out)
}

func runCommand(args []string, isLog *bool) (string, error) {
	// Use 'default' as a default namespace
	args = append([]string{"-n", "default"}, args...)

	// Remove unnecessary flags
	finalArgs := []string{}
	for _, a := range args {
		if a == "-f" || strings.HasPrefix(a, "--follow") {
			continue
		}
		if a == "-w" || strings.HasPrefix(a, "--watch") {
			continue
		}
		if a == "log" || a == "logs" {
			*isLog = true
		}
		finalArgs = append(finalArgs, a)
	}

	cmd := exec.Command("/usr/local/bin/kubectl", finalArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func formatAndSendLogs(rtm *slack.RTM, channelID, logs string, filename string) {
	params := slack.FileUploadParameters{
		Title:    filename,
		Content:  logs,
		Filetype: "log",
		Channels: []string{channelID},
	}
	_, err := rtm.UploadFile(params)
	if err != nil {
		logging.Logger.Error("Error in uploading file:", err)
	}
}

func formatAndSendMsg(rtm *slack.RTM, channelID, message string) {
	params := slack.PostMessageParameters{}
	params.AsUser = true
	channelID, _, err := rtm.PostMessage(channelID, "```"+message+"```", params)
	if err != nil {
		logging.Logger.Error("Error in sending message:", err)
	}
}

func formatAndSendHelp(rtm *slack.RTM, channelID string) {
	params := slack.PostMessageParameters{}
	params.AsUser = true
	helpMsg := "```" +
		"kubeops executes kubectl commands on k8s cluster and returns output.\n" +
		"Usages:\n" +
		"    @kubeops <kubectl command without `kubectl` prefix>\n" +
		"e.g:\n" +
		"    @kubeops get pods\n" +
		"    @kubeops logs podname -n namespace```"
	channelID, _, err := rtm.PostMessage(channelID, helpMsg, params)
	if err != nil {
		logging.Logger.Error("Error in sending message:", err)
	}
}
