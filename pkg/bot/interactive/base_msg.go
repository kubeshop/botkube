package interactive

import "github.com/kubeshop/botkube/pkg/api"

type Message struct {
	api.Base
	api.Message
}

// GenericMessage returns a message which has customized content. For example, it returns a message with customized commands based on bot name.
type GenericMessage interface {
	ForBot(botName string) Message
}
