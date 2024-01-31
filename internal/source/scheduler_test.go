package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
)

func TestStartingUniqueProcesses(t *testing.T) {
	// given
	files := config.YAMLFiles{
		readTestdataFile(t, "config.yaml"),
	}
	givenCfg, _, err := config.LoadWithDefaults(files)
	require.NoError(t, err)

	expectedProcesses := map[string]struct{}{
		"botkube/keptn@v1.0.0; interactive/true; keptn-us-east-2; keptn-eu-central-1": {},
		"botkube/keptn@v1.0.0; interactive/true; keptn-eu-central-1; keptn-us-east-2": {},
		"botkube/keptn@v1.0.0; interactive/true; keptn-eu-central-1":                  {},
		"botkube/keptn@v1.0.0; interactive/true; keptn-us-east-2":                     {},

		"botkube/keptn@v1.0.0; interactive/false; keptn-us-east-2; keptn-eu-central-1": {},
		"botkube/keptn@v1.0.0; interactive/false; keptn-eu-central-1; keptn-us-east-2": {},
		"botkube/keptn@v1.0.0; interactive/false; keptn-eu-central-1":                  {},
		"botkube/keptn@v1.0.0; interactive/false; keptn-us-east-2":                     {},
	}

	assertStarter := func(ctx context.Context, isInteractivitySupported bool, pluginName string, pluginConfig *source.Config, sources []string) error {
		// then configs are specified in a proper order
		var expConfig = &source.Config{
			RawYAML: mustYAMLMarshal(t, givenCfg.Sources[sources[0]].Plugins[pluginName].Config),
		}
		assert.Equal(t, expConfig, pluginConfig)

		// then only unique process are started
		key := []string{pluginName, fmt.Sprintf("interactive/%v", isInteractivitySupported)}
		key = append(key, sources...)
		processKey := strings.Join(key, "; ")
		_, found := expectedProcesses[processKey]
		if !found {
			t.Errorf("starting unwanted process for %s with sources %v", pluginName, sources)
		}
		delete(expectedProcesses, processKey)
		return nil
	}

	// when
	scheduler := NewScheduler(context.Background(), loggerx.NewNoop(), givenCfg, fakeDispatcherFunc(assertStarter), make(chan string))

	err = scheduler.Start(context.Background())
	require.NoError(t, err)
}

func TestStartedProcesses(t *testing.T) {
	p := &openedStreams{}

	assert.False(t, p.isStartedStreamWithConfiguration("plugin1", "cfg1"))
	assert.False(t, p.isStartedStreamWithConfiguration("plugin1", "cfg2"))
	assert.False(t, p.isStartedStreamWithConfiguration("plugin1", "cfg3"))

	p.reportStartedStreamWithConfiguration("plugin1", "cfg1")
	p.reportStartedStreamWithConfiguration("plugin1", "cfg2")
	p.reportStartedStreamWithConfiguration("plugin1", "cfg3")
	p.reportStartedStreamWithConfiguration("plugin1", "cfg3")
	p.reportStartedStreamWithConfiguration("plugin1", "cfg3")

	assert.True(t, p.isStartedStreamWithConfiguration("plugin1", "cfg1"))
	assert.True(t, p.isStartedStreamWithConfiguration("plugin1", "cfg2"))
	assert.True(t, p.isStartedStreamWithConfiguration("plugin1", "cfg3"))

	p.deleteAllStartedStreamsForPlugin("plugin1")

	assert.False(t, p.isStartedStreamWithConfiguration("plugin1", "cfg1"))
	assert.False(t, p.isStartedStreamWithConfiguration("plugin1", "cfg2"))
	assert.False(t, p.isStartedStreamWithConfiguration("plugin1", "cfg3"))
}

func mustYAMLMarshal(t *testing.T, in any) []byte {
	raw, err := yaml.Marshal(in)
	require.NoError(t, err)
	return raw
}

func readTestdataFile(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", t.Name(), name)
	out, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	return out
}

// The fakeDispatcherFunc type is an adapter to allow the use of
// ordinary functions as Dispatcher handlers.
type fakeDispatcherFunc func(ctx context.Context, isInteractivitySupported bool, pluginName string, pluginConfig *source.Config, sources []string) error

// ServeHTTP calls f(w, r).
func (f fakeDispatcherFunc) Dispatch(dispatch PluginDispatch) error {
	return f(dispatch.ctx, dispatch.isInteractivitySupported, dispatch.pluginName, dispatch.pluginConfig, []string{dispatch.sourceName})
}
