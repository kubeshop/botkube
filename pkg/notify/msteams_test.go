package notify

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit test PostCard
func TestPostCard(t *testing.T) {
	tests := map[string]struct {
		server   *httptest.Server
		expected error
	}{
		`Status Not Ok`: {
			httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			})),
			fmt.Errorf("Failed sending to the Teams Channel. Teams http response: %s, %s", string(http.StatusServiceUnavailable), ""),
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
			// create a dummy MsTeams object to test
			m := &MsTeams{
				URL:         ts.URL,
				ClusterName: "test",
			}

			_, err := m.PostCard(&MessageCard{})
			assert.Equal(t, test.expected, err)
		})
	}
}
