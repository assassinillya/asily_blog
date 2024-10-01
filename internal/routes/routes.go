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
	//r.PUT("/blog/reSetBlog", handlers.ReSetBlog)

	protected := r.Group("/blog").Use(middleware.JWTAuthMiddleware(config.C.AccessSecret))
	{
		protected.POST("/insertBlog", handlers.InsertBlog) // 插入博客
		protected.PUT("/reSetBlog", handlers.ReSetBlog)    // 编辑博客
	}

	r.PUT("/blog/viewAdd", handlers.ViewAdd)              // 博客阅读量增加
	r.POST("/login", handlers.Login)                      // 登录
	r.GET("/tags", handlers.Tags)                         // 获取所有Tag
	r.GET("/blog/getBlog", handlers.GetBlog)              // 查看当前博客
	r.DELETE("/comments/delete", handlers.DeleteComments) // 删除评论
	r.POST("/comments/Add", handlers.AddComment)          // 添加评论
	r.POST("/comments/like", handlers.LikeComment)        // 点赞评论

}
