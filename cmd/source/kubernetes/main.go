package main

import (
	"context"
	"github.com/kubeshop/botkube/internal/source/kubernetes"
	"github.com/kubeshop/botkube/pkg/api/source"
	"os"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

func main() {
	/*source.Serve(map[string]plugin.Plugin{
		prometheus.PluginName: &source.Plugin{
			Source: kubernetes.NewSource(version),
		},
	})*/
	file, err := os.ReadFile("/tmp/kube-config.yaml")
	if err != nil {
		return
	}
	s := kubernetes.NewSource("dev")
	_, err = s.Stream(context.Background(), source.StreamInput{Configs: []*source.Config{
		{
			RawYAML: file,
		},
	}})
	if err != nil {
		return
	}
}
