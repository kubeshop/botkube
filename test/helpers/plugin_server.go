package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/kubeshop/botkube/test/e2e/fake"
)

func main() {
	dir, err := os.Getwd()
	exitOnErr(err)

	binDir := filepath.Join(dir, "dist")
	indexEndpoint, startServerFn := fake.NewPluginServer(fake.PluginConfig{
		BinariesDirectory: binDir,
		Server: fake.PluginServer{
			Host: "http://localhost",
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
