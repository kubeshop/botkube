package state

import (
	"github.com/slack-go/slack"
)

type Container struct {
	SelectsBlockID string
	Fields         map[string]string
}

func (c *Container) GetSelectsBlockID() string {
	if c == nil {
		return ""
	}
	return c.SelectsBlockID
}

func (c *Container) GetField(name string) string {
	if c == nil {
		return ""
	}
	return c.Fields[name]
}

func ExtractSlackState(state *slack.BlockActionStates) *Container {
	if state == nil {
		return nil
	}

	cnt := Container{
		Fields: map[string]string{},
	}
	for blockID, blocks := range state.Values {
		cnt.SelectsBlockID = blockID
		for id, act := range blocks {
			//id = strings.TrimPrefix(id, kubectlCommandName)
			//id = strings.TrimSpace(id)
			cnt.Fields[id] = act.SelectedOption.Value
		}
	}
	return &cnt
}
