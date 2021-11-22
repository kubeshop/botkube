// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/utils"
	"io/ioutil"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/router/controller"
	"github.com/larksuite/oapi-sdk-go/core/tools"
	eventhttp "github.com/larksuite/oapi-sdk-go/event/http"
)

const (
	//Challenge authentication request
	Challenge = "challenge"
	//Message event type
	Message = "message"
	//Group chat group
	Group = "group"
	//Type event type
	Type = "type"
	//Event  lark event
	Event = "event"
	//TextWithoutAtBot message of without at bot
	TextWithoutAtBot = "text_without_at_bot"
	//ChatType lark chat type
	ChatType = "chat_type"
	//OpenChatID lark chat id
	OpenChatID = "open_chat_id"
	//ChatID lark chat id
	ChatID = "chat_id"
	//OpenID lark user id
	OpenID = "open_id"
	//Help lark common help doc
	Help = `BotKube: Event notification lark bot

Common actions for BotKube:

- @BotKube get cluster:    list of listening clusters
- @BotKube "kubectl commands" --cluster-name=[clusterName]:    View kubernetes resource information

Usage:
  @BotKube [command]

kubectl commands:
  verbs: ["api-resources", "api-versions", "cluster-info", "describe", "diff", "explain", "get", "logs", "top", "auth"]

  resources: ["deployments", "pods" , "namespaces", "daemonsets", "statefulsets", "storageclasses", "nodes"]

Flags:
  --cluster-name                   set the cluster
  --help,help                        help for BotKube

Use "@BotKube --help" for more information about a command.`
)

// RouterForward deal with the events of lark
func RouterForward(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Errorf("Failed to read the request body. Error: %s", err.Error())
		c.JSON(500, err.Error())
		return
	}
	//decrypt request body information
	b, err := tools.Decrypt(body, controller.LarkEncryptKey)
	if err != nil {
		log.Errorf("Failed to decrypt the request body. Error: %s", err.Error())
		c.JSON(500, err.Error())
		return
	}
	requestBody := make(map[string]interface{})
	err = json.Unmarshal(b, &requestBody)
	if err != nil {
		log.Errorf("Failed to unmarshal the request body. Error: %s", err.Error())
		c.JSON(500, err.Error())
		return
	}
	if _, ok := requestBody[Challenge]; ok {
		log.Infof("request type: %s", requestBody[Type])
		c.JSON(200, requestBody)
		return
	}

	//determine the type of request body
	event, ok := requestBody[Event].(map[string]interface{})
	if !ok {
		log.Error("Missing expected event object in the request")
		return
	}
	eventType := event[Type]
	if strings.EqualFold(eventType.(string), Message) {
		code, res := cmdHandle(event, body, c)
		c.JSON(code, res)
		return
	}
	c.Request.Body = ioutil.NopCloser(bytes.NewReader(body))
	eventhttp.Handle(controller.LarkConf, c.Request, c.Writer)
}

func cmdHandle(event map[string]interface{}, by []byte, c *gin.Context) (code int, body string) {
	message, ok := event[TextWithoutAtBot].(string)
	if !ok {
		log.Error("Missing expected message object in the request")
		return 500, fmt.Sprint("Missing expected message object")
	}
	cmd := strings.TrimSpace(message)
	if strings.Contains(cmd, "--cluster-name") {
		clusterName := utils.GetClusterNameFromKubectlCmd(cmd)
		if v, ok := controller.RouterMap[clusterName]; ok {
			c.Request.Body = ioutil.NopCloser(bytes.NewReader(by))
			return utils.NewProxy(v, c.Request)
		}
	}

	var messages []string
	chatType, ok := event[ChatType].(string)
	if !ok {
		log.Error("Missing expected chatType object in the request")
		return 500, fmt.Sprint("Missing expected chatType object")
	}

	chatID, ok := event[OpenChatID].(string)
	if !ok {
		log.Error("Missing expected chatID object in the request")
		return 500, fmt.Sprint("Missing expected chatID object")
	}

	openID, ok := event[OpenID].(string)
	if !ok {
		log.Error("Missing expected openID object in the request")
		return 500, fmt.Sprint("Missing expected openID object")
	}

	if strings.EqualFold(cmd, "get cluster") {
		log.Infof("get cluster: %s", tools.Prettify(event))
		for k := range controller.RouterMap {
			messages = append(messages, fmt.Sprintf("clusterName: %s\n", k))
		}
		SendMessage(chatType, chatID, openID, strings.Join(messages, ""))
		return 200, ""
	}

	if strings.Contains(cmd, "help") || strings.Contains(cmd, "/botkubehelp") {
		log.Infof("help: %s", tools.Prettify(event))
		messages = append(messages, Help)
		SendMessage(chatType, chatID, openID, strings.Join(messages, ""))
		return 200, ""
	}

	//others event
	Execute(chatType, chatID, openID, message)
	return
}

//Execute run kubernetes API
func Execute(chatType, chatID, openID, message string) {
	executor := execute.NewDefaultExecutor(message, true, true, "",
		"", config.LarkBot, "", true)
	response := executor.Execute()
	SendMessage(chatType, chatID, openID, response)
}

//SendMessage lark bot sends messages to the group
func SendMessage(chatType, chatID, openID, message string) {
	if chatType == Group {
		controller.Lark.SendMessage(ChatID, chatID, message)
		return
	}
	controller.Lark.SendMessage(OpenID, openID, message)
}
