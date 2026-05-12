// Package service 实现群组管理（创建、加入、退出、踢人、转让、禁言）
package service

import (
	"errors"
	"time"

	"AIM/internal/model"
)

// MemberInfo 群成员信息（含用户资料）
type MemberInfo struct {
	ID         uint        `json:"id"`
	GroupID    uint        `json:"group_id"`
	UserID     uint        `json:"user_id"`
	Username   string      `json:"username"`
	Nickname   string      `json:"nickname"`
	Role       string      `json:"role"`
	MutedUntil *time.Time  `json:"muted_until,omitempty"`
}

type GroupService struct{}

// CreateGroup 创建群组，创建者默认为群主
func (s *GroupService) CreateGroup(name string, ownerID uint) (*model.Group, error) {
	group := &model.Group{
		Name:    name,
		OwnerID: ownerID,
	}
	if err := model.DB.Create(group).Error; err != nil {
		return nil, err
	}

	member := &model.GroupMember{
		GroupID: group.ID,
		UserID:  ownerID,
		Role:    model.RoleOwner,
	}
	if err := model.DB.Create(member).Error; err != nil {
		return nil, err
	}

	return group, nil
}

// JoinGroup 加入群组
func (s *GroupService) JoinGroup(groupID, userID uint) error {
	// 检查群组是否存在
	var group model.Group
	if err := model.DB.First(&group, groupID).Error; err != nil {
		return errors.New("群组不存在")
	}

	// 检查是否已是成员
	var count int64
	model.DB.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count)
	if count > 0 {
		return errors.New("已是群成员")
	}

	member := &model.GroupMember{
		GroupID: groupID,
		UserID:  userID,
		Role:    model.RoleMember,
	}
	return model.DB.Create(member).Error
}

// LeaveGroup 退出群组。群主不可直接退出，需先转让
func (s *GroupService) LeaveGroup(groupID, userID uint) error {
	var member model.GroupMember
	if err := model.DB.Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error; err != nil {
		return errors.New("不是群成员")
	}
	if member.Role == model.RoleOwner {
		return errors.New("群主不可直接退群，请先转让群主")
	}
	return model.DB.Delete(&member).Error
}

// KickMember 踢出成员，仅群主和管理员可操作
func (s *GroupService) KickMember(groupID, operatorID, targetID uint) error {
	// 检查操作者权限
	opMember, err := s.getMember(groupID, operatorID)
	if err != nil {
		return err
	}
	if opMember.Role != model.RoleOwner && opMember.Role != model.RoleAdmin {
		return errors.New("无权限执行此操作")
	}

	// 不能踢自己
	if operatorID == targetID {
		return errors.New("不能踢出自己")
	}

	// 不能踢群主
	targetMember, err := s.getMember(groupID, targetID)
	if err != nil {
		return err
	}
	if targetMember.Role == model.RoleOwner {
		return errors.New("不能踢出群主")
	}

	return model.DB.Delete(&targetMember).Error
}

// TransferOwner 转让群主，仅群主可操作
func (s *GroupService) TransferOwner(groupID, ownerID, newOwnerID uint) error {
	var group model.Group
	if err := model.DB.First(&group, groupID).Error; err != nil {
		return errors.New("群组不存在")
	}
	if group.OwnerID != ownerID {
		return errors.New("仅群主可转让")
	}

	// 新群主必须是群成员
	newOwner, err := s.getMember(groupID, newOwnerID)
	if err != nil {
		return errors.New("新群主不是群成员")
	}

	// 更新群主
	group.OwnerID = newOwnerID
	if err := model.DB.Save(&group).Error; err != nil {
		return err
	}

	// 新群主角色改为 owner
	newOwner.Role = model.RoleOwner
	model.DB.Save(&newOwner)

	// 旧群主降级为管理员
	oldOwner, _ := s.getMember(groupID, ownerID)
	if oldOwner.ID != 0 {
		oldOwner.Role = model.RoleAdmin
		model.DB.Save(&oldOwner)
	}

	return nil
}

// MuteMember 禁言指定成员，仅群主和管理员可操作
// durationMinutes: 禁言分钟数
func (s *GroupService) MuteMember(groupID, operatorID, targetID uint, durationMinutes int) error {
	opMember, err := s.getMember(groupID, operatorID)
	if err != nil {
		return err
	}
	if opMember.Role != model.RoleOwner && opMember.Role != model.RoleAdmin {
		return errors.New("无权限执行禁言")
	}

	targetMember, err := s.getMember(groupID, targetID)
	if err != nil {
		return err
	}
	if targetMember.Role == model.RoleOwner {
		return errors.New("不能禁言群主")
	}

	until := time.Now().Add(time.Duration(durationMinutes) * time.Minute)
	return model.DB.Model(&targetMember).Update("muted_until", until).Error
}

// GetGroupMembers 获取群成员列表（含用户名/昵称）
func (s *GroupService) GetGroupMembers(groupID uint) ([]MemberInfo, error) {
	var members []model.GroupMember
	if err := model.DB.Where("group_id = ?", groupID).Find(&members).Error; err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return []MemberInfo{}, nil
	}

	// 批量获取用户信息
	userIDs := make([]uint, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}
	var users []model.User
	model.DB.Where("id IN ?", userIDs).Find(&users)
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]MemberInfo, len(members))
	for i, m := range members {
		u := userMap[m.UserID]
		result[i] = MemberInfo{
			ID:         m.ID,
			GroupID:    m.GroupID,
			UserID:     m.UserID,
			Username:   u.Username,
			Nickname:   u.Nickname,
			Role:       string(m.Role),
			MutedUntil: m.MutedUntil,
		}
	}
	return result, nil
}

// GetUserGroups 获取用户所在的群组列表
func (s *GroupService) GetUserGroups(userID uint) ([]model.Group, error) {
	var groupIDs []uint
	model.DB.Model(&model.GroupMember{}).
		Where("user_id = ?", userID).
		Pluck("group_id", &groupIDs)

	var groups []model.Group
	if len(groupIDs) > 0 {
		model.DB.Where("id IN ?", groupIDs).Find(&groups)
	}
	return groups, nil
}

// getMember 获取群成员信息
func (s *GroupService) getMember(groupID, userID uint) (*model.GroupMember, error) {
	var member model.GroupMember
	if err := model.DB.Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error; err != nil {
		return nil, errors.New("不是群成员")
	}
	return &member, nil
}

// GetGroupDetail 获取群组详情及当前用户在该群的成员信息
func (s *GroupService) GetGroupDetail(groupID, userID uint) (*model.Group, *model.GroupMember, error) {
	var group model.Group
	if err := model.DB.First(&group, groupID).Error; err != nil {
		return nil, nil, errors.New("群组不存在")
	}
	member, err := s.getMember(groupID, userID)
	if err != nil {
		return &group, nil, nil // 非成员也可看基本信息，但无操作权限
	}
	return &group, member, nil
}

// UpdateGroup 更新群资料，仅群主可操作
func (s *GroupService) UpdateGroup(groupID, operatorID uint, name, avatar, announce string) error {
	var group model.Group
	if err := model.DB.First(&group, groupID).Error; err != nil {
		return errors.New("群组不存在")
	}
	if group.OwnerID != operatorID {
		return errors.New("仅群主可编辑群资料")
	}
	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if avatar != "" {
		updates["avatar"] = avatar
	}
	// announce 允许置空（清空公告）
	updates["announce"] = announce
	if len(updates) > 0 {
		return model.DB.Model(&group).Updates(updates).Error
	}
	return nil
}

// SetAdmin 切换管理员身份，仅群主可操作
// 若 target 已是管理员则降为普通成员，否则提升为管理员
func (s *GroupService) SetAdmin(groupID, operatorID, targetID uint) error {
	var group model.Group
	if err := model.DB.First(&group, groupID).Error; err != nil {
		return errors.New("群组不存在")
	}
	if group.OwnerID != operatorID {
		return errors.New("仅群主可设置管理员")
	}
	if operatorID == targetID {
		return errors.New("群主已是最高权限")
	}
	targetMember, err := s.getMember(groupID, targetID)
	if err != nil {
		return err
	}
	if targetMember.Role == model.RoleOwner {
		return errors.New("不能修改群主的角色")
	}
	if targetMember.Role == model.RoleAdmin {
		targetMember.Role = model.RoleMember
	} else {
		targetMember.Role = model.RoleAdmin
	}
	return model.DB.Save(&targetMember).Error
}

// AddMember 添加成员到群组，群主/管理员可操作
func (s *GroupService) AddMember(groupID, operatorID, targetID uint) error {
	opMember, err := s.getMember(groupID, operatorID)
	if err != nil {
		return err
	}
	if opMember.Role != model.RoleOwner && opMember.Role != model.RoleAdmin {
		return errors.New("无权限执行此操作")
	}

	// 检查目标用户是否存在
	var user model.User
	if err := model.DB.First(&user, targetID).Error; err != nil {
		return errors.New("用户不存在")
	}

	// 检查是否已是成员
	var count int64
	model.DB.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, targetID).
		Count(&count)
	if count > 0 {
		return errors.New("该用户已是群成员")
	}

	member := &model.GroupMember{
		GroupID: groupID,
		UserID:  targetID,
		Role:    model.RoleMember,
	}
	return model.DB.Create(member).Error
}

// UnmuteMember 解除禁言，群主/管理员可操作
func (s *GroupService) UnmuteMember(groupID, operatorID, targetID uint) error {
	opMember, err := s.getMember(groupID, operatorID)
	if err != nil {
		return err
	}
	if opMember.Role != model.RoleOwner && opMember.Role != model.RoleAdmin {
		return errors.New("无权限执行此操作")
	}
	targetMember, err := s.getMember(groupID, targetID)
	if err != nil {
		return err
	}
	return model.DB.Model(&targetMember).Update("muted_until", nil).Error
}
