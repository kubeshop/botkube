package httpx

import (
	"net/http"
	"time"
)

const defaultTimeout = 30 * time.Second

// NewHTTPClient creates a new http client with timeout.
func NewHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: defaultTimeout,
	}
	return client
}
