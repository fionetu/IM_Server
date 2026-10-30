package model

import "gorm.io/gorm"

// Group 群
type Group struct {
	gorm.Model
	Name    string `gorm:"size:64;not null" json:"name"`
	OwnerID uint   `gorm:"not null" json:"owner_id"`
}

// GroupMember 群成员（group_id + user_id 联合唯一，防重复入群）
type GroupMember struct {
	gorm.Model
	GroupID uint `gorm:"not null;uniqueIndex:idx_group_user" json:"group_id"`
	UserID  uint `gorm:"not null;uniqueIndex:idx_group_user" json:"user_id"`
}
