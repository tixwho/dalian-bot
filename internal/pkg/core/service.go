package core

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
)

type ServiceRegistry struct {
	services     map[reflect.Type]Service
	serviceTypes []reflect.Type
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{services: make(map[reflect.Type]Service)}
}

func (s *ServiceRegistry) RegisterService(service Service) error {
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		return fmt.Errorf("service already exists: %v", kind)
	}
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
	return nil
}

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

func (s *ServiceRegistry) InstallTriggerChanForAll() (ch <-chan Trigger, err error) {
	triggerChan := make(chan Trigger, 100)
	for _, kind := range s.services {
		if triggerable, canTrigger := kind.(ITrigggerable); canTrigger {
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

type Service interface {
	Name() string
	Init(reg *ServiceRegistry) error
	Start(wg *sync.WaitGroup)
	Stop(wg *sync.WaitGroup) error
	Status() error
}

type TriggerType string

const (
	TriggerTypeDDTV    TriggerType = "ddtv"
	TriggerTypeDiscord             = "discord"
)

type Trigger struct {
	Type  TriggerType
	Bot   *Bot
	Event any //deligate to services for deref
}

type ITrigggerable interface {
	InstallTriggerChan(chan<- Trigger)
}

type TriggerableEmbedUtil struct {
	TriggerChan chan<- Trigger
}

func (t *TriggerableEmbedUtil) InstallTriggerChan(triggers chan<- Trigger) {
	t.TriggerChan = triggers
}

const (
	LogPromptUnknownTrigger = "received an unknown trigger type %s, check your [AcceptedTriggerTypes] setting!"
)
