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

package lark

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/infracloudio/botkube/pkg/log"

	"github.com/infracloudio/botkube/pkg/router/config"
)

//jsonHeader the HTTP request header
var jsonHeader = http.Header{"content-type": []string{"application/json; charset=utf-8"}}

//Client represent a Lark API client
type Client struct {
	url         string
	accessToken string
	client      *http.Client
}

// NewClient initializes and returns an API client.
func NewClient(url, token string) *Client {
	return &Client{
		url:         strings.TrimSuffix(url, "/"),
		accessToken: token,
		client:      &http.Client{},
	}
}

// SetHTTPClient replaces default http.Client with user given one.
func (c *Client) SetHTTPClient(client *http.Client) {
	c.client = client
}

func (c *Client) doRequest(method, path string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.url+"/open-apis"+path, body)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(c.accessToken, "") {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	for k, v := range header {
		req.Header[k] = v
	}

	return c.client.Do(req)
}

func (c *Client) getResponse(method, path string, header http.Header, body io.Reader) ([]byte, error) {
	resp, err := c.doRequest(method, path, header, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case 403:
		return nil, errors.New("403 Forbidden")
	case 404:
		return nil, errors.New("404 Not Found")
	}

	if resp.StatusCode/100 != 2 {
		errMap := make(map[string]interface{})
		if err = json.Unmarshal(data, &errMap); err != nil {
			return nil, err
		}
		message, ok := errMap["message"].(string)
		if !ok {
			log.Errorf("Missing expected message object: %+v", errMap["message"])
			return nil, fmt.Errorf("Missing expected message object")
		}
		return nil, errors.New(message)
	}

	return data, nil
}

func (c *Client) getParsedResponse(method, path string, header http.Header, body io.Reader, obj interface{}) error {
	data, err := c.getResponse(method, path, header, body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

// AppLarkOption contains lark app information
type AppLarkOption struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

//AccessToken contains lark token response body
type AccessToken struct {
	Code              int    `json:"code"`
	Expire            int    `json:"expire"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
}

func (c *Client) getToken(opt AppLarkOption) (*AccessToken, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	accessToken := new(AccessToken)
	return accessToken, c.getParsedResponse("POST",
		fmt.Sprint("/auth/v3/tenant_access_token/internal"), jsonHeader, bytes.NewReader(body), accessToken)
}

// ChatsOption contains lark chat response body
type ChatsOption struct {
	Code int    `json:"code"`
	Data Data   `json:"data"`
	Msg  string `json:"msg"`
}

//Data description of lark chats group
type Data struct {
	HasMore   bool         `json:"has_more"`
	Items     []ChatOption `json:"items"`
	PageToken string       `json:"page_token"`
}

//ChatOption contains lark chat information
type ChatOption struct {
	Avatar      string `json:"avatar"`
	ChatID      string `json:"chat_id"`
	Description string `json:"description"`
	External    bool   `json:"external"`
	Name        string `json:"name"`
	OwnerID     string `json:"owner_id"`
	OwnerIDType string `json:"owner_id_type"`
	TenantKey   string `json:"tenant_key"`
}

//ChatGroupList Gets the lark chat group list
func (c *Client) ChatGroupList(conf *config.RouterConfig) ([]string, error) {
	token, err := c.getToken(AppLarkOption{
		AppID:     conf.Communications.Lark.AppID,
		AppSecret: conf.Communications.Lark.AppSecret,
	})

	if err != nil {
		return nil, err
	}

	c.accessToken = token.TenantAccessToken
	chatsOption := new(ChatsOption)
	err = c.getParsedResponse("GET", "/im/v1/chats?page_size=100", jsonHeader, nil, &chatsOption)
	if err != nil {
		return nil, err
	}
	var chats []string
	for _, v := range chatsOption.Data.Items {
		chats = append(chats, v.ChatID)
	}
	return chats, err
}
