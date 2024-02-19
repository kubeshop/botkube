package plugin

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/providers/structs"

	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// MergeExecutorConfigs merges input configuration into a given destination.
// Rules:
// - Destination MUST be a pointer to a struct.
// - if `yaml:"omitempty"` tag is not specified, then empty fields are take into account, and resets previous value.
// - Merging strategy can be found here https://github.com/knadh/koanf#merge-behavior.
func MergeExecutorConfigs(in []*executor.Config, dest any) error {
	return mergeConfigs(nil, executorConfigs(in), dest)
}

// MergeExecutorConfigsWithDefaults merges input configuration into a given destination.
// Rules:
// - Destination MUST be a pointer to a struct.
// - Default MUST be a Go object with the `yaml` tag.
// - if `yaml:"omitempty"` tag is not specified, then empty fields are take into account, and resets previous value.
// - Merging strategy can be found here https://github.com/knadh/koanf#merge-behavior.
func MergeExecutorConfigsWithDefaults(defaults any, in []*executor.Config, dest any) error {
	return mergeConfigs(defaults, executorConfigs(in), dest)
}

// MergeSourceConfigs merges input configuration into a given destination.
// Rules:
// - Destination MUST be a pointer to a struct.
// - if `yaml:"omitempty"` tag is not specified, then empty fields are take into account, and resets previous value.
// - Merging strategy can be found here https://github.com/knadh/koanf#merge-behavior.
func MergeSourceConfigs(in []*source.Config, dest any) error {
	return mergeConfigs(nil, sourceConfigs(in), dest)
}

// MergeSourceConfigsWithDefaults merges input configuration into a given destination.
// Rules:
// - Destination MUST be a pointer to a struct.
// - Default MUST be a Go object with the `yaml` tag.
// - if `yaml:"omitempty"` tag is not specified, then empty fields are take into account, and resets previous value.
// - Merging strategy can be found here https://github.com/knadh/koanf#merge-behavior.
func MergeSourceConfigsWithDefaults(defaults any, in []*source.Config, dest any) error {
	return mergeConfigs(defaults, sourceConfigs(in), dest)
}

func mergeConfigs(defaults any, configs enumerable, dest any) error {
	k := koanf.New(".")

	if defaults != nil {
		err := k.Load(structs.ProviderWithDelim(defaults, "yaml", "."), nil)
		if err != nil {
			return err
		}
	}
	issues := multierror.New()
	configs.Each(func(getter yamlConfigGetter) {
		err := k.Load(rawbytes.Provider(getter.GetRawYAML()), yaml.Parser())
		if err != nil {
			issues = multierror.Append(issues, err)
		}
	})

	if err := issues.ErrorOrNil(); err != nil {
		return err
	}

	err := k.UnmarshalWithConf("", dest, koanf.UnmarshalConf{
		Tag: "yaml",
	})
	if err != nil {
		return err
	}
	return nil
}

type (
	yamlConfigGetter interface {
		GetRawYAML() []byte
	}
	enumerable interface {
		Each(handler func(getter yamlConfigGetter))
	}
)

type executorConfigs []*executor.Config

func (as executorConfigs) Each(handler func(yamlConfigGetter)) {
	for _, a := range as {
		handler(a)
	}
}

type sourceConfigs []*source.Config

func (as sourceConfigs) Each(handler func(yamlConfigGetter)) {
	for _, a := range as {
		handler(a)
	}
}
