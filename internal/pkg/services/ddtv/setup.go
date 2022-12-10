package ddtv

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func InitDDTVHook(engine *gin.Engine) error {
	engine.POST("/ddtv/webhook", handleFunc)
	return nil
}

func handleFunc(c *gin.Context) {
	var hook WebHook

	//debug
	fmt.Println("hook found")
	if err := c.BindJSON(&hook); err != nil {
		//failed. malformed.
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusOK)
}
