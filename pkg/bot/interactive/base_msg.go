package interactive

import "github.com/kubeshop/botkube/pkg/api"

// Message holds Botkube internal message model. It's useful to add Botkube specific header or description to plugin messages.
type Message struct {
	Header      string
	Description string
	api.Message
}

// GenericMessage returns a message which has customized content. For example, it returns a message with customized commands based on bot name.
type GenericMessage interface {
	ForBot(botName string) Message
}
