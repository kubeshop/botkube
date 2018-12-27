package main

import (
	"fmt"

	"github.com/infracloudio/kubeops/pkg/config"
	"github.com/infracloudio/kubeops/pkg/controller"
	log "github.com/infracloudio/kubeops/pkg/logging"
	"github.com/infracloudio/kubeops/pkg/slack"
)

func main() {
	log.Logger.Info("Starting controller")
	Config, err := config.New()
	if err != nil {
		log.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}
	log.Logger.Info(fmt.Sprintf("Configuration:: %+v\n", Config))

	sb := slack.NewSlackBot()
	go sb.Start()

	controller.RegisterInformers(Config)
}
