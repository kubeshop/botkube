package interactive

// MessageToPlaintext returns interactive message as a plaintext.
func MessageToPlaintext(msg CoreMessage, newlineFormatter func(in string) string) string {
	msg.Description = ""

	fmt := MDFormatter{
		NewlineFormatter:           newlineFormatter,
		HeaderFormatter:            NoFormatting,
		CodeBlockFormatter:         NoFormatting,
		AdaptiveCodeBlockFormatter: NoFormatting,
	}
	return RenderMessage(fmt, msg)
}
