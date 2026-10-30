package ws

import (
	"encoding/json"
	"time"

	"IM_Server/pkg"
	"IM_Server/service"

	"github.com/gorilla/websocket"
)

// 应用层消息类型
const (
	TypeChat  = "chat"  // 单聊
	TypeGroup = "group" // 群聊
	TypeAck   = "ack"   // 服务端回执（已收到并落库）
	TypeRead  = "read"  // 已读回执（对方读了你的消息）
	TypeError = "error" // 错误通知（如非好友、非群成员）
)

// 心跳参数
const (
	// pongWait 服务端等 pong 的最长时间，超时判定连接假死
	pongWait = 60 * time.Second
	// pingPeriod 服务端主动发 ping 的周期，必须小于 pongWait
	pingPeriod = 45 * time.Second
	// writeWait 单次写操作允许的最大耗时
	writeWait = 10 * time.Second
)

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID uint
}

// Message WebSocket 线路协议（和 model.Message 数据库表是两回事）
type Message struct {
	Type        string `json:"type"`                    // chat / group / ack / error
	MsgID       uint   `json:"msg_id,omitempty"`        // 落库后的消息 ID（服务端填）
	ClientMsgID string `json:"client_msg_id,omitempty"` // 客户端临时 ID，ACK 时原样带回用于对账
	From        uint   `json:"from"`
	To          uint   `json:"to"` // chat: 对方 userID；group: groupID
	Content     string `json:"content"`
	CreatedAt   int64  `json:"created_at,omitempty"`

	// MsgIDs 已读回执携带的被读消息 ID 列表（read 类型专用）
	MsgIDs []uint `json:"msg_ids,omitempty"`

	memberIDs []uint // 群聊扇出的成员列表（readPump 落库时查好，不进 JSON）
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close() // 全程序唯一关门点
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// hub 关闭了 send（注销或踢下线）：发关闭帧后退出
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			// 定期发 ping 探活；写不出去说明连接已死，退出
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	// 心跳：给每次读设截止时间，收到 pong 就续期。
	// 对端假死（断电/断网没发 FIN）时，ReadMessage 会在 pongWait 后超时返回 err
	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		service.RefreshOnline(c.userID) // 心跳顺便给 Redis 在线状态续期
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.hub.unregister <- c
			break
		}

		msg := &Message{}
		if err := json.Unmarshal(message, msg); err != nil {
			continue
		}
		msg.From = c.userID

		// 先落库再转发。落库放在各自的 readPump 里做：
		// 慢 DB 只卡当前这一个连接，不会堵住 hub 事件循环
		if !c.persist(msg) {
			continue
		}

		c.hub.forward <- msg
	}
}

// persist 按消息类型落库；失败时给发送者回 error 通知
func (c *Client) persist(msg *Message) bool {
	var err error
	switch msg.Type {
	case TypeGroup:
		var memberIDs []uint
		msg.MsgID, msg.CreatedAt, memberIDs, err = service.SaveGroupMessage(msg.To, c.userID, msg.Content)
		msg.memberIDs = memberIDs
	default:
		msg.Type = TypeChat
		msg.MsgID, msg.CreatedAt, err = service.SavePrivateMessage(c.userID, msg.To, msg.Content)
	}
	if err != nil {
		pkg.Log.Error("persist message: ", err)
		c.notifyError(err.Error())
		return false
	}
	return true
}

// notifyError 往自己的 send 信箱塞一条错误通知（由 writePump 写出）
func (c *Client) notifyError(content string) {
	data, err := json.Marshal(&Message{Type: TypeError, Content: content})
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default: // 自己的信箱都塞满了，说明连接已经很糟，直接丢弃
	}
}
