package model

import "gorm.io/gorm"

// Friend 好友关系。互为好友存两条记录（A→B、B→A），
// 用存储换查询简单：查"我的好友"只需单条件 WHERE user_id = me
type Friend struct {
	gorm.Model
	UserID   uint `gorm:"not null;uniqueIndex:idx_user_friend" json:"user_id"`
	FriendID uint `gorm:"not null;uniqueIndex:idx_user_friend" json:"friend_id"`
}

// FriendRequest 好友申请
type FriendRequest struct {
	gorm.Model
	FromUserID uint `gorm:"not null;index" json:"from_user_id"`
	ToUserID   uint `gorm:"not null;index" json:"to_user_id"`
	Status     int  `gorm:"not null;default:0" json:"status"` // 0待处理 1已同意 2已拒绝
}
