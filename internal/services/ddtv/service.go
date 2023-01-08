package ddtv

import (
	core2 "dalian-bot/internal/core"
	"dalian-bot/internal/services/web"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"reflect"
	"sync"
	"time"
)

type Service struct {
	WebService *web.Service
	core2.TriggerableEmbedUtil
}

func (s *Service) Name() string {
	return "ddtv"
}

func (s *Service) Init(reg *core2.ServiceRegistry) error {
	var webSrv *web.Service
	if err := reg.FetchService(&webSrv); err != nil {
		return err
	}
	s.WebService = webSrv
	// todo: load the path from config
	s.WebService.GinEngine.POST("/ddtv/webhook", s.handleWebhook)
	return reg.RegisterService(s)
}

func (s *Service) Start(wg *sync.WaitGroup) {
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

// WOW, you can attach a struct!
func (s *Service) handleWebhook(c *gin.Context) {
	var hook WebHook

	//debug
	fmt.Println("hook found")
	if err := c.BindJSON(&hook); err != nil {
		core2.Logger.Warnf("Error: %v\r\n", err)
		//failed. malformed.
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	//fmt.Println(hook)
	fmt.Printf("Hook is: %d\r\n", hook.Type)
	switch hook.Type {
	case HookStartRec:
		//debug
		core2.Logger.Infof("Started REC for %s at %s\r\n", hook.UserInfo.Name, hook.HookTime.Format(time.RFC3339))
	case HookRecComplete:
		core2.Logger.Infof("Completed REC for %s at %s\r\n", hook.UserInfo.Name, hook.HookTime.Format(time.RFC3339))
	case HookRunShellComplete:
		core2.Logger.Infof("Completed Shell for %s at %s\r\n", hook.UserInfo.Name, hook.HookTime.Format(time.RFC3339))
	default:
		core2.Logger.Infof(fmt.Sprintf("Unknown event type: %d", hook.Type))
	}
	c.Status(http.StatusOK)
	t := core2.Trigger{
		Type: core2.TriggerTypeDDTV,
		Event: Event{
			EventType: EventTypeWebhook,
			WebHook:   hook,
		},
	}
	s.TriggerChan <- t
}
