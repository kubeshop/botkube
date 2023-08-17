package pluginx

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/providers/structs"

	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/api/source"
)

// MergeExecutorConfigWithDefaults merges input configuration into a given destination.
// Rules:
// - Default MUST be a Go object with the `yaml` tag. Alternatively, it can be nil if there are no defaults.
// - if `yaml:"omitempty"` tag is not specified, then empty fields are taken into account, and resets the previous value.
// - Merging strategy can be found here https://github.com/knadh/koanf#merge-behavior.
func MergeExecutorConfigWithDefaults[T any](defaults any, in *executor.Config) (error, T) {
	if in == nil {
		var out T
		return nil, out
	}

	return mergeConfigs[T](defaults, in.RawYAML)
}

// LoadExecutorConfig is a syntax sugar to unmarshal input configuration into a given type without any defaults.
func LoadExecutorConfig[T any](in *executor.Config) (error, T) {
	return MergeExecutorConfigWithDefaults[T](nil, in)
}

// LoadSourceConfig is a syntax sugar to unmarshal input configuration into a given type without any defaults.
func LoadSourceConfig[T any](in *source.Config) (error, T) {
	return MergeSourceConfigWithDefaults[T](nil, in)
}

// MergeSourceConfigWithDefaults merges input configuration into a given destination.
// Rules:
// - Default MUST be a Go object with the `yaml` tag. Alternatively, it can be nil if there are no defaults.
// - if `yaml:"omitempty"` tag is not specified, then empty fields are taken into account, and resets the previous value.
// - Merging strategy can be found here https://github.com/knadh/koanf#merge-behavior.
func MergeSourceConfigWithDefaults[T any](defaults any, in *source.Config) (error, T) {
	if in == nil {
		var out T
		return nil, out
	}

	return mergeConfigs[T](defaults, in.RawYAML)
}

func mergeConfigs[T any](defaults any, config []byte) (error, T) {
	var dest T

	k := koanf.New(".")

	if defaults != nil {
		err := k.Load(structs.ProviderWithDelim(defaults, "yaml", "."), nil)
		if err != nil {
			return err, dest
		}
	}

	if config != nil {
		err := k.Load(rawbytes.Provider(config), yaml.Parser())
		if err != nil {
			return err, dest
		}
	}

	err := k.UnmarshalWithConf("", &dest, koanf.UnmarshalConf{
		Tag: "yaml",
	})
	if err != nil {
		return err, dest
	}
	return nil, dest
}
