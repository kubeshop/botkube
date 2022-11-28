package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/pkg/api/source"
)

const pluginName = "cm-watcher"

// Config holds executor configuration.
type Config struct {
	ConfigMapName string
}

// CMWatcher implements Botkube source plugin.
type CMWatcher struct{}

// Stream returns a given command as response.
func (CMWatcher) Stream(ctx context.Context) (source.StreamOutput, error) {
	// TODO: in request we should receive the executor configuration.
	cfg := Config{
		ConfigMapName: "cm-watcher-trigger",
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		return source.StreamOutput{}, err
	}
	out := source.StreamOutput{
		Output: make(chan []byte),
	}

	go func() {
		for {
			select {
			case <-time.Tick(1 * time.Second):
				out.Output <- raw
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

func main() {
	source.Serve(map[string]plugin.Plugin{
		pluginName: &source.Plugin{
			Source: &CMWatcher{},
		},
	})
}
