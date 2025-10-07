package routes

import (
	"PoolManagerVM/backend/controllers"
	"PoolManagerVM/backend/middlewares"

	"github.com/gin-gonic/gin"
)

func WebSocketRoutes(r *gin.Engine) {
	r.GET("/ws", middlewares.AuthMiddleware(), controllers.HandleWebSocket)
}
