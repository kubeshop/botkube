package main

import (
	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/internal/source/github_events"
	"github.com/kubeshop/botkube/pkg/api/source"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

func main() {
	source.Serve(map[string]plugin.Plugin{
		github_events.PluginName: &source.Plugin{
			Source: github_events.NewSource(version),
		},
	})
}
