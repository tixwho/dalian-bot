package main

import (
	"dalian-bot/internal/pkg/core"
	"dalian-bot/internal/pkg/plugins"
	"dalian-bot/internal/pkg/services/data"
	"dalian-bot/internal/pkg/services/ddtv"
	"dalian-bot/internal/pkg/services/discord"
	"dalian-bot/internal/pkg/services/web"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	logger, _ := zap.NewDevelopment()
	core.Logger = core.DalianLogger{SugaredLogger: logger.Sugar()}
	core.Logger.Infof("Dalian core logger initialized!")

	/* Read Config files */
	/*
		var cred = new(core.Cred)
		if err := core.GetCred(cred, "config/credentials.yaml"); err != nil {
			core.Logger.Panicf("failed opening credentials file!")
			panic("no credentials!")
		}

	*/
	cred, err := core.GetCredNew("config/credentials.yaml")
	if err != nil {
		panic("credential test failed")
	}

	dalianBot := core.NewBot()

	webService := web.Service{ServiceConfig: web.ServiceConfig{TrustedProxies: []string{"165.232.129.202"}}}
	webService.Init(dalianBot.ServiceRegistry)
	ddtvService := ddtv.Service{}
	ddtvService.Init(dalianBot.ServiceRegistry)
	dataService := data.Service{ServiceConfig: data.ServiceConfig{URI: cred.MongoURI}}
	dataService.Init(dalianBot.ServiceRegistry)
	discordService := discord.Service{ServiceConfig: discord.ServiceConfig{Token: cred.DiscordToken}}
	discordService.Init(dalianBot.ServiceRegistry)

	dalianBot.ServiceRegistry.StartAll()

	dalianBot.QuickRegisterPlugin(plugins.NewPingPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewWhatPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewHelpPlugin)

	dalianBot.Run()
	//graceful shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dalianBot.GracefulShutDown()
}
