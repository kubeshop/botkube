package webhook

import (
	"net/http/httptest"
	"sync"

	"github.com/infracloudio/botkube/test/e2e/utils"
)

// payloadCollection mutex to hold incoming json payloads
type payloadCollection struct {
	sync.RWMutex
	messages []utils.WebhookPayload
}

// Server represents a Webhook Test server
type Server struct {
	server           *httptest.Server
	ServerAddr       string
	receivedPayloads *payloadCollection
}
