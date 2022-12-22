package source

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	intConfig "github.com/kubeshop/botkube/internal/config"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
)

func TestStartingUniqueProcesses(t *testing.T) {
	// given
	files := intConfig.YAMLFiles{
		readTestdataFile(t, "config.yaml"),
	}
	givenCfg, _, err := config.LoadWithDefaults(files)
	require.NoError(t, err)

	expectedProcesses := map[string]struct{}{
		"botkube/keptn@v1.0.0; keptn-us-east-2; keptn-eu-central-1": {},
		"botkube/keptn@v1.0.0; keptn-eu-central-1; keptn-us-east-2": {},
		"botkube/keptn@v1.0.0; keptn-eu-central-1":                  {},
		"botkube/keptn@v1.0.0; keptn-us-east-2":                     {},
	}

	assertStarter := func(ctx context.Context, pluginName string, pluginConfigs []*source.Config, sources []string) error {
		// then configs are specified in a proper order
		var expConfigs []*source.Config
		for _, sourceName := range sources {
			expConfigs = append(expConfigs, &source.Config{
				RawYAML: mustYAMLMarshal(t, givenCfg.Sources[sourceName].Plugins[pluginName].Config),
			})
		}
		assert.Equal(t, expConfigs, pluginConfigs)

		// then only unique process are started
		key := []string{pluginName}
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
	scheduler := NewScheduler(loggerx.NewNoop(), givenCfg, fakeDispatcherFunc(assertStarter))

	err = scheduler.Start(context.Background())
	require.NoError(t, err)
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
type fakeDispatcherFunc func(ctx context.Context, pluginName string, pluginConfigs []*source.Config, sources []string) error

// ServeHTTP calls f(w, r).
func (f fakeDispatcherFunc) Dispatch(ctx context.Context, pluginName string, pluginConfigs []*source.Config, sources []string) error {
	return f(ctx, pluginName, pluginConfigs, sources)
}
