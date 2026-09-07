package service

import (
	"IM_Server/dao"
	"IM_Server/model"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrUserExist = errors.New("user already exists")
var ErrInvalidCredentials = errors.New("用户名或密码错误")

func Register(user *model.User) error {

	_, err := dao.GetUserByName(user.Username)
	if err == nil {
		return ErrUserExist
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	pwd, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(pwd)

	return dao.CreateUser(user)

}

func Login(username, password string) (user *model.User, err error) {
	user, err = dao.GetUserByName(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	return
}
