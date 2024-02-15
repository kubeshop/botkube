package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

func main() {
	pluginsDir := flag.String("plugins-dir", getEnv("PLUGINS_DIR", "plugin-dist"), "Plugins directory")
	host := flag.String("host", getEnv("PLUGIN_SERVER_HOST", "http://localhost"), "Local server host")
	port := flag.String("port", getEnv("PLUGIN_SERVER_PORT", "3010"), "Local server port")
	flag.Parse()

	dir, err := os.Getwd()
	loggerx.ExitOnError(err, "while getting current directory")

	portInt, err := strconv.Atoi(*port)
	loggerx.ExitOnError(err, "while casting server port value")

	binDir := filepath.Join(dir, *pluginsDir)
	indexEndpoint, startServerFn := pluginx.NewStaticPluginServer(pluginx.StaticPluginServerConfig{
		BinariesDirectory: binDir,
		Host:              *host,
		Port:              portInt,
	})

	log.Printf("Service plugin binaries from %s\n", binDir)
	log.Printf("Botkube repository index URL: %s", indexEndpoint)
	err = startServerFn()
	loggerx.ExitOnError(err, "while starting server")
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
