package main

import (
	"fmt"
	"log"
	"net/http"

	"botkube.io/demo/internal/quote"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

// Config holds the application configuration.
type Config struct {
	HTTPPort int      `envconfig:"default=8080"`
	Quotes   []string `envconfig:"optional"`
}

func main() {
	var cfg Config
	err := envconfig.Init(&cfg)
	exitOnError(err)

	logger := logrus.New().WithField("service", "quote")
	quoteProvider := quote.NewGenerator(cfg.Quotes)
	h := quote.NewHandler(logger, quoteProvider)
	http.HandleFunc("/quote", h.GetRandomQuoteHandler)

	logger.Infof("Server started on port %d", cfg.HTTPPort)
	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTPPort), nil)
	exitOnError(err)
}

func exitOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
