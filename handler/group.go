package handler

import (
	"IM_Server/middleware"
	"IM_Server/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CreateGroupReq struct {
	Name      string `json:"name" binding:"required"`
	MemberIDs []uint `json:"member_ids"`
}

// CreateGroup POST /group/create {"name": "周末球局", "member_ids": [2, 3]}
func CreateGroup(c *gin.Context) {
	var req CreateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	group, err := service.CreateGroup(middleware.GetUserID(c), req.Name, req.MemberIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "create failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"group": group})
}

type AddMembersReq struct {
	GroupID   uint   `json:"group_id" binding:"required"`
	MemberIDs []uint `json:"member_ids" binding:"required"`
}

// AddGroupMembers POST /group/add {"group_id": 1, "member_ids": [4]}
func AddGroupMembers(c *gin.Context) {
	var req AddMembersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	err := service.AddGroupMembers(middleware.GetUserID(c), req.GroupID, req.MemberIDs)
	if errors.Is(err, service.ErrNotGroupMember) {
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "add failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// GroupList GET /group/list
func GroupList(c *gin.Context) {
	groups, err := service.ListGroups(middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// GroupMembers GET /group/members?group_id=1
func GroupMembers(c *gin.Context) {
	groupID, _ := strconv.Atoi(c.Query("group_id"))
	if groupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "group_id required"})
		return
	}
	users, err := service.ListGroupMembers(middleware.GetUserID(c), uint(groupID))
	if errors.Is(err, service.ErrNotGroupMember) {
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": users})
}
