package config

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGql_GetDeployment(t *testing.T) {
	file, err := os.ReadFile("testdata/gql_get_deployment_success.json")
	assert.NoError(t, err)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(file))
	}))
	defer svr.Close()

	g := NewGqlClient(WithAPIURL(svr.URL))
	deployment, err := g.GetDeployment(context.Background(), "16")
	assert.NoError(t, err)
	assert.NotNil(t, deployment.BotkubeConfig)
}
