package handler

import (
	"net/http"
	"strconv"

	"AIM/internal/service"

	"github.com/gin-gonic/gin"
)

type FriendHandler struct {
	Svc *service.FriendService
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
