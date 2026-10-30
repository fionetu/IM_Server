package ws

import (
	"IM_Server/pkg"
	"IM_Server/service"
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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		pkg.Log.Error("ws upgrade: ", err)
		return
	}

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    hub,
		userID: claims.UserId,
	}
	hub.register <- client
	service.SetOnline(claims.UserId) // 写 Redis 在线状态（带 TTL，心跳续期）

	go client.writePump()
	go client.readPump()

}
