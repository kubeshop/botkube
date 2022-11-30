package source

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestStartingUniqueProcesses(t *testing.T) {
	// given
	givenCfg, _, err := config.LoadWithDefaults(func() []string {
		return []string{
			testdataFile(t, "config.yaml"),
		}
	})
	require.NoError(t, err)

	logger, _ := logtest.NewNullLogger()

	expectedProcesses := map[string]struct{}{
		"botkube/keptn@v1.0.0; keptn-us-east-2; keptn-eu-central-1": {},
		"botkube/keptn@v1.0.0; keptn-eu-central-1; keptn-us-east-2": {},
		"botkube/keptn@v1.0.0; keptn-eu-central-1":                  {},
		"botkube/keptn@v1.0.0; keptn-us-east-2":                     {},
	}

	assertStarter := func(ctx context.Context, pluginName string, pluginConfigs []any, sources []string) error {
		// then configs are specified in a proper order
		var expConfigs []any
		for _, sourceName := range sources {
			expConfigs = append(expConfigs, givenCfg.Sources[sourceName].Plugins[pluginName].Config)
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
	dispatcher := NewDispatcher(logger, nil, nil, givenCfg)
	dispatcher.starter = assertStarter

	err = dispatcher.Start(context.Background())
	require.NoError(t, err)
}

func testdataFile(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", t.Name(), name)
}
