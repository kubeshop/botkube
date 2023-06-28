package template

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api"
)

type (
	Template struct {
		Type         string       `yaml:"type"`
		Trigger      Trigger      `yaml:"trigger"`
		ParseMessage ParseMessage `yaml:"-"`
	}

	Trigger struct {
		Command string `yaml:"command"`
	}

	ParseMessage struct {
		Selects []Select          `yaml:"selects"`
		Actions map[string]string `yaml:"actions"`
		Preview string            `yaml:"preview"`
	}
	WrapMessage struct {
		Buttons api.Buttons `yaml:"buttons"`
	}
	TutorialMessage struct {
		Buttons  api.Buttons `yaml:"buttons"`
		Header   string      `yaml:"header"`
		Paginate Paginate    `yaml:"paginate"`
	}
	Paginate struct {
		Page        int `yaml:"page"`
		CurrentPage int `yaml:"-"`
	}
	Select struct {
		Name   string `yaml:"name"`
		KeyTpl string `yaml:"keyTpl"`
	}
)

func (su *Template) UnmarshalYAML(node *yaml.Node) error {
	var data struct {
		Type    string  `yaml:"type"`
		Trigger Trigger `yaml:"trigger"`
	}
	err := node.Decode(&data)
	if err != nil {
		return err
	}

	switch {
	case strings.HasPrefix(data.Type, "parser:"):
		var data struct {
			Message ParseMessage `yaml:"message"`
		}
		err = node.Decode(&data)
		if err != nil {
			return err
		}
		su.ParseMessage = data.Message
	}

	su.Type = data.Type
	su.Trigger = data.Trigger
	return nil
}

func FindWithPrefix(tpls []Template, cmd string) (Template, bool) {
	for idx := range tpls {
		item := tpls[idx]
		if item.Trigger.Command == "" {
			continue
		}
		if strings.HasPrefix(cmd, item.Trigger.Command) {
			return item, true
		}
	}

	return Template{}, false
}
