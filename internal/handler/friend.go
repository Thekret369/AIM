package handler

import (
	"net/http"
	"strconv"

	"LanLine/internal/service"
	"LanLine/internal/ws"

	"github.com/gin-gonic/gin"
)

type FriendHandler struct {
	Svc *service.FriendService
	Hub *ws.Hub
}

// AddFriend 发送好友申请
func (h *FriendHandler) AddFriend(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		FriendID uint   `json:"friend_id" binding:"required"`
		Message  string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	rel, err := h.Svc.AddFriend(userID, req.FriendID, req.Message)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"relation": rel})
}

// HandleRequest 同意或拒绝好友申请
func (h *FriendHandler) HandleRequest(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的申请 ID"})
		return
	}

	var req struct {
		Accept bool `json:"accept"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.Svc.HandleRequest(uint(id), userID, req.Accept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "操作成功"})
}

// DeleteFriend 删除好友
func (h *FriendHandler) DeleteFriend(c *gin.Context) {
	userID := c.GetUint("user_id")
	friendID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的好友 ID"})
		return
	}

	if err := h.Svc.DeleteFriend(userID, uint(friendID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已删除"})
}

// UpdateRemark 修改好友备注和分组
func (h *FriendHandler) UpdateRemark(c *gin.Context) {
	userID := c.GetUint("user_id")
	friendID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的好友 ID"})
		return
	}

	var req struct {
		Remark  string `json:"remark"`
		Note    string `json:"note"`
		GroupID *uint  `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.Svc.UpdateRemark(userID, uint(friendID), req.Remark, req.Note, req.GroupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已更新"})
}

// FriendList 获取好友列表
func (h *FriendHandler) FriendList(c *gin.Context) {
	userID := c.GetUint("user_id")

	list, err := h.Svc.FriendList(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"friends": list})
}

// GetOnlineFriends 返回所有在线好友的 ID 列表
func (h *FriendHandler) GetOnlineFriends(c *gin.Context) {
	userID := c.GetUint("user_id")

	friends, err := h.Svc.FriendList(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	onlineSet := make(map[uint]bool)
	for _, uid := range h.Hub.GetOnlineUserIDs() {
		onlineSet[uid] = true
	}

	type OnlineFriend struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		IsOnline bool   `json:"is_online"`
	}
	var result []OnlineFriend
	for _, f := range friends {
		result = append(result, OnlineFriend{
			ID:       f.FriendID,
			Username: f.Username,
			Nickname: f.Nickname,
			IsOnline: onlineSet[f.FriendID],
		})
	}
	c.JSON(http.StatusOK, gin.H{"friends": result})
}

// PendingRequests 获取待处理申请
func (h *FriendHandler) PendingRequests(c *gin.Context) {
	userID := c.GetUint("user_id")

	list, err := h.Svc.PendingRequests(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": list})
}
