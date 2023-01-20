package core

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
)

// ServiceRegistry Servcice controller embedded in the Bot.
type ServiceRegistry struct {
	services     map[reflect.Type]Service // store service instances
	serviceTypes []reflect.Type           // record service register orders.
}

// NewServiceRegistry Return a raw ServiceRegistry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{services: make(map[reflect.Type]Service)}
}

// RegisterService Register a service to registry.
func (s *ServiceRegistry) RegisterService(service Service) error {
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		return fmt.Errorf("service already exists: %v", kind)
	}
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
	return nil
}

// StartAll Run all registered services in separated goroutines.
func (s *ServiceRegistry) StartAll() {
	Logger.Debugf("Starting %d services: %v\r\n", len(s.serviceTypes), s.serviceTypes)
	wg := sync.WaitGroup{}
	wg.Add(len(s.serviceTypes))
	for _, kind := range s.serviceTypes {
		go s.services[kind].Start(&wg)
	}
	Logger.Debugf("Waiting for service...")
	wg.Wait()
	Logger.Infof("Finished starting all service! Services online now: %v", s.serviceTypes)
}

// InstallTriggerChanForAll Install main Trigger channel for all Services that are able to send Triggers.
func (s *ServiceRegistry) InstallTriggerChanForAll() (ch chan Trigger, err error) {
	triggerChan := make(chan Trigger, 100)
	for _, kind := range s.services {
		if triggerable, canTrigger := kind.(Trigggerable); canTrigger {
			triggerable.InstallTriggerChan(triggerChan)
		}
	}
	return triggerChan, nil
}

// StopAll ends every service in reverse order of registration, logging a
// panic if any of them fail to stop.
func (s *ServiceRegistry) StopAll() {
	wg := sync.WaitGroup{}
	wg.Add(len(s.serviceTypes))
	for i := len(s.serviceTypes) - 1; i >= 0; i-- {
		kind := s.serviceTypes[i]
		service := s.services[kind]
		if err := service.Stop(&wg); err != nil {
			log.Panicf("Could not stop the following service: %v, %v", kind, err)
		}
	}
	wg.Wait()
	Logger.Infof("ALL services stopped.")
}

// FetchService takes in a struct pointer and sets the value of that pointer
// to a service currently stored in the service registry. This ensures the input argument is
// set to the right pointer that refers to the originally registered service.
func (s *ServiceRegistry) FetchService(service interface{}) error {
	if reflect.TypeOf(service).Kind() != reflect.Ptr {
		return fmt.Errorf("provided type %s:%v", reflect.TypeOf(service), ErrServiceFetchNonPointer)
	}
	element := reflect.ValueOf(service).Elem()
	if running, ok := s.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return fmt.Errorf("provided type %s:%v", reflect.TypeOf(service), ErrServiceFetchUnknownService)
}

var (
	ErrServiceFetchNonPointer     = errors.New("input must be of pointer type")
	ErrServiceFetchUnknownService = errors.New("service not found in registry")
)

// Service Top-level service interface.
type Service interface {
	Name() string
	Init(reg *ServiceRegistry) error
	Start(wg *sync.WaitGroup)
	Stop(wg *sync.WaitGroup) error
	Status() error
}

// TriggerType Unique TriggerType, services use them to identify trigger options.
type TriggerType string

// Trigger An event that request bot responses.
// Typically consumed by registered Plugin for further analysis.
type Trigger struct {
	Type  TriggerType
	Bot   *Bot
	Event any //deligate to services for deref
}

// Trigggerable An interface that represent structs able to send triggers.
type Trigggerable interface {
	InstallTriggerChan(chan<- Trigger)
}

// TriggerableEmbedUtil An util that stores trigger channel.
type TriggerableEmbedUtil struct {
	TriggerChan chan<- Trigger
}

// InstallTriggerChan Install the given Trigger channel
func (t *TriggerableEmbedUtil) InstallTriggerChan(triggers chan<- Trigger) {
	t.TriggerChan = triggers
}

const (
	LogPromptUnknownTrigger = "received an unknown trigger type %s, check your [AcceptedTriggerTypes] setting!"
)
