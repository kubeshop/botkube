package pluginx

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/api"
)

const pluginsDir = "/tmp/plugins"

// DownloadDependencies downloads input dependencies into plugins directory.
func DownloadDependencies(in map[string]api.Dependency) error {
	selector := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	for depName, dep := range in {
		depPath := filepath.Join(pluginsDir, depName)
		if plugin.DoesBinaryExist(depPath) {
			continue
		}

		depURL, found := dep.URLs[selector]
		if !found {
			return fmt.Errorf("cannot find download url for %s platform for a dependency %q", selector, depName)
		}

		err := plugin.DownloadBinary(context.Background(), depPath, depURL)
		if err != nil {
			return fmt.Errorf("while downloading dependency %q for %q: %w", depName, pluginsDir, err)
		}
	}

	return nil
}
