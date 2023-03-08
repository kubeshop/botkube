package lifecycle

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
)

func TestNewReloadHandler_HappyPath(t *testing.T) {
	// given
	expectedResponse := `Deployment restarted successfully.`
	expectedStatusCode := http.StatusOK

	restarter := &fakeRestarter{}

	req := httptest.NewRequest(http.MethodPost, "/reload", nil)
	writer := httptest.NewRecorder()
	handler := newReloadHandler(loggerx.NewNoop(), restarter)

	// when
	handler(writer, req)

	res := writer.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	// then
	assert.Equal(t, expectedStatusCode, res.StatusCode)
	assert.Equal(t, expectedResponse, string(data))
	assert.True(t, restarter.called)
}

type fakeRestarter struct {
	called bool
}

func (r *fakeRestarter) Do(ctx context.Context) error {
	r.called = true
	return nil
}
