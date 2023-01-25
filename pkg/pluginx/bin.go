package pluginx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/api"
)

const (
	// internalDir is a temporary solution to store the plugin dependency binaries.
	// Once the official downloader will be used, it will build a dedicated path.
	internalDir = "/tmp/internal-plugins"
	dirPerms    = 0o775
)

// DownloadDependencies downloads input dependencies into plugins directory.
// Deprecated: Since 0.18, plugin's dependency are downloaded automatically by Botkube.
// This method will be removed after releasing 0.18 version.
//
// Migration path:
//
//	The syntax for defining plugin dependency remains the same:
//	   var depsDownloadLinks = map[string]api.Dependency{
//	   	"gh": {
//	   		URLs: map[string]string{
//	   			"darwin/amd64": "https://github.com/cli/cli/releases/download/v2.21.2/gh_2.21.2_macOS_amd64.tar.gz//gh_2.21.2_macOS_amd64/bin",
//	   			"linux/amd64":  "https://github.com/cli/cli/releases/download/v2.21.2/gh_2.21.2_linux_amd64.tar.gz//gh_2.21.2_linux_amd64/bin",
//	   		},
//	   	},
//	   }
//
//	however, instead of calling download method:
//	  pluginx.DownloadDependencies(depsDownloadLinks)
//
//	move the `depsDownloadLinks` property under the MetadataOutput object:
//	  Metadata(context.Context) (api.MetadataOutput, error) {
//	  	return api.MetadataOutput{
//	  		// ...
//	  		Dependencies: depsDownloadLinks,
//	  	}, nil
//	  }
func DownloadDependencies(in map[string]api.Dependency) error {
	selector := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(filepath.Dir(internalDir), dirPerms)
	if err != nil {
		return fmt.Errorf("while creating directory where repository index should be stored: %w", err)
	}

	err = os.Setenv(plugin.DependencyDirEnvName, internalDir)
	if err != nil {
		return fmt.Errorf("while setting the dependency : %w", err)
	}

	for depName, dep := range in {
		depPath := filepath.Join(internalDir, depName)
		if plugin.DoesBinaryExist(depPath) {
			continue
		}

		depURL, found := dep.URLs[selector]
		if !found {
			return fmt.Errorf("cannot find download url for %s platform for a dependency %q", selector, depName)
		}

		err := plugin.DownloadBinary(context.Background(), depPath, depURL)
		if err != nil {
			return fmt.Errorf("while downloading dependency %q for %q: %w", depName, internalDir, err)
		}
	}

	return nil
}
