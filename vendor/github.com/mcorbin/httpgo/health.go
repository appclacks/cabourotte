package http

import (
	"github.com/gin-gonic/gin"
)

func (c *Component) health(context *gin.Context) {
	context.JSON(200, NewResponse("ok"))
}
