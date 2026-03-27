package handlers

import (
	"asily_blog/internal/utils"
	"asily_blog/pkg/config"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Login 处理用户登录请求
func Login(c *gin.Context) {
	var requestUser struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&requestUser); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求格式")
		return
	}

	expectedUsername := config.C.User
	hashedPasswordFromConfig := config.C.Password

	if requestUser.Username != expectedUsername {
		respondError(c, http.StatusUnauthorized, "账号或密码错误")
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPasswordFromConfig), []byte(requestUser.Password))
	if err != nil {
		respondError(c, http.StatusUnauthorized, "账号或密码错误")
		return
	}

	const tokenExpireHours = 168

	token, err := utils.GenToken(utils.JwtPayLoad{
		Username: requestUser.Username,
	}, config.C.AccessSecret, tokenExpireHours)

	if err != nil {
		respondError(c, http.StatusInternalServerError, "生成Token失败")
		return
	}

	respondData(c, http.StatusOK, gin.H{"token": token})
}
