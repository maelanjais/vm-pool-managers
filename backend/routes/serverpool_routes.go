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
		serverpool.DELETE("/:id", middlewares.AuthMiddleware(), controllers.DeleteServerpool)
		serverpool.GET("mysp", middlewares.AuthMiddleware(), controllers.GetMyServerpools)
		serverpool.GET("mysp/:id", middlewares.AuthMiddleware(), controllers.GetServersInServerpool)
		serverpool.GET("images", middlewares.AuthMiddleware(), controllers.GetAllImages)
		serverpool.GET("flavor", middlewares.AuthMiddleware(), controllers.GetallFlavors)
		serverpool.GET("networks", middlewares.AuthMiddleware(), controllers.GetAllNetworks)
		serverpool.POST("rebuild", middlewares.AuthMiddleware(), controllers.RebuildServer)
		serverpool.POST("imagegroup", middlewares.AuthMiddleware(), controllers.GetGroupeImage)
		serverpool.GET("groupimagesname", middlewares.AuthMiddleware(), controllers.GetGroupeImagename)
	}
}
