package helm

import (
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/executor"
)

type Config struct {}

func MergeConfigs(configs []*executor.Config) (Config, error) {
	var out Config
	for _, rawCfg := range configs {
		var cfg Config
		err := yaml.Unmarshal(rawCfg.RawYAML, &cfg)
		if err != nil {
			return Config{}, err
		}

		// TODO: out...
	}

	return out, nil
}
