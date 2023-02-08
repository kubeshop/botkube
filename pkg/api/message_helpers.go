package api

// NewCodeBlockMessage returns message in a Markdown code block format.
func NewCodeBlockMessage(msg string, allowBotkubeFilter bool) Message {
	mType := DefaultMessage
	if allowBotkubeFilter {
		mType = BaseBodyWithFilterMessage
	}
	return Message{
		Type: mType,
		BaseBody: Body{
			CodeBlock: msg,
		},
	}
}

// NewPlaintextMessage returns message in a plaintext format.
func NewPlaintextMessage(msg string, useBotkubeFilter bool) Message {
	var mType MessageType
	if useBotkubeFilter {
		mType = BaseBodyWithFilterMessage
	}
	return Message{
		Type: mType,
		BaseBody: Body{
			Plaintext: msg,
		},
	}
}
