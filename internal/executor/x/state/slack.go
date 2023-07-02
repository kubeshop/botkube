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
			var val string
			switch {
			case act.SelectedOption.Value != "":
				val = act.SelectedOption.Value
			case act.Value != "":
				val = act.Value
			default:
				continue
			}
			cnt.Fields[id] = val
		}
	}
	return &cnt
}
