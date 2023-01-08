package core

import (
	"fmt"
	"reflect"
)

type PluginRegistry struct {
	plugins     map[reflect.Type]IPlugin
	pluginTypes []reflect.Type
}

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{plugins: make(map[reflect.Type]IPlugin)}
}

func (s *PluginRegistry) RegisterPlugin(plugin IPlugin) error {
	kind := reflect.TypeOf(plugin)
	if _, exists := s.plugins[kind]; exists {
		return fmt.Errorf("plugin already exists: %v", kind)
	}
	s.plugins[kind] = plugin
	s.pluginTypes = append(s.pluginTypes, kind)
	Logger.Debugf("Plugin [%s] registered.", kind)
	return nil
}

func (s *PluginRegistry) GetPlugins() map[reflect.Type]IPlugin {
	return s.plugins
}

// ILateInitCommand For commands that need a late init process in lifecycle (i.g.) database
type ILateInitCommand interface {
	LateInit()
}

// Plugin Basic command struct with no function
type Plugin struct {
	Name                 string
	AcceptedTriggerTypes []TriggerType
}

// GetName Return the name (unique identifier) of the plugin.
func (p *Plugin) GetName() string {
	return p.Name
}

func (p *Plugin) AcceptTrigger(t TriggerType) bool {
	for _, acceptedType := range p.AcceptedTriggerTypes {
		if t == acceptedType {
			return true
		}
	}
	return false
}

// IPlugin Top-level plugin interface
type IPlugin interface {

	// GetName all plugins have their name.
	GetName() string                  // provided by Plugin
	AcceptTrigger(t TriggerType) bool // provided by Plugin

	Init(reg *ServiceRegistry) error
	Trigger(trigger Trigger)
}
