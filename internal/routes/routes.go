package routes

import (
	"asily_blog/internal/handlers"
	"asily_blog/internal/middleware"
	"asily_blog/pkg/config"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.POST("/blog/test", handlers.Test)

	//r.POST("/blog/insertBlog", handlers.InsertBlog)
	//r.PUT("/blog/viewAdd", handlers.ViewAdd)
	//r.PUT("/blog/reSetBlog", handlers.ReSetBlog)

	protected := r.Group("/blog").Use(middleware.JWTAuthMiddleware(config.C.AccessSecret))
	{
		protected.POST("/insertBlog", handlers.InsertBlog)
		protected.PUT("/viewAdd", handlers.ViewAdd)
		protected.PUT("/reSetBlog", handlers.ReSetBlog)
	}

	r.POST("/login", handlers.Login)
	// 更多路由...
}
