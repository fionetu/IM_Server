package ws

import (
	"encoding/json"

	"IM_Server/pkg"
	"IM_Server/service"
)

var hub = newHub()

// notifyPayload HTTP 侧（如已读回执）要推给某个用户的消息，经 notify 通道进事件循环
type notifyPayload struct {
	userID uint
	data   []byte
}

type Hub struct {
	clients    map[uint]*Client
	register   chan *Client
	unregister chan *Client
	forward    chan *Message
	notify     chan *notifyPayload
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[uint]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		forward:    make(chan *Message),
		notify:     make(chan *notifyPayload, 256),
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
				// 异步删 Redis 在线状态：别让网络 IO 堵事件循环
				go service.SetOffline(client.userID)
			}
		case message := <-h.forward:
			h.route(message)
		case p := <-h.notify:
			h.deliver(p.userID, p.data)
		}
	}
}

// route 把一条已落库的消息投递出去，并给发送者回 ACK
func (h *Hub) route(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		pkg.Log.Error("marshal message: ", err)
		return
	}

	if message.Type == TypeGroup {
		// 群聊写扩散：给除发送者外的每个在线成员各投一份
		for _, id := range message.memberIDs {
			if id == message.From {
				continue
			}
			h.deliver(id, data)
		}
	} else {
		h.deliver(message.To, data)
	}

	// ACK：告诉发送者"服务端已收到并落库"，客户端凭 ClientMsgID 对账
	ack := &Message{
		Type:        TypeAck,
		MsgID:       message.MsgID,
		ClientMsgID: message.ClientMsgID,
	}
	ackData, err := json.Marshal(ack)
	if err != nil {
		pkg.Log.Error("marshal ack: ", err)
		return
	}
	h.deliver(message.From, ackData)
}

// deliver 投递给单个在线用户。
// send 缓冲塞满说明客户端读取太慢（网络差/假死），直接踢下线，
// 否则 run() 会堵在这一个客户端上，全系统转发瘫痪
func (h *Hub) deliver(userID uint, data []byte) {
	target, ok := h.clients[userID]
	if !ok {
		pkg.Log.Debugw("user offline, message stays in DB", "user_id", userID)
		return
	}
	select {
	case target.send <- data:
	default:
		delete(h.clients, userID)
		close(target.send)
	}
}

func StartHub() {
	go hub.run()
}

// NotifyUser 供 HTTP 层调用：把一条应用层消息推给指定在线用户（不在线则丢弃）。
// 必须经 notify 通道进事件循环，不能从外部直接碰 clients map / send 通道
func NotifyUser(userID uint, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		pkg.Log.Error("marshal notify: ", err)
		return
	}
	hub.notify <- &notifyPayload{userID: userID, data: data}
}
