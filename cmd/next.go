package main

import (
	"dalian-bot/internal/conf"
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
	cred, err := conf.GetCred("config/credentials.yaml")
	if err != nil {
		panic("credential test failed")
	}

	/* Generate bot template */
	dalianBot := core.NewBot()

	/* Initialize & register services */
	webService := web.Service{ServiceConfig: web.ServiceConfig{TrustedProxies: []string{"165.232.129.202"}}}
	webService.Init(dalianBot.ServiceRegistry)
	ddtvService := ddtv.Service{}
	ddtvService.Init(dalianBot.ServiceRegistry)
	dataService := data.Service{ServiceConfig: data.ServiceConfig{URI: cred.MongoURI.Value}}
	dataService.Init(dalianBot.ServiceRegistry)
	discordService := discord.Service{ServiceConfig: discord.ServiceConfig{Token: cred.DiscordToken.Value}}
	discordService.Init(dalianBot.ServiceRegistry)

	dalianBot.ServiceRegistry.StartAll()

	/* Initialize & register plugins*/
	dalianBot.QuickRegisterPlugin(plugins.NewPingPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewWhatPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewHelpPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewDDTVPlugin)
	dalianBot.QuickRegisterPlugin(plugins.NewArchivePlugin)

	/* Startup */
	dalianBot.Run()

	/* Lock main thread */
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	/* Graceful Shutdown */
	dalianBot.GracefulShutDown()
}
