package main

import (
	"IM_Server/dao"

	"github.com/gin-gonic/gin"
)

func main() {

	dao.InitDB()

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	r.Run(":8080")
}
