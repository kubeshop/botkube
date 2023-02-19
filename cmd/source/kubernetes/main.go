package main

import (
	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/internal/source/kubernetes"
	"github.com/kubeshop/botkube/pkg/api/source"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

func main() {
	source.Serve(map[string]plugin.Plugin{
		kubernetes.PluginName: &source.Plugin{
			Source: kubernetes.NewSource(version),
		},
	})
	/*file, err := os.ReadFile("/Users/huseyin/kube-config.yaml")
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
	}*/
}
