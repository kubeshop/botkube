package foo

import (
	"encoding/json"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/pkg/teamsx"
	"github.com/kubeshop/botkube/pkg/bot"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
	"os"
	"path/filepath"
	"testing"
)

func TestNewHelpMessage(t *testing.T) {
	// given
	platform := config.CloudSlackCommPlatformIntegration
	os.Setenv("CONFIG_PROVIDER_IDENTIFIER", "foo")
	// when
	msg := interactive.NewHelpMessage(platform, "Stage US", []string{"botkubeCloud/ai", "botkubeCloud/helm", "botkube/kubectl"}).Build(false)
	msg.ReplaceBotNamePlaceholder("@Botkube")

	// then
	blocks := bot.NewSlackRenderer().RenderAsSlackBlocks(msg)
	raw, err := json.MarshalIndent(SlackBuiltKit{Blocks: blocks}, "", "  ")
	require.NoError(t, err)
	//out := RenderMessage(DefaultMDFormatter(), msg)
	golden.Assert(t, string(raw), filepath.Join(t.Name(), "cloud-slack-help.golden.json"))

	// teams

	_, card, err := teamsx.NewMessageRendererAdapter(loggerx.New(config.Logger{
		Level:         "debug",
		DisableColors: false,
		Formatter:     "",
	}), "botkube", "botkube").RenderCoreMessageCardAndOptions(msg, "Botkube")
	require.NoError(t, err)
	raw, err = json.MarshalIndent(card, "", "  ")
	require.NoError(t, err)
	golden.Assert(t, string(raw), filepath.Join(t.Name(), "cloud-teams-help.golden.json"))
}

type SlackBuiltKit struct {
	Blocks []slack.Block `json:"blocks"`
}
