package main

import (
	"fmt"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/controller"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/infracloudio/botkube/pkg/slack"
)

func main() {
	log.Logger.Info("Starting controller")
	Config, err := config.New()
	if err != nil {
		log.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}
	log.Logger.Info(fmt.Sprintf("Configuration:: %+v\n", Config))

	if Config.Communications.Slack.Enable {
		sb := slack.NewSlackBot()
		go sb.Start()
	}

	controller.RegisterInformers(Config)
}
