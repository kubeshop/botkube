package main

import (
	"github.com/hashicorp/go-plugin"
	"github.com/kubeshop/botkube/internal/source/kubernetes"

	"github.com/kubeshop/botkube/internal/source/prometheus"
	"github.com/kubeshop/botkube/pkg/api/source"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

func main() {
	source.Serve(map[string]plugin.Plugin{
		prometheus.PluginName: &source.Plugin{
			Source: kubernetes.NewSource(version),
		},
	})
}
