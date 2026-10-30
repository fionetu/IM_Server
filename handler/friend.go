package handler

import (
	"IM_Server/middleware"
	"IM_Server/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ApplyFriendReq struct {
	ToUserID uint `json:"to_user_id" binding:"required"`
}

// ApplyFriend POST /friend/apply {"to_user_id": 2}
func ApplyFriend(c *gin.Context) {
	var req ApplyFriendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	err := service.ApplyFriend(middleware.GetUserID(c), req.ToUserID)
	switch {
	case errors.Is(err, service.ErrCannotAddSelf),
		errors.Is(err, service.ErrAlreadyFriend),
		errors.Is(err, service.ErrRequestPending),
		errors.Is(err, service.ErrUserNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"message": "apply failed"})
	default:
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

type HandleFriendReq struct {
	RequestID uint   `json:"request_id" binding:"required"`
	Action    string `json:"action" binding:"required,oneof=accept reject"`
}

// HandleFriend POST /friend/handle {"request_id": 1, "action": "accept"}
func HandleFriend(c *gin.Context) {
	var req HandleFriendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	err := service.HandleFriendRequest(req.RequestID, middleware.GetUserID(c), req.Action == "accept")
	switch {
	case errors.Is(err, service.ErrRequestNotFound),
		errors.Is(err, service.ErrRequestHandled):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrNotRequestHandler):
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"message": "handle failed"})
	default:
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

// FriendList GET /friend/list
func FriendList(c *gin.Context) {
	users, err := service.ListFriends(middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"friends": users})
}

// FriendRequests GET /friend/requests（发给我、待处理的申请）
func FriendRequests(c *gin.Context) {
	reqs, err := service.ListFriendRequests(middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": reqs})
}

// OnlineFriends GET /friend/online（在线好友的 userID 列表）
func OnlineFriends(c *gin.Context) {
	ids, err := service.OnlineFriendIDs(middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"online": ids})
}
