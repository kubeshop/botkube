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

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/infracloudio/botkube/pkg/config"

	"gopkg.in/yaml.v2"
)

// CommunicationRouterConfigFileName is a name of BotKube Router communication configuration file
var CommunicationRouterConfigFileName = "comm_router_config.yaml"

// RouterConfigFileName is a name of BotKube Router cache configuration file
var RouterConfigFileName = ".cache_router.yaml"

// RouterConfig structure of configuration yaml file
type RouterConfig struct {
	Communications CommunicationsRouterConfig
	Routers        []Router
}

// Router contains the route address to be forwarded
type Router struct {
	Key   string
	Value string
}

// CommunicationsRouterConfig contains communication config
type CommunicationsRouterConfig struct {
	Lark config.Lark
}

// NewRouters returns Router Config
func NewRouters() (*RouterConfig, error) {
	c := &RouterConfig{}
	configPath := os.Getenv("CONFIG_PATH")
	communicationRouterConfigFilePath := filepath.Join(configPath, CommunicationRouterConfigFileName)
	communicationRouterConfigFile, err := os.Open(communicationRouterConfigFilePath)
	defer communicationRouterConfigFile.Close()
	if err != nil {
		return c, err
	}
	b, err := ioutil.ReadAll(communicationRouterConfigFile)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		yaml.Unmarshal(b, c)
	}

	routerConfigFilePath := filepath.Join(configPath, RouterConfigFileName)
	routerConfigFile, err := os.Open(routerConfigFilePath)
	defer routerConfigFile.Close()
	if err != nil {
		return c, err
	}

	router, err := ioutil.ReadAll(routerConfigFile)
	if err != nil {
		return c, err
	}

	if len(router) != 0 {
		yaml.Unmarshal(router, c)
	}

	return c, err
}
