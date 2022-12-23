package main

import (
	"dalian-bot/internal/pkg/core"
	"dalian-bot/internal/pkg/plugins"
	"dalian-bot/internal/pkg/services/data"
	"dalian-bot/internal/pkg/services/ddtv"
	"dalian-bot/internal/pkg/services/discord"
	"dalian-bot/internal/pkg/services/web"
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"os"
	"os/signal"
	"syscall"
)

const VERSION = "1.0.0-alpha-1.3"

func main() {

	logger, _ := zap.NewDevelopment()
	core.Logger = core.DalianLogger{SugaredLogger: logger.Sugar()}
	core.Logger.Infof("Dalian core logger initialized!")

	/* Read Config files */
	var cred = new(Cred)
	if err := GetCred(cred, "config/credentials.yaml"); err != nil {
		core.Logger.Panicf("failed opening credentials file!")
		panic("no credentials!")
	}

	dalianBot := core.NewBot()

	webService := web.Service{ServiceConfig: web.ServiceConfig{TrustedProxies: []string{"165.232.129.202"}}}
	webService.Init(dalianBot.ServiceRegistry)
	ddtvService := ddtv.Service{}
	ddtvService.Init(dalianBot.ServiceRegistry)
	dataService := data.Service{ServiceConfig: data.ServiceConfig{URI: cred.Uri}}
	dataService.Init(dalianBot.ServiceRegistry)
	discordService := discord.Service{ServiceConfig: discord.ServiceConfig{Token: cred.Token}}
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

type Cred struct {
	Discord
	Mongo
}

type Discord struct {
	Token string `yaml:"token"`
}

type Mongo struct {
	Uri string `yaml:"uri"`
}

func GetCred(cred *Cred, fileLocation string) error {
	yamlFile, err := os.ReadFile(fileLocation)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, cred)
	if err != nil {
		return err
	}
	return nil
}
