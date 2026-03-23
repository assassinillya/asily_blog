package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JwtPayLoad jwt中payload数据
type JwtPayLoad struct {
	// 安全性修复：已从此结构体中移除 PassWord 字段。
	// JWT 的载荷是公开可见的（只是被 Base64 编码），绝不能在其中存储敏感信息如密码。
	Username string `json:"username"`
}

// CustomClaims 是我们自定义的 JWT 声明，它包含了我们的载荷和标准的注册声明
type CustomClaims struct {
	JwtPayLoad
	jwt.RegisteredClaims
}

// GenToken 创建一个新的 JWT
// payload: 需要包含在 token 中的用户数据
// accessSecret: 用于签发 token 的秘钥
// expires: token 的有效期，单位为小时
func GenToken(payload JwtPayLoad, accessSecret string, expires int) (string, error) {
	// 创建声明
	claims := CustomClaims{
		JwtPayLoad: payload,
		RegisteredClaims: jwt.RegisteredClaims{
			// 设置过期时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expires))),
			// 设置签发时间
			IssuedAt: jwt.NewNumericDate(time.Now()),
			// 设置主题
			Subject: "user-token",
		},
	}

	// 使用 HS256 签名算法创建一个新的 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用提供的秘钥对 token 进行签名，并获取完整的 token 字符串
	return token.SignedString([]byte(accessSecret))
}

// ParseToken 解析并验证一个 JWT 字符串
func ParseToken(tokenStr string, accessSecret string) (*CustomClaims, error) {
	// 解析 token，同时会验证签名和过期时间
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 确保 token 的签名算法是我们期望的
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("非预期的签名算法")
		}
		return []byte(accessSecret), nil
	})

	if err != nil {
		// 如果错误是由于 token 过期或格式无效，jwt 库会返回特定的错误类型
		return nil, err
	}

	// 检查解析出的 claims 是否是我们定义的类型，并且 token 是否有效
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的Token")
}
