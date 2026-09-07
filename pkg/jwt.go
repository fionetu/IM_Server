package pkg

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte(os.Getenv("SECRET"))

func init() {
	if len(secret) == 0 {
		log.Fatal("SECRET environment variable not set")
	}
}

type Claims struct {
	UserId uint `json:"userId"`
	jwt.RegisteredClaims
}

func GenerateToken(userId uint) (string, error) {
	exp := jwt.NewNumericDate(time.Now().Add(24 * time.Hour))
	issued := jwt.NewNumericDate(time.Now())
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{userId, jwt.RegisteredClaims{IssuedAt: issued, Issuer: "im-server", ExpiresAt: exp}})
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
