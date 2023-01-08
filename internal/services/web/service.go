package web

import (
	core2 "dalian-bot/internal/core"
	"github.com/gin-gonic/gin"
	"reflect"
	"sync"
)

type Service struct {
	GinEngine *gin.Engine
	ServiceConfig
}

type ServiceConfig struct {
	TrustedProxies []string
}

func (s *Service) Name() string {
	return "web"
}

func (s *Service) Init(reg *core2.ServiceRegistry) error {
	/* Setup Api Server */
	engine := gin.Default()
	//allow only redirection
	engine.SetTrustedProxies(s.TrustedProxies)
	s.GinEngine = engine
	return reg.RegisterService(s)
}

func (s *Service) Start(wg *sync.WaitGroup) {
	go s.GinEngine.Run(":8740")
	core2.Logger.Debugf("Service [%s] is now online.", reflect.TypeOf(s))
	wg.Done()
}

func (s *Service) Stop(wg *sync.WaitGroup) error {
	core2.Logger.Debugf("Service [%s] is successfully closed.", reflect.TypeOf(s))
	wg.Done()
	return nil
}

func (s *Service) Status() error {
	//TODO implement me
	panic("implement me")
}
