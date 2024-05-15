package sink

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
)

func TestPagerDuty_SendEvent(t *testing.T) {
	const integrationKey = "integration-key"

	tests := []struct {
		name       string
		eventType  string
		statusCode int
		expPath    string
		givenEvent map[string]any
	}{
		{
			name:       "alert event",
			givenEvent: fixK8sPodErrorAlert(),
			expPath:    "/v2/enqueue",
		},
		{
			name:       "change event",
			givenEvent: fixK8sDeployUpdateAlert(),
			expPath:    "/v2/change/enqueue",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.expPath, r.URL.Path)

				var payload struct {
					RoutingKey string `json:"routing_key"`
				}
				err := json.NewDecoder(r.Body).Decode(&payload)
				require.NoError(t, err)
				assert.Equal(t, integrationKey, payload.RoutingKey)

				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{}`))
			}))
			defer server.Close()

			pd, err := NewPagerDuty(loggerx.NewNoop(), 0, config.PagerDuty{
				Enabled:             true,
				IntegrationKey:      integrationKey,
				V2EventsAPIBasePath: server.URL,
				Bindings: config.SinkBindings{
					Sources: []string{"kubernetes-err"},
				},
			}, "labs", analytics.NewNoopReporter())
			require.NoError(t, err)

			err = pd.SendEvent(context.Background(), tc.givenEvent, []string{"kubernetes-err"})
			require.NoError(t, err)
		})
	}
}

func fixK8sPodErrorAlert() map[string]any {
	return map[string]any{
		"APIVersion": "v1",
		"Kind":       "Pod",
		"Title":      "v1/pods error",
		"Name":       "webapp",
		"Namespace":  "dev",
		"Resource":   "v1/pods",
		"Messages":   []string{"Back-off restarting failed container webapp in pod webapp_dev(0a405592-2615-4d0c-b399-52ada5a9cc1b)"},
		"Type":       "error",
		"Reason":     "BackOff",
		"Level":      "error",
		"Cluster":    "labs",
		"TimeStamp":  "2024-05-14T19:47:24.828568+09:00",
		"Count":      int32(1),
	}
}

func fixK8sDeployUpdateAlert() map[string]any {
	return map[string]any{
		"API Version": "apps/v1",
		"Cluster":     "labs",
		"Count":       0,
		"Kind":        "Deployment",
		"Level":       "info",
		"Messages":    []string{"status.availableReplicas:\n\t-: <none>\n\t+: 1\nstatus.readyReplicas:\n\t-: <none>\n\t+: 1\n"},
		"Name":        "nginx-deployment",
		"Namespace":   "botkube",
		"Resource":    "apps/v1/deployments",
		"Title":       "apps/v1/deployments updated",
		"Type":        "update",
		"TimeStamp":   "2024-05-14T19:47:24.828568+09:00",
	}
}
