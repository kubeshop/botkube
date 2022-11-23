package e2e

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/internal/plugin"
)

const (
	executorBinaryPrefix = "executor_"
	indexFileEndpoint    = "/botkube.yaml"
)

type PluginsConfig struct {
	BinariesDirectory string `envconfig:"default=dist"`
	Server            struct {
		Host string `envconfig:"default=http://host.k3d.internal"`
		Port int    `envconfig:"default=3000"`
	}
}

func NewPluginServer(cfg PluginsConfig) (string, func() error) {
	fs := http.FileServer(http.Dir(cfg.BinariesDirectory))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	basePath := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	http.HandleFunc(indexFileEndpoint, func(w http.ResponseWriter, _ *http.Request) {
		idx, err := buildIndex(basePath, cfg.BinariesDirectory)
		if err != nil {
			log.Printf("Cannot build index file: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		out, err := yaml.Marshal(idx)
		if err != nil {
			log.Printf("Cannot marshall index file: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		_, err = w.Write(out)
		if err != nil {
			log.Printf("Cannot send marshalled index file: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Listening on %s...", addr)

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 3 * time.Second,
	}

	return basePath + indexFileEndpoint, func() error {
		return server.ListenAndServe()
	}
}

func buildIndex(urlBasePath string, dir string) (plugin.Index, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return plugin.Index{}, err
	}

	entries := map[string]plugin.IndexEntry{}
	for _, entry := range files {
		if entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), executorBinaryPrefix) {
			continue
		}

		name := strings.TrimPrefix(entry.Name(), executorBinaryPrefix)
		name, _, _ = strings.Cut(name, "_")

		item, found := entries[name]
		if !found {
			item = plugin.IndexEntry{
				Name:        name,
				Type:        plugin.TypeExecutor,
				Description: "Executor description",
				Version:     "0.1.0",
			}
		}
		item.Links = append(item.Links, fmt.Sprintf("%s/static/%s", urlBasePath, entry.Name()))
		entries[name] = item
	}

	var out plugin.Index
	for _, item := range entries {
		out.Entries = append(out.Entries, item)
	}
	return out, nil
}
