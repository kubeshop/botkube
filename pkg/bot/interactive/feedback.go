package interactive

// Feedback generates Message structure.
func Feedback() Message {
	btnBuilder := buttonBuilder{}
	return Message{
		Sections: []Section{
			{
				Base: Base{
					Body: Body{
						Plaintext: ":wave: Hey, what's your experience with BotKube so far?",
					},
				},
				Buttons: []Button{
					btnBuilder.ForURL("Give feedback", "https://feedback.botkube.io", ButtonStylePrimary),
				},
			},
		},
		OnlyVisibleForYou: true,
	}
}
