package ws

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID uint
}

type Message struct {
	From    uint   `json:"from"`
	To      uint   `json:"to"`
	Content string `json:"content"`
}

func (c *Client) writePump() {

	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.hub.unregister <- c
			break
		}
		msg := &Message{}
		err = json.Unmarshal(message, msg)
		if err != nil {
			continue
		}
		msg.From = c.userID
		fmt.Println(msg.Content, msg.To)
		c.hub.forward <- msg
	}
}
