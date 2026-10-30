package dao

import (
	"IM_Server/model"
	"time"
)

func CreateMessage(msg *model.Message) error {
	return DB.Create(msg).Error
}

// GetMessagesBetween 拉取两人的聊天记录，游标翻页（id < beforeID），倒序返回最新的在前
func GetMessagesBetween(a, b, beforeID uint, limit int) ([]model.Message, error) {
	var msgs []model.Message
	db := DB.Where(
		"(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
		a, b, b, a,
	)
	if beforeID > 0 {
		db = db.Where("id < ?", beforeID)
	}
	err := db.Order("id DESC").Limit(limit).Find(&msgs).Error
	return msgs, err
}

// MarkMessagesRead 把 peer 发给 reader 的未读消息全部标记为已读，返回被标记的消息 ID
// （先查出 ID 再按主键更新：一条 UPDATE 的 WHERE 走 idx_pair 索引，且能拿到 ID 列表通知发送方）
func MarkMessagesRead(readerID, peerID uint) ([]uint, error) {
	var ids []uint
	err := DB.Model(&model.Message{}).
		Where("from_user_id = ? AND to_user_id = ? AND read_at IS NULL", peerID, readerID).
		Pluck("id", &ids).Error
	if err != nil || len(ids) == 0 {
		return ids, err
	}
	now := time.Now()
	err = DB.Model(&model.Message{}).
		Where("id IN ?", ids).
		Update("read_at", now).Error
	return ids, err
}

func CreateGroupMessage(msg *model.GroupMessage) error {
	return DB.Create(msg).Error
}

func GetGroupMessages(groupID, beforeID uint, limit int) ([]model.GroupMessage, error) {
	var msgs []model.GroupMessage
	db := DB.Where("group_id = ?", groupID)
	if beforeID > 0 {
		db = db.Where("id < ?", beforeID)
	}
	err := db.Order("id DESC").Limit(limit).Find(&msgs).Error
	return msgs, err
}
