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

	// UPSERT：已存在则更新 LastReadMsgID（取较大值），不存在则创建
	existing := model.MessageRead{}
	q := model.DB.Where("user_id = ?", p.FromUserID)
	if peerUserID == nil {
		q = q.Where("peer_user_id IS NULL")
	} else {
		q = q.Where("peer_user_id = ?", *peerUserID)
	}
	if groupID == nil {
		q = q.Where("group_id IS NULL")
	} else {
		q = q.Where("group_id = ?", *groupID)
	}
	err := q.First(&existing).Error
	if err != nil {
		model.DB.Create(&model.MessageRead{
			UserID: p.FromUserID, PeerUserID: peerUserID, GroupID: groupID,
			LastReadMsgID: lastMsgID,
		})
	} else if lastMsgID > existing.LastReadMsgID {
		model.DB.Model(&existing).Update("last_read_msg_id", lastMsgID)
	}

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
// afterID > 0 时为增量同步：返回 id > afterID 的消息，按 id 升序
func (s *ChatService) GetHistory(userID, peerID uint, page, pageSize int, afterID uint) ([]model.Message, int64, error) {
	var msgs []model.Message
	var total int64

	where := "(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)"
	args := []interface{}{userID, peerID, peerID, userID}

	model.DB.Model(&model.Message{}).Where(where, args...).Count(&total)

	db := model.DB.Where(where, args...)
	if afterID > 0 {
		// 增量同步：只返回客户端缺失的新消息，按 id 升序
		err := db.Where("id > ?", afterID).
			Preload("FromUser").Order("id ASC").Find(&msgs).Error
		if err != nil {
			return nil, 0, err
		}
	} else {
		// 首次加载：分页拉取，最新在前
		err := db.Preload("FromUser").
			Order("created_at DESC").
			Offset((page - 1) * pageSize).
			Limit(pageSize).
			Find(&msgs).Error
		if err != nil {
			return nil, 0, err
		}
	}
	return msgs, total, nil
}

// GetGroupHistory 拉取群聊消息记录
// afterID > 0 时为增量同步：返回 id > afterID 的消息，按 id 升序
func (s *ChatService) GetGroupHistory(userID, groupID uint, page, pageSize int, afterID uint) ([]model.Message, int64, error) {
	var count int64
	model.DB.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count)
	if count == 0 {
		return nil, 0, errors.New("不属于该群组")
	}

	var msgs []model.Message
	var total int64

	model.DB.Model(&model.Message{}).Where("group_id = ?", groupID).Count(&total)

	db := model.DB.Where("group_id = ?", groupID)
	if afterID > 0 {
		// 增量同步：只返回客户端缺失的新消息，按 id 升序
		err := db.Where("id > ?", afterID).
			Preload("FromUser").Order("id ASC").Find(&msgs).Error
		if err != nil {
			return nil, 0, err
		}
	} else {
		// 首次加载：分页拉取，最新在前
		err := db.Preload("FromUser").
			Order("created_at DESC").
			Offset((page - 1) * pageSize).
			Limit(pageSize).
			Find(&msgs).Error
		if err != nil {
			return nil, 0, err
		}
	}
	return msgs, total, nil
}

// ReadInfo 会话已读信息
type ReadInfo struct {
	MyLastRead   uint `json:"my_last_read"`   // 我读到的最后一条对方消息 ID
	PeerLastRead uint `json:"peer_last_read"` // 对方读到的最后一条我的消息 ID（仅单聊）
}

// GetReadInfo 获取用户在指定会话中的已读位置
func (s *ChatService) GetReadInfo(userID, peerID uint, groupID *uint) ReadInfo {
	var info ReadInfo
	if groupID != nil {
		var r model.MessageRead
		model.DB.Where("user_id = ? AND group_id = ? AND peer_user_id IS NULL", userID, *groupID).First(&r)
		info.MyLastRead = r.LastReadMsgID
	} else {
		var myRead model.MessageRead
		model.DB.Where("user_id = ? AND peer_user_id = ? AND group_id IS NULL", userID, peerID).First(&myRead)
		info.MyLastRead = myRead.LastReadMsgID
		var peerRead model.MessageRead
		model.DB.Where("user_id = ? AND peer_user_id = ? AND group_id IS NULL", peerID, userID).First(&peerRead)
		info.PeerLastRead = peerRead.LastReadMsgID
	}
	return info
}

// GetGroupReads 返回群内各用户最后已读消息 ID，前端用于重建已读扇形图
// 返回 map[userID]lastReadMsgID，不包含查询者本人（pie 图排除发送者）
func (s *ChatService) GetGroupReads(groupID uint) (map[uint]uint, error) {
	type row struct {
		UserID uint
		MaxID  uint
	}
	var rows []row
	if err := model.DB.Model(&model.MessageRead{}).
		Select("user_id, MAX(last_read_msg_id) AS max_id").
		Where("group_id = ? AND peer_user_id IS NULL", groupID).
		Group("user_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]uint)
	for _, r := range rows {
		result[r.UserID] = r.MaxID
	}
	return result, nil
}
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
