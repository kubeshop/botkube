package template

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api"
)

type (
	// Template represents a template for message parsing.
	Template struct {
		Type                 string          `yaml:"type"`
		Trigger              Trigger         `yaml:"trigger"`
		SkipCommandExecution bool            `yaml:"-"`
		ParseMessage         ParseMessage    `yaml:"-"`
		WrapMessage          WrapMessage     `yaml:"-"`
		TutorialMessage      TutorialMessage `yaml:"-"`
	}

	// Trigger represents the trigger configuration for a template.
	Trigger struct {
		Command CommandMatchers `yaml:"command"`
	}

	// CommandMatchers represents different command matching strategies.
	CommandMatchers struct {
		Prefix string `yaml:"prefix"`
		Regexp string `yaml:"regex"`
	}

	// ParseMessage holds template for message that will be parsed by defined parser.
	ParseMessage struct {
		Selects []Select          `yaml:"selects"`
		Actions map[string]string `yaml:"actions"`
		Preview string            `yaml:"preview"`
	}

	// WrapMessage holds template for wrapping command output with additional context.
	WrapMessage struct {
		Buttons api.Buttons `yaml:"buttons"`
	}

	// Select holds template select primitive definition.
	Select struct {
		Name   string `yaml:"name"`
		KeyTpl string `yaml:"keyTpl"`
	}

	// TutorialMessage holds template interactive tutorial message.
	TutorialMessage struct {
		Buttons  api.Buttons `yaml:"buttons"`
		Header   string      `yaml:"header"`
		Paginate Paginate    `yaml:"paginate"`
	}

	// Paginate holds data required to do the pagination.
	Paginate struct {
		Page        int `yaml:"page"`
		CurrentPage int `yaml:"-"`
	}
)

// UnmarshalYAML is a custom unmarshaler for Template allowing to unmarshal into a proper struct
// base on defined template type.
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
	case data.Type == "wrapper":
		var data struct {
			Message WrapMessage `yaml:"message"`
		}
		err = node.Decode(&data)
		if err != nil {
			return err
		}
		su.WrapMessage = data.Message
	case data.Type == "tutorial":
		var data struct {
			Message TutorialMessage `yaml:"message"`
		}
		err = node.Decode(&data)
		if err != nil {
			return err
		}

		su.SkipCommandExecution = true
		su.TutorialMessage = data.Message
	}

	su.Type = data.Type
	su.Trigger = data.Trigger
	return nil
}

// FindTemplate finds a template with a matching command prefix.
func FindTemplate(tpls []Template, cmd string) (Template, bool) {
	for _, item := range tpls {
		switch {
		case item.Trigger.Command.Prefix != "":
			if strings.HasPrefix(cmd, item.Trigger.Command.Prefix) {
				return item, true
			}
		case item.Trigger.Command.Regexp != "":
			matched, _ := regexp.MatchString(item.Trigger.Command.Regexp, cmd)
			if matched {
				return item, true
			}
		}
	}

	return Template{}, false
}
