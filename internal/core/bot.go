// Package core
// Core functionalities for Dalian bot.
// /**
package core

import (
	"go.uber.org/zap"
)

const VERSION = "1.0.0-beta-1"

// Logger Global logger for Bot behaviors.
var Logger DalianLogger

// DalianLogger Logger instance for Dalian. struct reserved for more methods.
type DalianLogger struct {
	*zap.SugaredLogger
}

// Bot Dalian Bot instance.
// A Bot include two sorts of components, Service and Plugin.
// Service generates triggers, and Plugin received them and act.
// For more information about Service and Plugin, see their respective documentations.
type Bot struct {
	ServiceRegistry *ServiceRegistry // ServiceRegistry A Service provider maintained by Bot
	PluginRegistry  *PluginRegistry  // PluginRegistry A Plugin provider maintained by Bot
	DispatcherChan  <-chan Trigger   // DispatcherChan channel for dispatching Trigger to Plugin
}

func NewBot() *Bot {
	bot := &Bot{
		ServiceRegistry: NewServiceRegistry(),
		PluginRegistry:  NewPluginRegistry(),
	}
	return bot
}

func (b *Bot) Run() {
	b.DispatcherChan, _ = b.ServiceRegistry.InstallTriggerChanForAll()
	go func(ch <-chan Trigger) {
		for {
			if trigger, ok := <-ch; !ok {
				//channel closed
				return
			} else {
				//receives a trigger, dispatching...
				Logger.Debugf("trigger:%v", trigger)
				trigger.Bot = b //injecting bot object
				for _, plugin := range b.PluginRegistry.GetPlugins() {
					go plugin.Trigger(trigger)
				}
			}
		}
	}(b.DispatcherChan)
}

func (b *Bot) QuickRegisterPlugin(f func(reg *ServiceRegistry) IPlugin) error {
	plugin := f(b.ServiceRegistry)
	return b.PluginRegistry.RegisterPlugin(plugin)
}

func (b *Bot) GracefulShutDown() {
	Logger.Infof("Received termination signal...")
	b.ServiceRegistry.StopAll()
}

type MessengerConfig struct {
	Prefix    string
	Separator string
	BotID     string
}
