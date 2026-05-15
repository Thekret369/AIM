// Package service 实现消息路由与分发（单聊、群聊、广播）
package service

import (
	"errors"

	"AIM/internal/model"
	"AIM/internal/ws"
)

// ChatService 消息服务
// 依赖 ws.Hub 将消息实时推送到目标用户的 WebSocket 连接
// 同时处理输入状态路由和已读回执
type ChatService struct {
	Hub *ws.Hub
}

// MarkRead 标记消息已读，更新 MessageRead 表
// 单聊：通知消息发送者；群聊：广播到全群成员
func (s *ChatService) MarkRead(p *ws.ReadReceiptPayload) {
	if len(p.MessageIDs) == 0 {
		return
	}
	lastMsgID := p.MessageIDs[len(p.MessageIDs)-1]

	var msg model.Message
	if err := model.DB.First(&msg, p.MessageIDs[0]).Error; err != nil {
		return
	}

	var peerUserID, groupID *uint
	if msg.IsToUser() {
		peerUserID = &msg.FromUserID
	} else if msg.IsToGroup() {
		groupID = msg.GroupID
	}

	read := model.MessageRead{
		UserID:        p.FromUserID,
		PeerUserID:    peerUserID,
		GroupID:       groupID,
		LastReadMsgID: lastMsgID,
	}
	model.DB.Where("user_id = ? AND peer_user_id = ? AND group_id = ?",
		p.FromUserID, peerUserID, groupID).
		Assign(&read).
		FirstOrCreate(&read)

	if peerUserID != nil {
		s.Hub.SendReadReceipt(*peerUserID, p)
	} else if groupID != nil {
		var memberIDs []uint
		model.DB.Model(&model.GroupMember{}).
			Where("group_id = ?", *groupID).
			Pluck("user_id", &memberIDs)
		s.Hub.SendReadReceiptToUsers(memberIDs, p)
	}
}

// HandleTyping 路由输入状态给对方/群成员
func (s *ChatService) HandleTyping(p *ws.TypingPayload) {
	if p.GroupID > 0 {
		// 群聊：广播给群内成员（排除自己）
		var memberIDs []uint
		model.DB.Model(&model.GroupMember{}).
			Where("group_id = ?", p.GroupID).
			Pluck("user_id", &memberIDs)
		s.Hub.SendTypingToUsers(memberIDs, p)
	} else if p.ToUserID > 0 {
		s.Hub.SendTyping(p.ToUserID, p)
	}
}

// Send 发送消息并持久化到数据库，然后通过 Hub 推送到客户端
func (s *ChatService) Send(msg *model.Message) error {
	// 持久化
	if err := model.DB.Create(msg).Error; err != nil {
		return err
	}

	// 预加载发送者信息
	model.DB.Preload("FromUser").First(msg, msg.ID)

	// 按消息类型路由推送
	switch {
	case msg.IsBroadcast():
		// 广播：推送给所有在线用户
		s.Hub.Broadcast(msg)
	case msg.IsToGroup():
		// 群聊：推送给群内所有在线成员
		var memberIDs []uint
		model.DB.Model(&model.GroupMember{}).
			Where("group_id = ?", msg.GroupID).
			Pluck("user_id", &memberIDs)
		s.Hub.SendToUsers(memberIDs, msg)
	case msg.IsToUser():
		// 单聊：推送给接收者和发送者（同步多端消息）
		s.Hub.SendToUser(*msg.ToUserID, msg)
		if msg.FromUserID != *msg.ToUserID {
			s.Hub.SendToUser(msg.FromUserID, msg)
		}
	}
	return nil
}

// GetHistory 拉取两个用户之间的聊天记录（单聊）
func (s *ChatService) GetHistory(userID, peerID uint, page, pageSize int) ([]model.Message, int64, error) {
	var msgs []model.Message
	var total int64

	query := model.DB.Model(&model.Message{}).
		Where("(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
			userID, peerID, peerID, userID)

	query.Count(&total)

	if err := query.Preload("FromUser").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&msgs).Error; err != nil {
		return nil, 0, err
	}
	return msgs, total, nil
}

// GetGroupHistory 拉取群聊消息记录
func (s *ChatService) GetGroupHistory(userID, groupID uint, page, pageSize int) ([]model.Message, int64, error) {
	// 先校验用户是否在此群内
	var count int64
	model.DB.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count)
	if count == 0 {
		return nil, 0, errors.New("不属于该群组")
	}

	var msgs []model.Message
	var total int64

	query := model.DB.Model(&model.Message{}).Where("group_id = ?", groupID)
	query.Count(&total)

	if err := query.Preload("FromUser").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&msgs).Error; err != nil {
		return nil, 0, err
	}
	return msgs, total, nil
}

// GetBroadcastHistory 拉取广播消息记录
func (s *ChatService) GetBroadcastHistory(page, pageSize int) ([]model.Message, int64, error) {
	var msgs []model.Message
	var total int64

	query := model.DB.Model(&model.Message{}).
		Where("to_user_id IS NULL AND group_id IS NULL")
	query.Count(&total)

	if err := query.Preload("FromUser").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&msgs).Error; err != nil {
		return nil, 0, err
	}
	return msgs, total, nil
}
