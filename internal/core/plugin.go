package core

import (
	"fmt"
	"reflect"
)

// PluginRegistry Plugin controller embedded in the Bot.
type PluginRegistry struct {
	plugins     map[reflect.Type]IPlugin // store valid plugin instances
	pluginTypes []reflect.Type           // record plguin regsitration order
}

// NewPluginRegistry Return a raw PluginRegistry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{plugins: make(map[reflect.Type]IPlugin)}
}

// RegisterPlugin Register plugin to registry.
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

// GetPlugins Get all plugins registered.
func (s *PluginRegistry) GetPlugins() map[reflect.Type]IPlugin {
	return s.plugins
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

// AcceptTrigger Return if the Plugin accept certain type of Trigger.
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

	Init(reg *ServiceRegistry) error // should be implemented
	Trigger(trigger Trigger)         // should be implemented.
}
