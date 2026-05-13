// Package service 实现用户认证业务逻辑（注册、登录、JWT 签发）
package service

import (
	"errors"
	"time"

	"AIM/internal/middleware"
	"AIM/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	JWTSecret    string
	JWTExpireHrs int
}

// Register 注册普通用户。AI 用户不可通过此方法创建，需由 AI 模块内部注册。
func (s *AuthService) Register(username, password, nickname string) (*model.User, error) {
	// 检查用户名是否已存在
	var existing model.User
	if err := model.DB.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, errors.New("用户名已被占用")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username: username,
		Password: string(hash),
		Nickname: nickname,
	}
	if err := model.DB.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// Login 验证用户名密码，返回 JWT token
func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	var user model.User
	if err := model.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("用户名或密码错误")
		}
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("用户名或密码错误")
	}

	// 递增 TokenVersion（原子操作，避免并发登录的竞态条件）
	if err := model.DB.Model(&user).Update("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
		return "", nil, err
	}
	// 重新读取最新版本号
	if err := model.DB.First(&user, user.ID).Error; err != nil {
		return "", nil, err
	}

	token, err := s.generateToken(&user)
	if err != nil {
		return "", nil, err
	}
	return token, &user, nil
}

// generateToken 签发 JWT，写入 token_version 用于顶出旧登录
func (s *AuthService) generateToken(user *model.User) (string, error) {
	claims := &middleware.Claims{
		UserID:       user.ID,
		Username:     user.Username,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.JWTExpireHrs) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.JWTSecret))
}

// GetProfile 获取用户个人信息
func (s *AuthService) GetProfile(userID uint) (*model.User, error) {
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// UpdateProfile 更新昵称、头像和简介
func (s *AuthService) UpdateProfile(userID uint, nickname, avatar, bio string) (*model.User, error) {
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if nickname != "" {
		updates["nickname"] = nickname
	}
	if avatar != "" {
		updates["avatar"] = avatar
	}
	// bio 允许置空
	updates["bio"] = bio
	if len(updates) > 0 {
		if err := model.DB.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	// 重新查询以获取最新数据
	model.DB.First(&user, userID)
	return &user, nil
}

// ChangePassword 修改密码，需验证旧密码
func (s *AuthService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("旧密码错误")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return model.DB.Model(&user).Update("password", string(hash)).Error
}
