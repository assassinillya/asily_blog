package routes

import (
	"asily_blog/internal/handlers"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.POST("/test", handlers.Test)
	//r.POST("/blogs", handlers.CreateBlog)
	// 更多路由...
}
