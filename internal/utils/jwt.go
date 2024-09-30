package utils

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

// JwtPayLoad jwt中payload数据
type JwtPayLoad struct {
	Username string `json:"username"`
	PassWord string `json:"passWord"`
}

type CustomClaims struct {
	JwtPayLoad
	jwt.RegisteredClaims
}

// GenToken 创建 Token   payload 秘钥 过期时间
func GenToken(payload JwtPayLoad, accessSecret string, expires int) (string, error) {
	claim := CustomClaims{
		JwtPayLoad: payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expires))),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	return token.SignedString([]byte(accessSecret))
}

// ParseToken 解析 token
func ParseToken(tokenStr string, accessSecret string) (*CustomClaims, error) {

	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(accessSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("无效的Token")
}
