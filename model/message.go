package model

import (
	"time"

	"gorm.io/gorm"
)

// Message 单聊消息表
type Message struct {
	gorm.Model
	FromUserID uint   `gorm:"not null;index:idx_pair,priority:1" json:"from_user_id"`
	ToUserID   uint   `gorm:"not null;index:idx_pair,priority:2" json:"to_user_id"`
	Content    string `gorm:"type:text;not null" json:"content"`
	// ReadAt NULL 表示未读；用时间而不是 bool，顺便记录了"什么时候读的"
	ReadAt *time.Time `json:"read_at"`
}

// GroupMessage 群聊消息表
type GroupMessage struct {
	gorm.Model
	GroupID    uint   `gorm:"not null;index" json:"group_id"`
	FromUserID uint   `gorm:"not null" json:"from_user_id"`
	Content    string `gorm:"type:text;not null" json:"content"`
}
