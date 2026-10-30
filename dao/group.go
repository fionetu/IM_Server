package dao

import (
	"IM_Server/model"

	"gorm.io/gorm"
)

// CreateGroup 事务：建群 + 批量加成员
func CreateGroup(group *model.Group, memberIDs []uint) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}
		members := make([]model.GroupMember, 0, len(memberIDs))
		for _, id := range memberIDs {
			members = append(members, model.GroupMember{GroupID: group.ID, UserID: id})
		}
		return tx.Create(&members).Error
	})
}

// AddGroupMembers 批量加人；重复加已有成员会撞联合唯一索引报错
func AddGroupMembers(groupID uint, memberIDs []uint) error {
	members := make([]model.GroupMember, 0, len(memberIDs))
	for _, id := range memberIDs {
		members = append(members, model.GroupMember{GroupID: groupID, UserID: id})
	}
	return DB.Create(&members).Error
}

func IsGroupMember(groupID, userID uint) (bool, error) {
	var count int64
	err := DB.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetGroupMemberIDs 只取 user_id 一列，Pluck 扫进 []uint
func GetGroupMemberIDs(groupID uint) ([]uint, error) {
	var ids []uint
	err := DB.Model(&model.GroupMember{}).
		Where("group_id = ?", groupID).
		Pluck("user_id", &ids).Error
	return ids, err
}

// ListGroupsByUser 联表查"我加入的群"
func ListGroupsByUser(userID uint) ([]model.Group, error) {
	var groups []model.Group
	err := DB.Table("groups").
		Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ? AND group_members.deleted_at IS NULL", userID).
		Find(&groups).Error
	return groups, err
}

// ListGroupMembers 联表查群成员的用户信息
func ListGroupMembers(groupID uint) ([]model.User, error) {
	var users []model.User
	err := DB.Table("users").
		Joins("JOIN group_members ON group_members.user_id = users.id").
		Where("group_members.group_id = ? AND group_members.deleted_at IS NULL", groupID).
		Find(&users).Error
	return users, err
}
