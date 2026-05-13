package handler

import (
	"net/http"
	"strconv"

	"AIM/internal/service"

	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	Svc *service.GroupService
}

// CreateGroup 创建群组
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	group, err := h.Svc.CreateGroup(req.Name, req.Description, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"group": group})
}

// JoinGroup 加入群组
func (h *GroupHandler) JoinGroup(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}

	if err := h.Svc.JoinGroup(uint(groupID), userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已加入"})
}

// LeaveGroup 退出群组
func (h *GroupHandler) LeaveGroup(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}

	if err := h.Svc.LeaveGroup(uint(groupID), userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已退出"})
}

// KickMember 踢出成员
func (h *GroupHandler) KickMember(c *gin.Context) {
	operatorID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}

	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.Svc.KickMember(uint(groupID), operatorID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已踢出"})
}

// TransferOwner 转让群主
func (h *GroupHandler) TransferOwner(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}

	var req struct {
		NewOwnerID uint `json:"new_owner_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.Svc.TransferOwner(uint(groupID), userID, req.NewOwnerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已转让"})
}

// MuteMember 禁言成员
func (h *GroupHandler) MuteMember(c *gin.Context) {
	operatorID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}

	var req struct {
		UserID  uint `json:"user_id" binding:"required"`
		Minutes int  `json:"minutes" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.Svc.MuteMember(uint(groupID), operatorID, req.UserID, req.Minutes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已禁言"})
}

// GetGroupMembers 获取群成员列表
func (h *GroupHandler) GetGroupMembers(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}

	members, err := h.Svc.GetGroupMembers(uint(groupID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

// GetGroupDetail 获取群组详情 + 当前用户的成员角色
func (h *GroupHandler) GetGroupDetail(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	group, member, err := h.Svc.GetGroupDetail(uint(groupID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	myRole := ""
	isMuted := false
	dnd := false
	if member != nil {
		myRole = string(member.Role)
		isMuted = member.IsMuted()
		dnd = member.DND
	}
	c.JSON(http.StatusOK, gin.H{
		"group":    group,
		"my_role":  myRole,
		"is_muted": isMuted,
		"dnd":      dnd,
	})
}

// UpdateGroup 更新群资料（名称、头像、简介、公告）
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		Avatar      string `json:"avatar"`
		Description string `json:"description"`
		Announce    string `json:"announce"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := h.Svc.UpdateGroup(uint(groupID), userID, req.Name, req.Avatar, req.Description, req.Announce); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已更新"})
}

// SetAdmin 切换管理员身份
func (h *GroupHandler) SetAdmin(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	targetID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户 ID"})
		return
	}
	if err := h.Svc.SetAdmin(uint(groupID), userID, uint(targetID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "操作成功"})
}

// UnmuteMember 解除禁言
func (h *GroupHandler) UnmuteMember(c *gin.Context) {
	operatorID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := h.Svc.UnmuteMember(uint(groupID), operatorID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已解除禁言"})
}

// AddMember 添加成员到群组
func (h *GroupHandler) AddMember(c *gin.Context) {
	operatorID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := h.Svc.AddMember(uint(groupID), operatorID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "已添加"})
}

// GetUserGroups 获取用户所在的群组列表
func (h *GroupHandler) GetUserGroups(c *gin.Context) {
	userID := c.GetUint("user_id")

	groups, err := h.Svc.GetUserGroups(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// ToggleDND 切换群消息免打扰
func (h *GroupHandler) ToggleDND(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	dnd, err := h.Svc.ToggleDND(uint(groupID), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dnd": dnd})
}

// CreateAnnouncement 发布群公告（保留历史记录）
func (h *GroupHandler) CreateAnnouncement(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	ann, err := h.Svc.CreateAnnouncement(uint(groupID), userID, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"announcement": ann})
}

// GetAnnouncements 获取群公告列表（含历史）
func (h *GroupHandler) GetAnnouncements(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	anns, err := h.Svc.GetAnnouncements(uint(groupID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"announcements": anns})
}
