// Package service 实现好友关系管理（申请、同意、拒绝、删除、备注、分组）
package service

import (
	"errors"

	"AIM/internal/model"

	"gorm.io/gorm"
)

// FriendInfo 好友信息（含对方用户资料）
type FriendInfo struct {
	ID       uint   `json:"id"`
	UserID   uint   `json:"user_id"`
	FriendID uint   `json:"friend_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
	Note     string `json:"note"`
}

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
		if err := model.DB.Save(&rel).Error; err != nil {
			return err
		}
		// 双向创建：被添加方也能在好友列表中看到对方
		reverse := &model.FriendRelation{
			UserID:   rel.FriendID,
			FriendID: rel.UserID,
			Status:   "accepted",
		}
		return model.DB.Create(reverse).Error
	} else {
		return model.DB.Delete(&rel).Error // 拒绝则删除记录
	}
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

// FriendList 获取好友列表（含对方用户名/昵称）
func (s *FriendService) FriendList(userID uint) ([]FriendInfo, error) {
	var list []model.FriendRelation
	if err := model.DB.Where("user_id = ? AND status = 'accepted'", userID).
		Find(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []FriendInfo{}, nil
		}
		return nil, err
	}
	if len(list) == 0 {
		return []FriendInfo{}, nil
	}

	// 批量获取好友用户信息
	friendIDs := make([]uint, len(list))
	for i, r := range list {
		friendIDs[i] = r.FriendID
	}
	var users []model.User
	model.DB.Where("id IN ?", friendIDs).Find(&users)
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]FriendInfo, len(list))
	for i, r := range list {
		u := userMap[r.FriendID]
		result[i] = FriendInfo{
			ID:       r.ID,
			UserID:   r.UserID,
			FriendID: r.FriendID,
			Username: u.Username,
			Nickname: u.Nickname,
			Status:   r.Status,
			Remark:   r.Remark,
			Note:     r.Note,
		}
	}
	return result, nil
}

// PendingRequests 获取待处理的好友申请（含申请者用户名/昵称）
func (s *FriendService) PendingRequests(userID uint) ([]FriendInfo, error) {
	var list []model.FriendRelation
	if err := model.DB.Where("friend_id = ? AND status = 'pending'", userID).
		Find(&list).Error; err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []FriendInfo{}, nil
	}

	// 批量获取申请者用户信息
	userIDs := make([]uint, len(list))
	for i, r := range list {
		userIDs[i] = r.UserID
	}
	var users []model.User
	model.DB.Where("id IN ?", userIDs).Find(&users)
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]FriendInfo, len(list))
	for i, r := range list {
		u := userMap[r.UserID]
		result[i] = FriendInfo{
			ID:       r.ID,
			UserID:   r.UserID,
			FriendID: r.FriendID,
			Username: u.Username,
			Nickname: u.Nickname,
			Status:   r.Status,
			Remark:   r.Remark,
			Note:     r.Note,
		}
	}
	return result, nil
}
