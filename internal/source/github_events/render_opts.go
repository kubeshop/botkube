package github_events

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/kubeshop/botkube/internal/source/github_events/templates"
	"github.com/kubeshop/botkube/pkg/api"
)

// WithExtraButtons adds extra buttons to the first section of the api.Message.
func WithExtraButtons(btns []ExtraButton) templates.MessageMutatorOption {
	return func(message api.Message, payload any) api.Message {
		var actBtns api.Buttons
		for _, act := range btns {
			btn, err := renderActionButton(act, payload)
			if err != nil {
				// log error
				continue
			}
			actBtns = append(actBtns, btn)
		}

		if len(actBtns) == 0 {
			return message
		}
		if len(message.Sections) == 0 {
			message.Sections = append(message.Sections, api.Section{
				Buttons: actBtns,
			})
		} else {
			message.Sections[0].Buttons = append(message.Sections[0].Buttons, actBtns...)
		}

		return message

	}
}

func renderActionButton(tpl ExtraButton, e any) (api.Button, error) {
	tmpl, err := template.New("btn").Funcs(sprig.FuncMap()).Parse(tpl.CommandTpl)
	if err != nil {
		return api.Button{}, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, e)
	if err != nil {
		return api.Button{}, err
	}

	btns := api.NewMessageButtonBuilder()

	return btns.ForCommandWithoutDesc(tpl.DisplayName, buf.String(), api.ButtonStyle(tpl.Style)), nil
}
