package handler

import (
	"IM_Server/middleware"
	"IM_Server/service"
	"IM_Server/ws"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MessageHistory GET /message/history?peer_id=2&before_id=0&limit=20
// 历史消息即离线消息：所有消息都落库了，上线后拉历史就是补离线期间的
func MessageHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	peerID, _ := strconv.Atoi(c.Query("peer_id"))
	beforeID, _ := strconv.Atoi(c.DefaultQuery("before_id", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if peerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "peer_id required"})
		return
	}
	msgs, err := service.GetHistory(userID, uint(peerID), uint(beforeID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

// MarkRead POST /message/read  {"peer_id": 2}
// 把 peer 发给我的消息全部标记已读；若对方在线，通过 WebSocket 推已读回执
func MarkRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		PeerID uint `json:"peer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "peer_id required"})
		return
	}
	ids, err := service.MarkRead(userID, req.PeerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "mark read failed"})
		return
	}
	if len(ids) > 0 {
		ws.NotifyUser(req.PeerID, &ws.Message{
			Type:   ws.TypeRead,
			From:   userID,
			To:     req.PeerID,
			MsgIDs: ids,
		})
	}
	c.JSON(http.StatusOK, gin.H{"read_msg_ids": ids})
}

// GroupMessageHistory GET /message/group/history?group_id=1&before_id=0&limit=20
func GroupMessageHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	groupID, _ := strconv.Atoi(c.Query("group_id"))
	beforeID, _ := strconv.Atoi(c.DefaultQuery("before_id", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if groupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "group_id required"})
		return
	}
	msgs, err := service.GetGroupHistory(userID, uint(groupID), uint(beforeID), limit)
	if errors.Is(err, service.ErrNotGroupMember) {
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}
