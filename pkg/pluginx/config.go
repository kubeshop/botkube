package pluginx

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"

	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// MergeExecutorConfigs merges input configuration into a given destination.
// Destination MUST be a pointer to a struct.
// To learn more about merge strategy, visit:
//
//	https://github.com/knadh/koanf#merge-behavior
func MergeExecutorConfigs(in []*executor.Config, dest any) error {
	return mergeConfigs(executorConfigs(in), dest)
}

// MergeSourceConfigs merges input configuration into a given destination.
// Destination MUST be a pointer to a struct.
// To learn more about merge strategy, visit:
//
//	https://github.com/knadh/koanf#merge-behavior
func MergeSourceConfigs(in []*source.Config, dest any) error {
	return mergeConfigs(sourceConfigs(in), dest)
}

func mergeConfigs(configs enumerable, dest any) error {
	k := koanf.New(".")

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

	err := k.Unmarshal("", dest)
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
