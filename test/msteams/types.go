package msteams

import (
	"net/http/httptest"
	"sync"

	"github.com/infracloudio/botkube/test/e2e/utils"
)

// cardCollection mutex to hold incoming message cards
type cardCollection struct {
	sync.RWMutex
	messages []utils.MsTeamsCard
}

// Server represents a Webhook Test server
type Server struct {
	server        *httptest.Server
	ServerAddr    string
	receivedCards *cardCollection
}
