package ddtv

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func InitDDTVHook(engine *gin.Engine) error {
	engine.POST("/ddtv/webhook", handleFunc)
	return nil
}

func handleFunc(c *gin.Context) {
	var hook WebHook

	if err := c.BindJSON(&hook); err != nil {
		//failed. malformed.
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
}
