package service

import (
	"IM_Server/dao"
	"strconv"
	"time"
)

// 在线状态存 Redis：key 带 TTL，心跳（pong）续期。
// 崩溃/假死等没走到注销逻辑的场景，key 到期自动过期，不会残留"幽灵在线"。
const onlineTTL = 75 * time.Second // 必须大于 ws 的 pongWait(60s)

func onlineKey(userID uint) string {
	return "online:" + strconv.FormatUint(uint64(userID), 10)
}

// SetOnline 连接建立时调用
func SetOnline(userID uint) {
	dao.RDB.Set(dao.Ctx, onlineKey(userID), 1, onlineTTL)
}

// RefreshOnline 每次收到 pong 续期 TTL
func RefreshOnline(userID uint) {
	dao.RDB.Expire(dao.Ctx, onlineKey(userID), onlineTTL)
}

// SetOffline 正常断开时主动删 key，立刻反映离线
func SetOffline(userID uint) {
	dao.RDB.Del(dao.Ctx, onlineKey(userID))
}

// IsOnline 判断单个用户是否在线
func IsOnline(userID uint) bool {
	n, err := dao.RDB.Exists(dao.Ctx, onlineKey(userID)).Result()
	return err == nil && n > 0
}

// OnlineFriendIDs 返回好友中在线的用户 ID
func OnlineFriendIDs(userID uint) ([]uint, error) {
	friendIDs, err := dao.GetFriendIDs(userID)
	if err != nil {
		return nil, err
	}
	online := make([]uint, 0, len(friendIDs))
	for _, id := range friendIDs {
		if IsOnline(id) {
			online = append(online, id)
		}
	}
	return online, nil
}
