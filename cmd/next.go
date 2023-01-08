package main

import (
	core2 "dalian-bot/internal/core"
	plugins2 "dalian-bot/internal/plugins"
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
	core2.Logger = core2.DalianLogger{SugaredLogger: logger.Sugar()}
	core2.Logger.Infof("Dalian core logger initialized!")

	/* Read Config files */
	/*
		var cred = new(core.Cred)
		if err := core.GetCred(cred, "config/credentials.yaml"); err != nil {
			core.Logger.Panicf("failed opening credentials file!")
			panic("no credentials!")
		}

	*/
	cred, err := core2.GetCredNew("config/credentials.yaml")
	if err != nil {
		panic("credential test failed")
	}

	dalianBot := core2.NewBot()

	webService := web.Service{ServiceConfig: web.ServiceConfig{TrustedProxies: []string{"165.232.129.202"}}}
	webService.Init(dalianBot.ServiceRegistry)
	ddtvService := ddtv.Service{}
	ddtvService.Init(dalianBot.ServiceRegistry)
	dataService := data.Service{ServiceConfig: data.ServiceConfig{URI: cred.MongoURI.Value}}
	dataService.Init(dalianBot.ServiceRegistry)
	discordService := discord.Service{ServiceConfig: discord.ServiceConfig{Token: cred.DiscordToken.Value}}
	discordService.Init(dalianBot.ServiceRegistry)

	dalianBot.ServiceRegistry.StartAll()

	dalianBot.QuickRegisterPlugin(plugins2.NewPingPlugin)
	dalianBot.QuickRegisterPlugin(plugins2.NewWhatPlugin)
	dalianBot.QuickRegisterPlugin(plugins2.NewHelpPlugin)
	dalianBot.QuickRegisterPlugin(plugins2.NewDDTVPlugin)
	dalianBot.QuickRegisterPlugin(plugins2.NewArchivePlugin)

	dalianBot.Run()
	//graceful shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dalianBot.GracefulShutDown()
}
