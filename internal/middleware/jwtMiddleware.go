package middleware

import (
	"asily_blog/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

// JWTAuthMiddleware 鉴权中间件
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中获取 Token
		tokenString := c.GetHeader("Authorization")

		// 检查Token是否为空
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未携带Token"})
			c.Abort()
			return
		}

		// 解析 Token
		//claims, err := utils.ParseToken(tokenString, secret)
		_, err := utils.ParseToken(tokenString, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token无效或已过期"})
			c.Abort()
			return
		}

		// 将解析后的用户信息保存在上下文中，以便后续处理使用
		//c.Set("userID", claims.UserID)

		// 继续处理
		c.Next()
	}
}
