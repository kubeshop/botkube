package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/ptr"
)

func Test(t *testing.T) {
	file, err := os.ReadFile("testdata/payload.json")
	require.NoError(t, err)

	var raw github.Event
	err = json.Unmarshal(file, &raw)
	require.NoError(t, err)

	e, err := raw.ParsePayload()
	require.NoError(t, err)

	fmt.Println(ptr.ToValue(raw.Type))
	messageRenderer := Get(ptr.ToValue(raw.Type))
	if messageRenderer == nil {
		return
	}

	fmt.Println(messageRenderer(&raw, e))
}
