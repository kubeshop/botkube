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

package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/router/config"
	"github.com/infracloudio/botkube/pkg/router/lark"
	"github.com/infracloudio/botkube/pkg/utils"
	larkconfig "github.com/larksuite/oapi-sdk-go/core/config"
)

const (
	// RetryLimit max retry times
	RetryLimit = 3
)

// Lark LarkBot object
var Lark *lark.RouterBot

//LarkClient lark http client
var LarkClient *lark.Client

// LarkConf lark Config
var LarkConf *larkconfig.Config

// LarkPort lark Config
var LarkPort string

// LarkMessagePath lark Config
var LarkMessagePath string

// LarkEncryptKey lark encryptKey
var LarkEncryptKey string

// RouterMap router map
var RouterMap map[string]string

func init() {
	conf, _ := config.NewRouters()
	Lark = lark.NewLarkBot(conf)
	LarkClient = lark.NewClient(conf.Communications.Lark.Endpoint, "")
	LarkConf = Lark.Conf
	LarkPort = strconv.Itoa(Lark.Port)
	LarkMessagePath = Lark.MessagePath
	LarkEncryptKey = Lark.Conf.GetAppSettings().EncryptKey
	Lark.Start()
	RouterMap = make(map[string]string)
	setRouter(RouterMap, conf.Routers)
}

// setRouter the route address was initialized
func setRouter(router map[string]string, routers []config.Router) {
	for _, item := range routers {
		if _, ok := router[item.Key]; !ok {
			log.Infof("add the router cluster: %s,address: %s", item.Key, item.Value)
			router[item.Key] = item.Value
		}
	}
}

//ConfWatch Listening for RouterConfigFileName file
func ConfWatch() {
	for {
		configPath := os.Getenv("CONFIG_PATH")
		configFile := filepath.Join(configPath, config.RouterConfigFileName)

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal("Failed to create file watcher ", err.Error())
		}

		done := make(chan bool)
		go func() {
			for {
				select {
				case _, ok := <-watcher.Events:
					if !ok {
						log.Errorf("Error in getting events for config file:%s. Error: %s", configFile, err.Error())
						return
					}
					log.Infof("Config file %s is updated. Hence restarting the Pod", configFile)
					done <- true
				case err, ok := <-watcher.Errors:
					if !ok {
						log.Errorf("Error in getting events for config file:%s. Error: %s", configFile, err.Error())
						return
					}
				default:

				}
			}
		}()
		log.Infof("Registering watcher on configfile %s", configFile)
		err = watcher.Add(configFile)
		if err != nil {
			log.Errorf("Unable to register watch on config file:%s. Error: %s", configFile, err.Error())
			return
		}
		<-done

		conf, _ := config.NewRouters()
		chats, err := LarkClient.ChatGroupList(conf)
		if err != nil {
			log.Errorf("Failed to lark chats. Error: %s", err.Error())
		}
		for _, chat := range chats {
			go Lark.SendMessage("chat_id", chat, fmt.Sprintf("Config file %s is updated. Hence reloading the Pod", configFile))
		}
		// Wait for Notifier to send message
		time.Sleep(2 * time.Second)
		//reload the configuration
		setRouter(RouterMap, conf.Routers)
	}
}

//CheckRouters check all routes periodically
func CheckRouters() {
	tick := time.Tick(time.Minute * 30)
	for {
		select {
		case <-tick:
			for k, v := range RouterMap {
				if !botKubeLiveNess(v) {
					conf, _ := config.NewRouters()
					chats, err := LarkClient.ChatGroupList(conf)
					if err != nil {
						log.Errorf("Failed to lark chats. Error: %s", err.Error())
					}
					for _, chat := range chats {
						go Lark.SendMessage("chat_id", chat, fmt.Sprintf("the cluster %s address is not available. please check the network", k))
					}
				}
			}
		default:

		}
	}
}

//botKubeLiveNess check whether the botkube service is available
func botKubeLiveNess(url string) bool {
	for i := 0; i < RetryLimit; i++ {
		_, message := utils.NewProxy(url, &http.Request{})
		if strings.Contains(message, "end of JSON input") {
			return true
		}
	}
	return false
}
