package routes

import (
	"PoolManagerVM/backend/controllers"
	"PoolManagerVM/backend/middlewares"

	"github.com/gin-gonic/gin"
)

func ServerpoolRoutes(r *gin.Engine) {
	serverpool := r.Group("/serverpool")
	{
		serverpool.GET("", controllers.GetServerpool)
		serverpool.POST("", middlewares.AuthMiddleware(), controllers.CreateServerpool)
		serverpool.DELETE("", middlewares.AuthMiddleware(), controllers.DeleteServerpool)
		serverpool.GET("mysp", middlewares.AuthMiddleware(), controllers.GetMyServerpools)
	}
}
