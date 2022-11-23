package plugin

import (
	"net/http"
	"time"
)

const defaultTimeout = 30 * time.Second

// newHTTPClient creates a new http client with timeout.
func newHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: defaultTimeout,
	}
	return client
}
