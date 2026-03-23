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
	// 定义用于绑定请求 JSON 的结构体
	var requestUser struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// 绑定并验证请求数据
	if err := c.ShouldBindJSON(&requestUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	// 从配置文件加载正确的用户名和密码哈希
	expectedUsername := config.C.User
	hashedPasswordFromConfig := config.C.Password // 现在这里应该是哈希值

	// 1. 验证用户名
	// 注意：这里的用户名比较仍然是直接比较。如果需要防止用户名枚举，可以采用与密码比较类似的恒定时间比较法，
	// 但通常情况下，优先保护密码是更重要的。
	if requestUser.Username != expectedUsername {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号或密码错误"})
		return
	}

	// 2. 验证密码
	// 使用 bcrypt.CompareHashAndPassword 来安全地比较用户输入的密码和存储在配置文件中的哈希。
	// 这个函数本身就能抵抗时序攻击。
	err := bcrypt.CompareHashAndPassword([]byte(hashedPasswordFromConfig), []byte(requestUser.Password))
	if err != nil {
		// 如果 err 不为 nil，说明密码不匹配。
		// err 可能是 bcrypt.ErrMismatchedHashAndPassword，也可能是哈希格式错误。
		// 为安全起见，我们统一返回相同的错误信息。
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号或密码错误"})
		return
	}

	// --- 密码验证通过 ---

	// 改进建议：Token 的过期时间（如此处的 168 小时）应该配置在 config.json 中，而不是硬编码。
	const tokenExpireHours = 168 // 7 days

	// 创建 Token
	token, err := utils.GenToken(utils.JwtPayLoad{
		Username: requestUser.Username,
	}, config.C.AccessSecret, tokenExpireHours)

	if err != nil {
		// 如果 Token 生成失败，这是一个服务器内部错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成Token失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
