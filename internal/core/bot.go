// Package core
// Core functionalities for Dalian bot.
// /**
package core

import (
	"go.uber.org/zap"
)

const VERSION = "2.0.0"

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
	DispatcherChan  chan Trigger     // DispatcherChan channel for dispatching Trigger to Plugin

	//todo: add a channel that listening to auditing messages, or add an AuditService in Bot
}

// NewBot prepare a bot template to be filled.
func NewBot() *Bot {
	bot := &Bot{
		ServiceRegistry: NewServiceRegistry(),
		PluginRegistry:  NewPluginRegistry(),
	}
	return bot
}

// Run start the bot and listen to incoming events.
func (b *Bot) Run() {
	b.DispatcherChan, _ = b.ServiceRegistry.InstallTriggerChanForAll()
	//loop until channel closed.
	go func(ch <-chan Trigger) {
		for {
			if trigger, ok := <-ch; !ok {
				//channel closed
				return
			} else {
				// receives a trigger, dispatching...
				Logger.Debugf("trigger:%v", trigger)
				trigger.Bot = b //inject bot instance to Trigger.
				for _, plugin := range b.PluginRegistry.GetPlugins() {
					go plugin.Trigger(trigger)
				}
			}
		}
	}(b.DispatcherChan)
}

// QuickRegisterPlugin a shortcut for initialize and register a plugin to bot.
func (b *Bot) QuickRegisterPlugin(f func(reg *ServiceRegistry) IPlugin) error {
	plugin := f(b.ServiceRegistry)
	return b.PluginRegistry.RegisterPlugin(plugin)
}

// GracefulShutDown close all services before shutdown.
func (b *Bot) GracefulShutDown() {
	Logger.Infof("Received termination signal...")
	close(b.DispatcherChan)     // close trigger channel
	b.ServiceRegistry.StopAll() // stop all services.
}

// MessengerConfig Basic config for messenger, records command prefix, separator, and botID.
type MessengerConfig struct {
	Prefix    string
	Separator string
	BotID     string
}
