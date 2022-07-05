package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit test PostWebhook
func TestPostWebhook(t *testing.T) {
	tests := map[string]struct {
		server   *httptest.Server
		expected error
	}{
		`Status Not Ok`: {
			httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			})),
			fmt.Errorf("Error Posting Webhook: %s", fmt.Sprint(http.StatusServiceUnavailable)),
		},
		`Status Ok`: {
			httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})),
			nil,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			ts := test.server
			defer ts.Close()
			// create a dummy webhook object to test
			w := &Webhook{
				URL: ts.URL,
			}

			err := w.PostWebhook(context.Background(), &WebhookPayload{})
			assert.Equal(t, test.expected, err)
		})
	}
}
