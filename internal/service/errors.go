package service

import (
	"errors"
	"strings"
)

var (
	ErrMessageDeleteForbidden = errors.New("无权删除该消息")
	ErrMessageNotFound        = errors.New("消息不存在")
	ErrMessageRecallForbidden = errors.New("只能撤回自己发送的消息")
	ErrMessageRecallExpired   = errors.New("消息已超过 2 分钟，不能撤回")
	ErrMessageAlreadyRecalled = errors.New("消息已撤回")
)

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "constraint failed")
}
