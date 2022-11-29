package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"

	"botkube.io/demo/internal/meme"
)

// Config holds the application configuration.
type Config struct {
	HTTPPort        int    `envconfig:"PORT,default=9090"`
	QuoteServiceUrl string `envconfig:"QUOTE_URL"`
}

func main() {
	var cfg Config
	err := envconfig.Init(&cfg)
	exitOnError(err)

	logger := logrus.New().WithField("service", "meme")
	quoteClient := meme.NewQuoteClient(cfg.QuoteServiceUrl)

	err = retry.Do(
		func() error {
			_, err := quoteClient.Get()
			return err
		},
		retry.OnRetry(func(n uint, err error) {
			logger.Errorf("Check connection health... (attempt no %d): %s", n+1, err)
		}),
		retry.Attempts(10),
		retry.LastErrorOnly(true),
	)
	exitOnError(err)

	h := meme.NewHandler(logger, meme.NewGenerator(quoteClient))
	http.HandleFunc("/meme", h.GetRandomMemeHandler)

	logger.Infof("Server started on port %d", cfg.HTTPPort)
	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTPPort), nil)
	exitOnError(err)
}

func exitOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
