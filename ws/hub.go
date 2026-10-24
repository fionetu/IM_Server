package ws

import (
	"encoding/json"
	"fmt"
)

var hub = newHub()

type Hub struct {
	clients    map[uint]*Client
	register   chan *Client
	unregister chan *Client
	forward    chan *Message
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[uint]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		forward:    make(chan *Message),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.userID] = client
		case client := <-h.unregister:
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
			}
		case message := <-h.forward:
			target, ok := h.clients[message.To]
			if !ok {
				fmt.Println(message.To, message.From)
			} else {
				mess, err := json.Marshal(message)
				if err != nil {
					fmt.Println(err)
					continue
				}
				target.send <- mess
			}
		}

	}
}

func StartHub() {
	go hub.run()
}
