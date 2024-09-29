package routes

import (
	"asily_blog/internal/handlers"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.POST("/blog/test", handlers.Test)
	r.POST("/blog/insertBlog", handlers.InsertBlog)
	r.PUT("/blog/viewAdd", handlers.ViewAdd)
	r.PUT("/blog/reSetBlog", handlers.ReSetBlog)
	// 更多路由...
}
