package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null;size:32;" json:"username"`
	Password string `gorm:"not null;size:60;" json:"-"`
	Nickname string `gorm:"size:32;" json:"nickname"`
	Age      int    `gorm:"default:0;" json:"age"`
	Sex      int    `gorm:"default:0;" json:"sex"`
	Avatar   string `gorm:"size:100;" json:"avatar"`
}
