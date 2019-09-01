package webhook

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"

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
	log.Print("starting Mock Webhook server")
	s.server.Start()
}

// GetAPIURL returns the api url you can pass to webhook
func (s *Server) GetAPIURL() string {
	return "http://" + s.ServerAddr + "/"
}

// GetReceivedPayloads returns all messages received
func (s *Server) GetReceivedPayloads() []utils.WebhookPayload {
	s.receivedPayloads.RLock()
	m := s.receivedPayloads.messages
	s.receivedPayloads.RUnlock()
	return m
}
