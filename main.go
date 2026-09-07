package main

import (
	"IM_Server/dao"
	"IM_Server/handler"

	"github.com/gin-gonic/gin"
)

func main() {

	dao.InitDB()

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	r.POST("/user/register", handler.Register)
	r.POST("/user/login", handler.Login)
	r.Run(":8080")
}
