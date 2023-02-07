package interactive

import "github.com/kubeshop/botkube/pkg/api"

// Feedback generates Message structure.
func Feedback() Message {
	btnBuilder := api.ButtonBuilder{}
	return Message{
		Message: api.Message{
			Sections: []api.Section{
				{
					Base: api.Base{
						Body: api.Body{
							Plaintext: ":wave: Hey, what's your experience with Botkube so far?",
						},
					},
					Buttons: []api.Button{
						btnBuilder.ForURL("Give feedback", "https://feedback.botkube.io", api.ButtonStylePrimary),
					},
				},
			},
			OnlyVisibleForYou: true,
		},
	}
}
