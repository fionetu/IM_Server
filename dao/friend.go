package dao

import (
	"IM_Server/model"

	"gorm.io/gorm"
)

func IsFriend(a, b uint) (bool, error) {
	var count int64
	err := DB.Model(&model.Friend{}).
		Where("user_id = ? AND friend_id = ?", a, b).
		Count(&count).Error
	return count > 0, err
}

func CreateFriendRequest(req *model.FriendRequest) error {
	return DB.Create(req).Error
}

func GetFriendRequestByID(id uint) (*model.FriendRequest, error) {
	var req model.FriendRequest
	if err := DB.First(&req, id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

// GetPendingRequestBetween 查两人之间任意方向的待处理申请
func GetPendingRequestBetween(a, b uint) (*model.FriendRequest, error) {
	var req model.FriendRequest
	err := DB.Where(
		"status = 0 AND ((from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?))",
		a, b, b, a,
	).First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// GetFriendIDs 只查好友 ID 列表（在线状态过滤等场景不需要联表）
func GetFriendIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := DB.Model(&model.Friend{}).
		Where("user_id = ?", userID).
		Pluck("friend_id", &ids).Error
	return ids, err
}

// ListFriends 联表查出好友的用户信息
func ListFriends(userID uint) ([]model.User, error) {
	var users []model.User
	err := DB.Table("users").
		Joins("JOIN friends ON friends.friend_id = users.id").
		Where("friends.user_id = ? AND friends.deleted_at IS NULL", userID).
		Find(&users).Error
	return users, err
}

// ListPendingRequests 发给当前用户、待处理的申请
func ListPendingRequests(userID uint) ([]model.FriendRequest, error) {
	var reqs []model.FriendRequest
	err := DB.Where("to_user_id = ? AND status = 0", userID).
		Order("id DESC").Find(&reqs).Error
	return reqs, err
}

// AcceptFriendRequest 事务：更新申请状态 + 插入双向好友关系，要么都成功要么都回滚
func AcceptFriendRequest(req *model.FriendRequest) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.FriendRequest{}).
			Where("id = ?", req.ID).
			Update("status", 1).Error; err != nil {
			return err
		}
		pairs := []model.Friend{
			{UserID: req.FromUserID, FriendID: req.ToUserID},
			{UserID: req.ToUserID, FriendID: req.FromUserID},
		}
		return tx.Create(&pairs).Error
	})
}

func RejectFriendRequest(id uint) error {
	return DB.Model(&model.FriendRequest{}).
		Where("id = ?", id).
		Update("status", 2).Error
}
