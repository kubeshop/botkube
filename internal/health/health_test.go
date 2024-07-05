package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestServeHTTPUnavailable(t *testing.T) {
	// given
	checker := NewChecker(context.TODO(), &config.Config{}, nil)
	expectedStatus := checker.GetStatus()

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	// when
	checker.ServeHTTP(rr, req)

	// then
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp Status
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, BotkubeStatusUnhealthy, resp.Botkube.Status)
	assert.Equal(t, resp.Botkube.Status, expectedStatus.Botkube.Status)
}

func TestServeHTTPOK(t *testing.T) {
	// given
	checker := NewChecker(context.TODO(), &config.Config{}, nil)
	checker.MarkAsReady()
	expectedStatus := checker.GetStatus()

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	// when
	checker.ServeHTTP(rr, req)

	// then
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp Status
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, BotkubeStatusHealthy, resp.Botkube.Status)
	assert.Equal(t, resp.Botkube.Status, expectedStatus.Botkube.Status)
}
