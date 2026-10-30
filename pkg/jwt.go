package pkg

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret []byte
var expireHours = 24

var ErrInvalidToken = errors.New("invalid token")

// InitJWT 用配置里的密钥初始化；不再从环境变量隐式读取，启动时显式注入
func InitJWT(secretStr string, hours int) {
	if secretStr == "" {
		panic("jwt secret is empty")
	}
	secret = []byte(secretStr)
	if hours > 0 {
		expireHours = hours
	}
}

type Claims struct {
	UserId uint `json:"userId"`
	jwt.RegisteredClaims
}

func GenerateToken(userId uint) (string, error) {
	exp := jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour))
	issued := jwt.NewNumericDate(time.Now())
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{userId, jwt.RegisteredClaims{IssuedAt: issued, Issuer: "im-server", ExpiresAt: exp}})
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
