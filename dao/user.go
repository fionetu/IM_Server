package dao

import "IM_Server/model"

func GetUserByName(username string) (*model.User, error) {
	var user model.User
	result := DB.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}
func GetUserByID(id uint) (*model.User, error) {
	var user model.User
	if err := DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func CreateUser(user *model.User) error {
	result := DB.Create(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
