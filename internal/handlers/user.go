package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Test(c *gin.Context) {
	// 用户注册逻辑
	c.JSON(http.StatusOK, gin.H{"message": "test!test!"})
}
