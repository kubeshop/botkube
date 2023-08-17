package pluginx

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/api/source"
)

type (
	exampleConfig struct {
		HelmDriver    string `yaml:"helmDriver"`
		HelmCacheDir  string `yaml:"helmCacheDir"`
		HelmConfigDir string `yaml:"helmConfigDir"`
		// yaml tag is on purpose different from field name
		ExampleList []string         `yaml:"list"`
		NestedProp  configNestedProp `yaml:"nestedProp"`
	}
	configNestedProp struct {
		Value string `yaml:"value"`
		Key   string `yaml:"key"`
	}
)

var (
	fixInputConfig = []byte(
		heredoc.Doc(`
            helmDriver: "configmap"
            helmCacheDir: "/mnt/test"
            list: ["item1"]
            nestedProp:
              key: "cfg-key"
			`))

	fixDefaultConfig = exampleConfig{
		HelmDriver:    "secret",
		HelmCacheDir:  "/tmp/helm/.cache",
		HelmConfigDir: "/tmp/helm/",
		ExampleList:   []string{"default-item"},
		NestedProp: configNestedProp{
			Value: "default-val",
		},
	}
)

func TestMergeExecutorConfigWithDefaults(t *testing.T) {
	// given
	cfg := &executor.Config{RawYAML: fixInputConfig}

	// when
	err, out := MergeExecutorConfigWithDefaults[exampleConfig](fixDefaultConfig, cfg)

	// then
	require.NoError(t, err)
	assertExpExampleConfigWithDefaults(t, out)
}

func TestLoadExecutorConfig(t *testing.T) {
	// given
	cfg := &executor.Config{RawYAML: fixInputConfig}

	// when
	err, out := LoadExecutorConfig[exampleConfig](cfg)

	// then
	require.NoError(t, err)
	assertExpExampleConfig(t, out)
}

func TestMergeSourceConfigWithDefaults(t *testing.T) {
	// given
	var cfg = &source.Config{RawYAML: fixInputConfig}

	// when
	err, out := MergeSourceConfigWithDefaults[exampleConfig](fixDefaultConfig, cfg)

	// then
	require.NoError(t, err)
	assertExpExampleConfigWithDefaults(t, out)
}

func TestMergeSourceConfigs(t *testing.T) {
	// given
	var cfg = &source.Config{RawYAML: fixInputConfig}

	// when
	err, out := LoadSourceConfig[exampleConfig](cfg)

	// then
	require.NoError(t, err)
	assertExpExampleConfig(t, out)
}

func assertExpExampleConfig(t *testing.T, got exampleConfig) {
	t.Helper()

	assert.Equal(t, "configmap", got.HelmDriver)
	assert.Equal(t, "/mnt/test", got.HelmCacheDir)
	assert.Equal(t, "cfg-key", got.NestedProp.Key)
	assert.Equal(t, []string{"item1"}, got.ExampleList)
	assert.Empty(t, got.HelmConfigDir)
	assert.Empty(t, got.NestedProp.Value)
}

func assertExpExampleConfigWithDefaults(t *testing.T, got exampleConfig) {
	t.Helper()

	assert.Equal(t, "configmap", got.HelmDriver)
	assert.Equal(t, "/mnt/test", got.HelmCacheDir)
	assert.Equal(t, "/tmp/helm/", got.HelmConfigDir)
	assert.Equal(t, []string{"item1"}, got.ExampleList)
	assert.Equal(t, "default-val", got.NestedProp.Value)
	assert.Equal(t, "cfg-key", got.NestedProp.Key)
}
