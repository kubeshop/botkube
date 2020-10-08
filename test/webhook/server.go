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

package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/test/e2e/utils"
)

// handle chat.postMessage
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	decoder := json.NewDecoder(r.Body)

	var t utils.WebhookPayload

	err := decoder.Decode(&t)
	if err != nil {
		panic(err)
	}

	// update message in mutex
	s.receivedPayloads.Lock()
	log.Debugf("Incoming Webhook Messages :%#v", t)
	s.receivedPayloads.messages = append(s.receivedPayloads.messages, t)
	s.receivedPayloads.Unlock()

}

// NewTestServer returns a slacktest.Server ready to be started
func NewTestServer() *Server {

	s := &Server{
		receivedPayloads: &payloadCollection{},
	}
	httpserver := httptest.NewUnstartedServer(s)
	addr := httpserver.Listener.Addr().String()
	s.ServerAddr = addr
	s.server = httpserver
	return s
}

// Start starts the test server
func (s *Server) Start() {
	log.Info("starting Mock Webhook server")
	s.server.Start()
}

// GetAPIURL returns the api url you can pass to webhook
func (s *Server) GetAPIURL() string {
	return "http://" + s.ServerAddr + "/"
}

// GetReceivedPayloads returns all messages received
func (s *Server) GetReceivedPayloads() []utils.WebhookPayload {
	s.receivedPayloads.Lock()
	m := s.receivedPayloads.messages
	s.receivedPayloads.Unlock()
	return m
}
