// Package service 实现联系人分组管理
package service

import (
	"errors"
	"strings"

	"AIM/internal/model"

	"gorm.io/gorm"
)

type ContactGroupService struct{}

// Create 创建联系人分组
func (s *ContactGroupService) Create(userID uint, name string) (*model.ContactGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("分组名称不能为空")
	}
	var user model.User
	if err := model.DB.Select("id").First(&user, userID).Error; err != nil {
		return nil, errors.New("用户不存在")
	}
	group := &model.ContactGroup{
		UserID: userID,
		Name:   name,
	}
	if err := model.DB.Create(group).Error; err != nil {
		if isUniqueConstraintError(err) {
			return nil, errors.New("分组名称已存在")
		}
		return nil, err
	}
	return group, nil
}

// Update 编辑分组名称
func (s *ContactGroupService) Update(groupID, userID uint, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("分组名称不能为空")
	}
	var group model.ContactGroup
	if err := model.DB.Where("id = ? AND user_id = ?", groupID, userID).
		First(&group).Error; err != nil {
		return errors.New("分组不存在")
	}
	if err := model.DB.Model(&group).Update("name", name).Error; err != nil {
		if isUniqueConstraintError(err) {
			return errors.New("分组名称已存在")
		}
		return err
	}
	return nil
}

// Delete 删除联系人分组（分组下的好友不会被删除，仅清空 group_id）
func (s *ContactGroupService) Delete(groupID, userID uint) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var group model.ContactGroup
		if err := tx.Where("id = ? AND user_id = ?", groupID, userID).
			First(&group).Error; err != nil {
			return errors.New("分组不存在")
		}
		// 清空关联好友和删除分组必须同事务提交，避免留下悬挂 group_id。
		if err := tx.Model(&model.FriendRelation{}).
			Where("group_id = ? AND user_id = ?", groupID, userID).
			Update("group_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&group).Error
	})
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
