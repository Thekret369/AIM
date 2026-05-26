// Package service 实现用户配置的业务逻辑
package service

import (
	"encoding/json"

	"LanLine/internal/model"

	"gorm.io/datatypes"
)

// SettingsService 管理用户偏好设置
type SettingsService struct{}

// GetSettings 获取用户设置，未配置时返回默认值
func (s *SettingsService) GetSettings(userID uint) (*model.UserSettings, error) {
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return model.ParseSettings(user.Settings), nil
}

// UpdateSettings 全量更新用户设置
func (s *SettingsService) UpdateSettings(userID uint, input *model.UserSettings) (*model.UserSettings, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if err := model.DB.Model(&model.User{}).Where("id = ?", userID).
		Update("settings", datatypes.JSON(data)).Error; err != nil {
		return nil, err
	}
	return input, nil
}
