package core

import (
	"fmt"
	"reflect"
)

type PluginRegistry struct {
	plugins     map[reflect.Type]INewPlugin
	pluginTypes []reflect.Type
}

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{plugins: make(map[reflect.Type]INewPlugin)}
}

func (s *PluginRegistry) RegisterPlugin(plugin INewPlugin) error {
	kind := reflect.TypeOf(plugin)
	if _, exists := s.plugins[kind]; exists {
		return fmt.Errorf("plugin already exists: %v", kind)
	}
	s.plugins[kind] = plugin
	s.pluginTypes = append(s.pluginTypes, kind)
	Logger.Debugf("Plugin [%s] registered.", kind)
	return nil
}

func (s *PluginRegistry) GetPlugins() map[reflect.Type]INewPlugin {
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

// INewPlugin temporal use. wll be replaced
type INewPlugin interface {
	// GetName All command must have a name (unique identifier)
	GetName() string
	Init(reg *ServiceRegistry) error
	AcceptTrigger(t TriggerType) bool
	Trigger(trigger Trigger)
}
