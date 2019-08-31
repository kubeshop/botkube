package msteams

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

	var t utils.MsTeamsCard

	err := decoder.Decode(&t)
	if err != nil {
		panic(err)
	}

	// update message in mutex
	s.receivedCards.Lock()
	s.receivedCards.messages = append(s.receivedCards.messages, t)
	s.receivedCards.Unlock()

}

// NewTestServer returns a slacktest.Server ready to be started
func NewTestServer() *Server {

	s := &Server{
		receivedCards: &cardCollection{},
	}
	httpserver := httptest.NewUnstartedServer(s)
	addr := httpserver.Listener.Addr().String()
	s.ServerAddr = addr
	s.server = httpserver
	return s
}

// Start starts the test server
func (s *Server) Start() {
	log.Print("starting Mock MsTeams server")
	s.server.Start()
}

// GetAPIURL returns the api url you can pass to msteams
func (s *Server) GetAPIURL() string {
	return "http://" + s.ServerAddr + "/"
}

// GetReceivedCards returns all messages received
func (s *Server) GetReceivedCards() []utils.MsTeamsCard {
	s.receivedCards.RLock()
	m := s.receivedCards.messages
	s.receivedCards.RUnlock()
	return m
}
