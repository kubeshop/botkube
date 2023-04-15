package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/plugin"
)

const filePerm = 0o644

func main() {
	var (
		urlBasePath      = flag.String("url-base-path", os.Getenv("PLUGIN_DOWNLOAD_URL_BASE_PATH"), "Defines the URL base path for downloading the plugin binaries")
		binsDir          = flag.String("binaries-path", "./plugin-dist", "Defines the local path to plugins binaries folder")
		output           = flag.String("output-path", "./plugins-index.yaml", "Defines the local path where index YAML should be saved")
		pluginNameFilter = flag.String("plugin-name-filter", "", "Defines the plugin name regex for plugins which should be included in the index. Other plugins will be skipped.")
	)

	flag.Parse()
	logger := logrus.New()

	idxBuilder := plugin.NewIndexBuilder(logger)

	absBinsDir, err := filepath.Abs(*binsDir)
	loggerx.ExitOnError(err, "while resolving an absolute path of binaries folder")

	log := logger.WithFields(logrus.Fields{
		"binDir":           absBinsDir,
		"urlBasePath":      *urlBasePath,
		"pluginNameFilter": *pluginNameFilter,
	})

	log.Info("Building index..")
	idx, err := idxBuilder.Build(absBinsDir, *urlBasePath, *pluginNameFilter)
	loggerx.ExitOnError(err, "while building plugin index")

	raw, err := yaml.Marshal(idx)
	loggerx.ExitOnError(err, "while marshaling index into YAML format")

	logger.WithField("output", *output).Info("Saving index file...")
	err = os.WriteFile(*output, raw, filePerm)
	loggerx.ExitOnError(err, "while saving index file")
}
