package ws

import (
	"IM_Server/pkg"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func Connect(c *gin.Context) {

	token := c.Query("token")
	claims, err := pkg.ParseToken(token)
	if err != nil {
		c.JSON(401, gin.H{"message": err.Error()})
		return
	}
	fmt.Println("鉴权")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("升级成功")

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    hub,
		userID: claims.UserId,
	}
	hub.register <- client

	go client.writePump()
	go client.readPump()

}
