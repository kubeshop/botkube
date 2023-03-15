package config

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hasura/go-graphql-client"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistNotificationsEnabled(t *testing.T) {
	testCases := []struct {
		Name          string
		ErrMsg        string
		m             *RemotePersistenceManager
		commGroupName string
		platform      CommPlatformIntegration
		channelAlias  string
		enabled       bool
	}{
		{
			Name:          "OK",
			ErrMsg:        "",
			m:             newRemotePersistenceManager(`{"data": {"patchDeploymentConfig": true}}`),
			commGroupName: "default",
			platform:      DiscordCommPlatformIntegration,
			channelAlias:  "botkube",
			enabled:       true,
		},
		{
			Name:     "Unsupported bot platform",
			ErrMsg:   "unsupported platform to persist data",
			m:        newRemotePersistenceManager(`{"data": {"patchDeploymentConfig": true}}`),
			platform: WebhookCommPlatformIntegration,
		},
		{
			Name:          "Received Success == false",
			ErrMsg:        "while persisting notifications config: while retrying: failed to persist notifications config enabled=true for channel botkube",
			m:             newRemotePersistenceManager(`{"data": {"patchDeploymentConfig": false}}`),
			commGroupName: "default",
			platform:      DiscordCommPlatformIntegration,
			channelAlias:  "botkube",
			enabled:       true,
		},
		{
			Name:   "Received error",
			ErrMsg: "while persisting notifications config: while retrying: Message: this is an error, Locations: []",
			m: newRemotePersistenceManager(`{
				"errors": [
					{
						"message": "this is an error"
					}
				]
			}`),
			commGroupName: "default",
			platform:      DiscordCommPlatformIntegration,
			channelAlias:  "botkube",
			enabled:       true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.m.PersistNotificationsEnabled(context.TODO(), tc.commGroupName, tc.platform, tc.channelAlias, tc.enabled)
			if tc.ErrMsg != "" {
				assert.EqualError(t, err, tc.ErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPersistSourceBindings(t *testing.T) {
	testCases := []struct {
		Name          string
		ErrMsg        string
		m             *RemotePersistenceManager
		commGroupName string
		platform      CommPlatformIntegration
		channelAlias  string
		sources       []string
	}{
		{
			Name:          "OK",
			ErrMsg:        "",
			m:             newRemotePersistenceManager(`{"data": {"patchDeploymentConfig": true}}`),
			commGroupName: "default",
			platform:      DiscordCommPlatformIntegration,
			channelAlias:  "botkube",
			sources:       []string{"aaa", "bbb", "ccc"},
		},
		{
			Name:     "Unsupported bot platform",
			ErrMsg:   "unsupported platform to persist data",
			m:        newRemotePersistenceManager(`{"data": {"patchDeploymentConfig": true}}`),
			platform: WebhookCommPlatformIntegration,
		},
		{
			Name:          "Received Success == false",
			ErrMsg:        "while persisting source bindings config: while retrying: failed to persist source bindings config sources=[aaa, bbb, ccc] for channel botkube",
			m:             newRemotePersistenceManager(`{"data": {"patchDeploymentConfig": false}}`),
			commGroupName: "default",
			platform:      DiscordCommPlatformIntegration,
			channelAlias:  "botkube",
			sources:       []string{"aaa", "bbb", "ccc"},
		},
		{
			Name:   "Received error",
			ErrMsg: "while persisting source bindings config: while retrying: Message: this is an error, Locations: []",
			m: newRemotePersistenceManager(`{
				"errors": [
					{
						"message": "this is an error"
					}
				]
			}`),
			commGroupName: "default",
			platform:      DiscordCommPlatformIntegration,
			channelAlias:  "botkube",
			sources:       []string{"aaa", "bbb", "ccc"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.m.PersistSourceBindings(context.TODO(), tc.commGroupName, tc.platform, tc.channelAlias, tc.sources)
			if tc.ErrMsg != "" {
				assert.EqualError(t, err, tc.ErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func newRemotePersistenceManager(resp string) *RemotePersistenceManager {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, resp)
	})

	client := graphql.NewClient("/graphql", &http.Client{Transport: localRoundTripper{handler: mux}})

	return &RemotePersistenceManager{
		log:          logrus.New(),
		gql:          &fakeGraphql{c: client},
		resVerClient: fakeVersionClient{},
	}
}

type fakeGraphql struct {
	c *graphql.Client
}

func (f *fakeGraphql) Client() *graphql.Client {
	return f.c
}

func (f *fakeGraphql) DeploymentID() string {
	return "10"
}

type fakeVersionClient struct{}

func (fakeVersionClient) GetResourceVersion(ctx context.Context) (int, error) {
	return 0, nil
}

// localRoundTripper is an http.RoundTripper that executes HTTP transactions
// by using handler directly, instead of going over an HTTP connection.
type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

func mustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}
