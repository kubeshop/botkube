package bot

import (
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeState(t *testing.T) {
	// given
	fix := &slack.BlockActionStates{
		Values: map[string]map[string]slack.BlockAction{
			"dropdown-block-id-403aca17d958": {
				"@Botkube kc-cmd-builder --resource-name": {
					SelectedOption: slack.OptionBlockObject{
						Value: "nginx2",
					},
				},
				"@Botkube kc-cmd-builder --resource-type": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "pods",
					},
				},
				"@Botkube kc-cmd-builder --verbs": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "get",
					},
				},
			},
		},
	}

	exp := &slack.BlockActionStates{
		Values: map[string]map[string]slack.BlockAction{
			"dropdown-block-id-403aca17d958": {
				"kc-cmd-builder --resource-name": {
					SelectedOption: slack.OptionBlockObject{
						Value: "nginx2",
					},
				},
				"kc-cmd-builder --resource-type": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "pods",
					},
				},
				"kc-cmd-builder --verbs": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "get",
					},
				},
			},
		},
	}

	// when
	out := removeBotNameFromIDs("@Botkube", fix)

	// then
	assert.Equal(t, exp, out)
}
