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
	return func(message api.Message, payload any) (api.Message, error) {
		var actBtns api.Buttons
		for _, act := range btns {
			btn, err := renderActionButton(act, payload)
			if err != nil {
				return api.Message{}, err
			}
			actBtns = append(actBtns, btn)
		}

		if len(actBtns) == 0 {
			return message, nil
		}
		if len(message.Sections) == 0 {
			message.Sections = append(message.Sections, api.Section{
				Buttons: actBtns,
			})
		} else {
			message.Sections[0].Buttons = append(message.Sections[0].Buttons, actBtns...)
		}

		return message, nil
	}
}

func renderActionButton(tpl ExtraButton, e any) (api.Button, error) {
	btns := api.NewMessageButtonBuilder()

	if tpl.URL != "" {
		value, err := RenderGoTpl(tpl.URL, e)
		if err != nil {
			return api.Button{}, err
		}
		return btns.ForURL(tpl.DisplayName, value, api.ButtonStyle(tpl.Style)), nil
	}

	value, err := RenderGoTpl(tpl.URL, e)
	if err != nil {
		return api.Button{}, err
	}
	return btns.ForCommandWithoutDesc(tpl.DisplayName, value, api.ButtonStyle(tpl.Style)), nil
}

// WithCustomPreview generates a custom api.Message preview.
func WithCustomPreview(previewTpl string) templates.MessageMutatorOption {
	return func(message api.Message, payload any) (api.Message, error) {
		preview, err := RenderGoTpl(previewTpl, payload)
		if err != nil {
			return api.Message{}, err
		}
		// custom preview means that we ignore our renders
		return api.Message{
			BaseBody: api.Body{
				CodeBlock: preview,
			},
		}, nil
	}
}

func RenderGoTpl(tpl string, data any) (string, error) {
	tmpl, err := template.New("btn").Funcs(sprig.FuncMap()).Parse(tpl)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, data)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}
