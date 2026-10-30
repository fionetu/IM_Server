package middleware

import (
	"IM_Server/pkg"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuth HTTP 接口鉴权：从 Authorization: Bearer <token> 解析出 userID 塞进 context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		tokenString, found := strings.CutPrefix(auth, "Bearer ")
		if !found || tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing or malformed authorization header"})
			return
		}
		claims, err := pkg.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
			return
		}
		c.Set("userID", claims.UserId)
		c.Next()
	}
}

// GetUserID 从 context 取出 JWTAuth 塞进去的 userID
func GetUserID(c *gin.Context) uint {
	v, _ := c.Get("userID")
	id, _ := v.(uint)
	return id
}
