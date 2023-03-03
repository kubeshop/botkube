package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/internal/graphql"
)

func TestGql_GetDeployment(t *testing.T) {
	expectedBody := fmt.Sprintf(`{"query":"query ($id:ID!){deployment(id: $id){resourceVersion,yamlConfig}}","variables":{"id":"my-id"}}%s`, "\n")
	file, err := os.ReadFile("testdata/gql_get_deployment_success.json")
	assert.NoError(t, err)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		t.Log("body", string(bodyBytes))
		t.Log("expected body", expectedBody)

		if !strings.EqualFold(string(bodyBytes), expectedBody) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fmt.Fprint(w, string(file))
	}))
	defer svr.Close()

	gqlClient := graphql.NewGqlClient(
		graphql.WithEndpoint(svr.URL),
		graphql.WithAPIKey("api-key"),
		graphql.WithDeploymentID("my-id"),
	)
	g := NewDeploymentClient(gqlClient)
	deployment, err := g.GetConfigWithResourceVersion(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, deployment.YAMLConfig)
}
