// Package service 实现用户认证业务逻辑（注册、登录、JWT 签发）
package service

import (
	"errors"
	"time"

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

	token, err := s.generateToken(&user)
	if err != nil {
		return "", nil, err
	}
	return token, &user, nil
}

// generateToken 签发 JWT
func (s *AuthService) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Duration(s.JWTExpireHrs) * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.JWTSecret))
}
