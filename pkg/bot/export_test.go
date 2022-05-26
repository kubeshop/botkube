package bot

func (b *SlackBot) StripUnmarshallingErrEventDetails(errMessage string) string {
	return b.stripUnmarshallingErrEventDetails(errMessage)
}
