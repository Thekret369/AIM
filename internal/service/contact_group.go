// Package service 实现联系人分组管理
package service

import (
	"errors"

	"AIM/internal/model"
)

type ContactGroupService struct{}

// Create 创建联系人分组
func (s *ContactGroupService) Create(userID uint, name string) (*model.ContactGroup, error) {
	group := &model.ContactGroup{
		UserID: userID,
		Name:   name,
	}
	if err := model.DB.Create(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

// Update 编辑分组名称
func (s *ContactGroupService) Update(groupID, userID uint, name string) error {
	result := model.DB.Model(&model.ContactGroup{}).
		Where("id = ? AND user_id = ?", groupID, userID).
		Update("name", name)
	if result.RowsAffected == 0 {
		return errors.New("分组不存在")
	}
	return result.Error
}

// Delete 删除联系人分组（分组下的好友不会被删除，仅清空 group_id）
func (s *ContactGroupService) Delete(groupID, userID uint) error {
	// 先清空该分组下所有好友的 group_id
	model.DB.Model(&model.FriendRelation{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("group_id", nil)

	result := model.DB.Where("id = ? AND user_id = ?", groupID, userID).
		Delete(&model.ContactGroup{})
	if result.RowsAffected == 0 {
		return errors.New("分组不存在")
	}
	return result.Error
}

// List 获取用户的所有联系人分组
func (s *ContactGroupService) List(userID uint) ([]model.ContactGroup, error) {
	var groups []model.ContactGroup
	if err := model.DB.Where("user_id = ?", userID).
		Order("sort_order ASC").
		Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}
