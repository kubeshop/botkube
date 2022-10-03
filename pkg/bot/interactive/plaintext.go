package interactive

// MessageToPlaintext returns interactive message as a plaintext.
func MessageToPlaintext(msg Message, newlineFormatter func(in string) string) string {
	msg.Description = ""

	fmt := MDFormatter{
		newlineFormatter:           newlineFormatter,
		headerFormatter:            NoFormatting,
		codeBlockFormatter:         NoFormatting,
		adaptiveCodeBlockFormatter: NoFormatting,
	}
	return RenderMessage(fmt, msg)
}
