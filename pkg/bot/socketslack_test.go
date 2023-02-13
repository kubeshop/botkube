package bot

import (
	"testing"
)

func TestNormalizeState(t *testing.T) {
	//// given
	//fix := []*slack.BlockAction{
	//	{
	//		ActionID: "@Botkube kubectl @builder --verbs",
	//		BlockID:  "13a13f04-4fc8-47e1-a167-f87980b3e834",
	//		Type:     "static_select",
	//		ActionTs: "1675869299.369379",
	//		SelectedOption: slack.OptionBlockObject{
	//			Text: &slack.TextBlockObject{
	//				Type:  "plain_text",
	//				Text:  "get",
	//				Emoji: true,
	//			},
	//			Value: "get",
	//		},
	//	},
	//}
	//fx := &slack.BlockActionStates{
	//	Values: map[string]map[string]slack.BlockAction{
	//		""
	//	},
	//}
	//// when
	//out := removeBotNameFromIDs("@Botkube", fix)
	//// then

}
