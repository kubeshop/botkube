package msg

import (
	"fmt"
	"github.com/kubeshop/botkube/internal/ptr"
	"github.com/nsf/jsondiff"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

func Test(t *testing.T) {

	exp, err := os.ReadFile("exp-msg.json")
	require.NoError(t, err)

	got, err := os.ReadFile("teams-msg.json")
	require.NoError(t, err)

	opts := jsondiff.DefaultConsoleOptions()
	opts.SkipMatches = true
	result, diff := jsondiff.Compare(exp, got, ptr.FromType(opts))

	changed := strings.Contains(diff, "=>") || strings.Contains(diff, "<changed>") || strings.Contains(diff, "</removed≥")
	fmt.Println(diff)
	fmt.Println(changed)
	fmt.Println(result)
}
