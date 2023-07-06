package output

import (
	"github.com/kubeshop/botkube/internal/executor/x/state"
	"github.com/kubeshop/botkube/internal/executor/x/template"
	"github.com/kubeshop/botkube/pkg/api"
)

type CommandWrapper struct{}

func NewCommandWrapper() *CommandWrapper {
	return &CommandWrapper{}
}

func (p *CommandWrapper) RenderMessage(_, output string, _ *state.Container, msgCtx *template.Template) (api.Message, error) {
	msg := msgCtx.WrapMessage

	return api.Message{
		Sections: []api.Section{
			{
				Base: api.Base{
					Body: api.Body{
						CodeBlock: output,
					},
				},
			},
			{
				Buttons: msg.Buttons,
			},
		},
	}, nil
}
