package fake

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/internal/plugin"
)

const indexFileEndpoint = "/botkube.yaml"

type (
	// PluginConfig holds configuration for fake plugin server.
	PluginConfig struct {
		BinariesDirectory string
		Server            PluginServer
	}

	// PluginServer holds configuration for HTTP plugin server.
	PluginServer struct {
		Host string `envconfig:"default=http://host.k3d.internal"`
		Port int    `envconfig:"default=3000"`
	}
)

// NewPluginServer return function to start the fake plugin HTTP server.
func NewPluginServer(cfg PluginConfig) (string, func() error) {
	fs := http.FileServer(http.Dir(cfg.BinariesDirectory))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	basePath := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	builder := plugin.NewIndexBuilder(logrus.New())

	http.HandleFunc(indexFileEndpoint, func(w http.ResponseWriter, _ *http.Request) {
		idx, err := builder.Build(cfg.BinariesDirectory, basePath+"/static")
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
