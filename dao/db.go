package dao

import (
	"IM_Server/model"
	"IM_Server/pkg"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	db, err := gorm.Open(mysql.Open(pkg.Cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		pkg.Log.Fatal("connect mysql: ", err)
	}
	DB = db
	err = DB.AutoMigrate(
		&model.User{},
		&model.Message{},
		&model.GroupMessage{},
		&model.Friend{},
		&model.FriendRequest{},
		&model.Group{},
		&model.GroupMember{},
	)
	if err != nil {
		pkg.Log.Fatal("auto migrate: ", err)
	}
}
