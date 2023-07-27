package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kubeshop/botkube/test/fake"
)

func main() {
	dir, err := os.Getwd()
	exitOnErr(err)

	host := os.Getenv("PLUGIN_SERVER_HOST")
	port := os.Getenv("PLUGIN_SERVER_PORT")
	if host == "" {
		host = "http://localhost"
	}
	if port == "" {
		port = "3010"
	}
	portInt, err := strconv.Atoi(port)
	exitOnErr(err)

	binDir := filepath.Join(dir, "plugin-dist")
	indexEndpoint, startServerFn := fake.NewPluginServer(fake.PluginConfig{
		BinariesDirectory: binDir,
		Server: fake.PluginServer{
			Host: host,
			Port: portInt,
		},
	})

	log.Printf("Service plugin binaries from %s\n", binDir)
	log.Printf("Botkube repository index URL: %s", indexEndpoint)
	err = startServerFn()
	exitOnErr(err)
}

func exitOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
