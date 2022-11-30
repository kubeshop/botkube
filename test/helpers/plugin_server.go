package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/kubeshop/botkube/test/fake"
)

func main() {
	dir, err := os.Getwd()
	exitOnErr(err)

	host := os.Getenv("PLUGIN_SERVER_HOST")
	if host == "" {
		host = "http://localhost"
	}

	binDir := filepath.Join(dir, "dist")
	indexEndpoint, startServerFn := fake.NewPluginServer(fake.PluginConfig{
		BinariesDirectory: binDir,
		Server: fake.PluginServer{
			Host: host,
			Port: 3000,
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
