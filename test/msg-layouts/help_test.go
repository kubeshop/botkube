package msg_layouts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/pkg/teamsx"
	"github.com/kubeshop/botkube/pkg/bot"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
)

// TestNewHelpMessage generates help message directly in Teams and Slack format.
// It's defined here as it requires the Cloud Teams renderer.
// The output is stored in 'testdata/TestNewHelpMessage/' folder. You can just copy-paste it into dedicated editors to see the message layout:
//   - Slack: https://app.slack.com/block-kit-builder/
//   - Teams: https://adaptivecards.io/designer/
//   - Discord: it's only markdown, just post as a normal message in discord channel
//   - Mattermost: it's only markdown, just post as a normal message in discord channel
//
// To update the golden files:
//
//	go test -v -run TestNewHelpMessage -update
func TestNewHelpMessage(t *testing.T) {
	// Cloud options
	platform := config.CloudSlackCommPlatformIntegration
	os.Setenv("CONFIG_PROVIDER_IDENTIFIER", "42")
	msg := interactive.NewHelpMessage(platform, "Stage US", []string{"botkubeCloud/ai", "botkubeCloud/helm", "botkube/kubectl"}).Build(false)
	msg.ReplaceBotNamePlaceholder("@Botkube")

	// Slack
	blocks := bot.NewSlackRenderer().RenderAsSlackBlocks(msg)
	assertJSONGoldenFiles(t, SlackBuiltKit{Blocks: blocks}, "cloud-slack-help.golden.json")

	// Teams
	_, card, err := teamsx.NewMessageRendererAdapter(loggerx.NewNoop(), "botkube", "botkube").RenderCoreMessageCardAndOptions(msg, "Botkube")
	require.NoError(t, err)
	assertJSONGoldenFiles(t, card, "cloud-teams-help.golden.json")

	// Non cloud options
	os.Setenv("CONFIG_PROVIDER_IDENTIFIER", "")
	msg = interactive.NewHelpMessage(config.DiscordCommPlatformIntegration, "Stage US", []string{"botkube/kubectl"}).Build(false)
	msg.ReplaceBotNamePlaceholder("@Botkube")

	// discord - we have only markdown formatter
	md := bot.NewDiscordRenderer().MessageToMarkdown(msg)
	golden.Assert(t, md, filepath.Join(t.Name(), "discord-help.golden.md"))

	// mattermost - we have only markdown formatter
	md = bot.NewMattermostRenderer().MessageToMarkdown(msg)
	golden.Assert(t, md, filepath.Join(t.Name(), "mattermost-help.golden.md"))
}

type SlackBuiltKit struct {
	Blocks []slack.Block `json:"blocks"`
}

func assertJSONGoldenFiles(t *testing.T, in any, goldenFile string) {
	t.Helper()

	raw, err := json.MarshalIndent(in, "", "  ")
	require.NoError(t, err)
	golden.Assert(t, string(raw), filepath.Join(t.Name(), goldenFile))
}
