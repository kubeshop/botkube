package plugin

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/loggerx"
)

const indexFileEndpoint = "/botkube.yaml"

type (
	// StaticPluginServerConfig holds configuration for fake plugin server.
	StaticPluginServerConfig struct {
		BinariesDirectory string
		Host              string `envconfig:"default=http://host.k3d.internal"`
		Port              int    `envconfig:"default=3000"`
	}
)

// NewStaticPluginServer return function to start the static plugin HTTP server suitable for local development or e2e tests.
func NewStaticPluginServer(cfg StaticPluginServerConfig) (string, func() error) {
	fs := http.FileServer(http.Dir(cfg.BinariesDirectory))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	basePath := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	builder := NewIndexBuilder(loggerx.NewNoop())

	http.HandleFunc(indexFileEndpoint, func(w http.ResponseWriter, _ *http.Request) {
		isArchive := os.Getenv("OUTPUT_MODE") == "archive"
		idx, err := builder.Build(cfg.BinariesDirectory, basePath+"/static", ".*", true, isArchive)
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

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listening on %s...", addr)

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 3 * time.Second,
	}

	return basePath + indexFileEndpoint, func() error {
		return server.ListenAndServe()
	}
}
