package service

import (
	"IM_Server/dao"
	"IM_Server/model"
	"errors"
)

var (
	ErrCannotAddSelf     = errors.New("不能添加自己为好友")
	ErrAlreadyFriend     = errors.New("你们已经是好友了")
	ErrRequestPending    = errors.New("已有待处理的申请")
	ErrRequestNotFound   = errors.New("申请不存在")
	ErrNotRequestHandler = errors.New("无权处理该申请")
	ErrRequestHandled    = errors.New("申请已处理")
)

// IsFriend 给 ws 层用的布尔版（错误时保守返回 false）
func IsFriend(a, b uint) bool {
	ok, err := dao.IsFriend(a, b)
	return err == nil && ok
}

func ApplyFriend(fromID, toID uint) error {
	if fromID == toID {
		return ErrCannotAddSelf
	}
	if _, err := dao.GetUserByID(toID); err != nil {
		return ErrUserNotFound
	}
	if IsFriend(fromID, toID) {
		return ErrAlreadyFriend
	}
	// 任意方向已有待处理申请就不重复创建
	if _, err := dao.GetPendingRequestBetween(fromID, toID); err == nil {
		return ErrRequestPending
	}
	return dao.CreateFriendRequest(&model.FriendRequest{
		FromUserID: fromID,
		ToUserID:   toID,
	})
}

func HandleFriendRequest(requestID, userID uint, accept bool) error {
	req, err := dao.GetFriendRequestByID(requestID)
	if err != nil {
		return ErrRequestNotFound
	}
	if req.ToUserID != userID {
		return ErrNotRequestHandler
	}
	if req.Status != 0 {
		return ErrRequestHandled
	}
	if accept {
		return dao.AcceptFriendRequest(req)
	}
	return dao.RejectFriendRequest(requestID)
}

func ListFriends(userID uint) ([]model.User, error) {
	return dao.ListFriends(userID)
}

func ListFriendRequests(userID uint) ([]model.FriendRequest, error) {
	return dao.ListPendingRequests(userID)
}
