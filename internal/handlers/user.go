package handlers

import (
	"asily_blog/internal/utils"
	"asily_blog/pkg/config"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Login(c *gin.Context) {
	// 没有设置Logout接口, 因为只有一个用户
	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "转码失败 " + err.Error()})
		return
	}

	if config.C.Password == user.Password && config.C.User == user.Username {
		token, err := utils.GenToken(utils.JwtPayLoad{
			Username: user.Username,
			PassWord: user.Password,
		}, config.C.AccessSecret, 168)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "生成Token失败 " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"Token": token})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "账号或密码错误"})
}
