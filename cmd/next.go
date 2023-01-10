package main

import (
	"dalian-bot/internal/core"
	"dalian-bot/internal/plugins"
	"dalian-bot/internal/services/data"
	"dalian-bot/internal/services/ddtv"
	"dalian-bot/internal/services/discord"
	"dalian-bot/internal/services/web"
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
	cred, err := core.GetCred("config/credentials.yaml")
	if err != nil {
		panic("credential test failed")
	}

	dalianBot := core.NewBot()

	webService := web.Service{ServiceConfig: web.ServiceConfig{TrustedProxies: []string{"165.232.129.202"}}}
	webService.Init(dalianBot.ServiceRegistry)
	ddtvService := ddtv.Service{}
	ddtvService.Init(dalianBot.ServiceRegistry)
	dataService := data.Service{ServiceConfig: data.ServiceConfig{URI: cred.MongoURI.Value}}
	dataService.Init(dalianBot.ServiceRegistry)
	discordService := discord.Service{ServiceConfig: discord.ServiceConfig{Token: cred.DiscordToken.Value}}
	discordService.Init(dalianBot.ServiceRegistry)

	dalianBot.ServiceRegistry.StartAll()

	dalianBot.QuickRegisterPlugin(plugins.NewPingPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewWhatPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewHelpPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewDDTVPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewArchivePlugin)

	dalianBot.Run()
	//graceful shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dalianBot.GracefulShutDown()
}
