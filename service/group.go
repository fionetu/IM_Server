package service

import (
	"IM_Server/dao"
	"IM_Server/model"
	"errors"
)

var ErrNotGroupMember = errors.New("你不是该群成员")

func IsGroupMember(groupID, userID uint) bool {
	ok, err := dao.IsGroupMember(groupID, userID)
	return err == nil && ok
}

// CreateGroup 建群，群主自动入群，成员去重
func CreateGroup(ownerID uint, name string, memberIDs []uint) (*model.Group, error) {
	seen := map[uint]struct{}{ownerID: {}}
	unique := []uint{ownerID}
	for _, id := range memberIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	group := &model.Group{Name: name, OwnerID: ownerID}
	if err := dao.CreateGroup(group, unique); err != nil {
		return nil, err
	}
	return group, nil
}

// AddGroupMembers 拉人入群，要求操作者本身是群成员
func AddGroupMembers(operatorID, groupID uint, memberIDs []uint) error {
	if !IsGroupMember(groupID, operatorID) {
		return ErrNotGroupMember
	}
	return dao.AddGroupMembers(groupID, memberIDs)
}

func ListGroups(userID uint) ([]model.Group, error) {
	return dao.ListGroupsByUser(userID)
}

func ListGroupMembers(userID, groupID uint) ([]model.User, error) {
	if !IsGroupMember(groupID, userID) {
		return nil, ErrNotGroupMember
	}
	return dao.ListGroupMembers(groupID)
}
