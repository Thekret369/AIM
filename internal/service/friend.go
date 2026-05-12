// Package service 实现好友关系管理（申请、同意、拒绝、删除、备注、分组）
package service

import (
	"errors"

	"AIM/internal/model"

	"gorm.io/gorm"
)

type FriendService struct{}

// AddFriend 发送好友申请，status=pending
func (s *FriendService) AddFriend(userID, friendID uint, message string) (*model.FriendRelation, error) {
	if userID == friendID {
		return nil, errors.New("不能添加自己为好友")
	}

	// 检查是否已存在好友关系
	var existing model.FriendRelation
	err := model.DB.Where(
		"(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID, friendID, friendID, userID,
	).First(&existing).Error
	if err == nil {
		switch existing.Status {
		case "pending":
			return nil, errors.New("已有待处理的申请")
		case "accepted":
			return nil, errors.New("已经是好友")
		}
	}

	rel := &model.FriendRelation{
		UserID:   userID,
		FriendID: friendID,
		Status:   "pending",
		Remark:   message,
	}
	if err := model.DB.Create(rel).Error; err != nil {
		return nil, err
	}
	return rel, nil
}

// HandleRequest 同意或拒绝好友申请
func (s *FriendService) HandleRequest(relationID, userID uint, accept bool) error {
	var rel model.FriendRelation
	if err := model.DB.First(&rel, relationID).Error; err != nil {
		return errors.New("好友申请不存在")
	}
	// 只有接收申请的一方才能处理
	if rel.FriendID != userID {
		return errors.New("无权处理该申请")
	}
	if rel.Status != "pending" {
		return errors.New("申请已被处理")
	}

	if accept {
		rel.Status = "accepted"
	} else {
		return model.DB.Delete(&rel).Error // 拒绝则删除记录
	}
	return model.DB.Save(&rel).Error
}

// DeleteFriend 删除好友（双向删除）
func (s *FriendService) DeleteFriend(userID, friendID uint) error {
	return model.DB.Where(
		"(user_id = ? AND friend_id = ? AND status = 'accepted') OR (user_id = ? AND friend_id = ? AND status = 'accepted')",
		userID, friendID, friendID, userID,
	).Delete(&model.FriendRelation{}).Error
}

// UpdateRemark 修改好友备注名或备注信息
func (s *FriendService) UpdateRemark(userID, friendID uint, remark, note string, groupID *uint) error {
	return model.DB.Model(&model.FriendRelation{}).
		Where("user_id = ? AND friend_id = ? AND status = 'accepted'", userID, friendID).
		Updates(map[string]interface{}{"remark": remark, "note": note, "group_id": groupID}).Error
}

// FriendList 获取好友列表（含备注和分组信息）
func (s *FriendService) FriendList(userID uint) ([]model.FriendRelation, error) {
	var list []model.FriendRelation
	if err := model.DB.Where("user_id = ? AND status = 'accepted'", userID).
		Find(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.FriendRelation{}, nil
		}
		return nil, err
	}
	return list, nil
}

// PendingRequests 获取待处理的好友申请
func (s *FriendService) PendingRequests(userID uint) ([]model.FriendRelation, error) {
	var list []model.FriendRelation
	if err := model.DB.Where("friend_id = ? AND status = 'pending'", userID).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
