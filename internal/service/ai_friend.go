package service

import (
	"errors"

	"AIM/internal/model"

	"gorm.io/gorm"
)

// ensureSystemAIFriendsForUserTx 将当前普通用户与所有启用的内置 AI 建立双向好友关系。
func ensureSystemAIFriendsForUserTx(tx *gorm.DB, userID uint) error {
	if userID == 0 {
		return nil
	}

	var user model.User
	if err := tx.Select("id", "is_ai").First(&user, userID).Error; err != nil {
		return err
	}
	if user.IsAI {
		return nil
	}

	var botUserIDs []uint
	if err := tx.Model(&model.AIBot{}).
		Where("is_system = ? AND status = ?", true, model.AIBotStatusEnabled).
		Pluck("user_id", &botUserIDs).Error; err != nil {
		return err
	}

	for _, botUserID := range botUserIDs {
		if err := ensureAcceptedFriendPairTx(tx, userID, botUserID); err != nil {
			return err
		}
	}
	return nil
}

// ensureSystemAIFriendForAllUsersTx 将指定内置 AI 补齐到所有普通账号的好友列表。
func ensureSystemAIFriendForAllUsersTx(tx *gorm.DB, botUserID uint) error {
	if botUserID == 0 {
		return nil
	}

	var users []model.User
	if err := tx.Select("id").Where("is_ai = ?", false).Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if err := ensureAcceptedFriendPairTx(tx, user.ID, botUserID); err != nil {
			return err
		}
	}
	return nil
}

func ensureAcceptedFriendPairTx(tx *gorm.DB, userID, friendID uint) error {
	if userID == 0 || friendID == 0 || userID == friendID {
		return nil
	}
	if err := ensureAcceptedFriendRelationTx(tx, userID, friendID); err != nil {
		return err
	}
	return ensureAcceptedFriendRelationTx(tx, friendID, userID)
}

func ensureAcceptedFriendRelationTx(tx *gorm.DB, userID, friendID uint) error {
	var rel model.FriendRelation
	err := tx.Where("user_id = ? AND friend_id = ?", userID, friendID).First(&rel).Error
	if err == nil {
		if rel.Status != "accepted" {
			return tx.Model(&rel).Update("status", "accepted").Error
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return tx.Create(&model.FriendRelation{
		UserID:   userID,
		FriendID: friendID,
		Status:   "accepted",
	}).Error
}
