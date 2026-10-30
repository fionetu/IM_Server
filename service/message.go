package service

import (
	"IM_Server/dao"
	"IM_Server/model"
	"errors"
)

var ErrNotFriend = errors.New("对方不是你的好友")

// SavePrivateMessage 单聊消息落库，返回消息 ID 和入库时间戳
func SavePrivateMessage(from, to uint, content string) (uint, int64, error) {
	if !IsFriend(from, to) {
		return 0, 0, ErrNotFriend
	}
	msg := &model.Message{
		FromUserID: from,
		ToUserID:   to,
		Content:    content,
	}
	if err := dao.CreateMessage(msg); err != nil {
		return 0, 0, err
	}
	return msg.ID, msg.CreatedAt.Unix(), nil
}

// SaveGroupMessage 群聊消息落库，并返回群成员 ID 列表（供 hub 扇出）
func SaveGroupMessage(groupID, from uint, content string) (uint, int64, []uint, error) {
	if !IsGroupMember(groupID, from) {
		return 0, 0, nil, ErrNotGroupMember
	}
	msg := &model.GroupMessage{
		GroupID:    groupID,
		FromUserID: from,
		Content:    content,
	}
	if err := dao.CreateGroupMessage(msg); err != nil {
		return 0, 0, nil, err
	}
	memberIDs, err := dao.GetGroupMemberIDs(groupID)
	if err != nil {
		return 0, 0, nil, err
	}
	return msg.ID, msg.CreatedAt.Unix(), memberIDs, nil
}

// MarkRead 把 peer 发给我的消息标记为已读，返回本次标记的消息 ID。
// 通知发送方（WebSocket 推送）由 handler 层做：service 不依赖 ws 包，避免循环引用
func MarkRead(readerID, peerID uint) ([]uint, error) {
	return dao.MarkMessagesRead(readerID, peerID)
}

// GetHistory 单聊历史消息（游标翻页：beforeID=0 表示从最新开始）
func GetHistory(userID, peerID, beforeID uint, limit int) ([]model.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return dao.GetMessagesBetween(userID, peerID, beforeID, limit)
}

// GetGroupHistory 群聊历史消息，要求调用者是群成员
func GetGroupHistory(userID, groupID, beforeID uint, limit int) ([]model.GroupMessage, error) {
	if !IsGroupMember(groupID, userID) {
		return nil, ErrNotGroupMember
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return dao.GetGroupMessages(groupID, beforeID, limit)
}
