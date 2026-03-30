package routes

import (
	"asily_blog/internal/handlers"
	"asily_blog/internal/middleware"
	"asily_blog/pkg/config"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error":   "请求的接口不存在",
			"message": "请检查请求路径，或由前端跳转到自定义 404 页面",
			"path":    c.Request.URL.Path,
		})
	})

	// --- 登录 ---
	// POST /login - 用户登录获取 Token
	r.POST("/login", handlers.Login)

	// --- 公共可访问的 API (无需认证) ---
	public := r.Group("/")
	{
		// 博客相关
		// GET /blogs?page=1&limit=10 - 分页获取博客列表
		public.GET("/blogs", handlers.GetBlogs)
		// GET /blogs/count - 获取博客总数
		public.GET("/blogs/count", handlers.GetBlogCount)
		// GET /blogs/search?q=keyword - 搜索博客
		public.GET("/blogs/search", handlers.SearchBlogs)
		// GET /blogs/{id} - 获取单篇博客详情
		public.GET("/blogs/:blogId", handlers.GetBlog)
		// PUT /blogs/like - 点赞博客
		public.PUT("/blogs/like", handlers.LikeBlog)
		// PUT /blogs/unlike - 取消点赞博客
		public.PUT("/blogs/unlike", handlers.UnLikeBlog)

		// 标签相关
		// GET /tags - 获取所有标签
		public.GET("/tags", handlers.GetTags) // 修正：之前为 handlers.Tags，已重命名为 GetTags

		// 评论相关
		// GET /comments/{blogId}?page=1&limit=10 - 获取某篇博客下的评论
		public.GET("/comments/:blogId", handlers.GetComment)
		// GET /comments/count/{blogId} - 获取某篇博客的评论总数
		public.GET("/comments/count/:blogId", handlers.GetCommentCount)
		// POST /comments - 添加评论 (通常评论是开放的)
		public.POST("/comments", handlers.AddComment)
		// PUT /comments/like - 点赞评论
		public.PUT("/comments/like", handlers.LikeComment)
		// PUT /comments/unlike - 取消点赞评论
		public.PUT("/comments/unlike", handlers.UnLikeComment)

		// 友链相关
		// GET /friend-links?page=1&limit=10 - 获取友情链接列表
		public.GET("/friend-links", handlers.GetLinks)
	}

	// --- 需要管理员认证的 API ---
	admin := r.Group("/").Use(middleware.JWTAuthMiddleware(config.C.AccessSecret))
	{
		// 博客管理
		// POST /blogs - 创建新博客
		admin.POST("/blogs", handlers.InsertBlog)
		// PUT /blogs - 编辑博客
		admin.PUT("/blogs", handlers.ReSetBlog)
		// DELETE /blogs - 删除博客
		admin.DELETE("/blogs", handlers.DeleteBlog)

		// 评论管理
		// DELETE /comments - 删除评论 (管理员权限)
		admin.DELETE("/comments", handlers.DeleteComments)
		// PUT /comments - 编辑评论 (管理员权限)
		admin.PUT("/comments", handlers.ResetComments)

		// 友链管理
		// POST /friend-links - 添加友情链接
		admin.POST("/friend-links", handlers.AddLink)
		// PUT /friend-links - 更新友情链接
		admin.PUT("/friend-links", handlers.UpdateLink)
		// DELETE /friend-links - 删除友情链接
		admin.DELETE("/friend-links", handlers.DeleteLink)
	}

	// --- 废弃或未使用的路由 (保留以供参考，但建议删除) ---
	// r.PUT("/blog/viewAdd", handlers.ViewAdd) // 已被 GetBlog 内部逻辑替代
	// r.POST("/blog/test", handlers.Test) // 测试路由，生产环境应移除
}
