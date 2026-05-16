package service

import (
	"errors"
	"strings"

	"AIM/internal/model"
	"AIM/internal/ws"

	"gorm.io/gorm"
)

type ChatService struct {
	Hub *ws.Hub
}

func (s *ChatService) SendFromClient(msg *model.Message) error {
	if err := s.validateClientMessage(msg); err != nil {
		return err
	}
	return s.Send(msg)
}

func (s *ChatService) validateClientMessage(msg *model.Message) error {
	if msg == nil {
		return errors.New("消息不能为空")
	}
	hasUser := msg.ToUserID != nil && *msg.ToUserID > 0
	hasGroup := msg.GroupID != nil && *msg.GroupID > 0
	if hasUser == hasGroup {
		return errors.New("消息目标无效")
	}
	if hasGroup {
		var member model.GroupMember
		err := model.DB.Where("group_id = ? AND user_id = ?", *msg.GroupID, msg.FromUserID).First(&member).Error
		if err != nil {
			return errors.New("不是群成员")
		}
		if member.IsMuted() {
			return errors.New("已被禁言")
		}
	}
	return nil
}

func (s *ChatService) MarkRead(p *ws.ReadReceiptPayload) {
	if p == nil || len(p.MessageIDs) == 0 {
		return
	}

	var msgs []model.Message
	if err := model.DB.Where("id IN ?", p.MessageIDs).
		Order("id ASC").
		Find(&msgs).Error; err != nil || len(msgs) == 0 {
		return
	}

	var peerUserID uint
	var groupID uint
	hasPeer := false
	hasGroup := false
	validIDs := make([]uint, 0, len(msgs))

	for _, msg := range msgs {
		switch {
		case msg.IsToUser():
			if msg.ToUserID == nil || *msg.ToUserID != p.FromUserID || msg.FromUserID == p.FromUserID || hasGroup {
				continue
			}
			if !hasPeer {
				peerUserID = msg.FromUserID
				hasPeer = true
			}
			if peerUserID != msg.FromUserID {
				continue
			}
			validIDs = append(validIDs, msg.ID)
		case msg.IsToGroup():
			if msg.GroupID == nil || msg.FromUserID == p.FromUserID || hasPeer {
				continue
			}
			if !hasGroup {
				groupID = *msg.GroupID
				hasGroup = true
			}
			if groupID != *msg.GroupID {
				continue
			}
			validIDs = append(validIDs, msg.ID)
		}
	}
	if len(validIDs) == 0 {
		return
	}

	if hasGroup {
		var memberCount int64
		model.DB.Model(&model.GroupMember{}).
			Where("group_id = ? AND user_id = ?", groupID, p.FromUserID).
			Count(&memberCount)
		if memberCount == 0 {
			return
		}
	}

	lastMsgID := validIDs[len(validIDs)-1]
	var peerPtr, groupPtr *uint
	if hasPeer {
		peerPtr = &peerUserID
	}
	if hasGroup {
		groupPtr = &groupID
	}

	existing := model.MessageRead{}
	q := model.DB.Where("user_id = ?", p.FromUserID)
	if peerPtr == nil {
		q = q.Where("peer_user_id IS NULL")
	} else {
		q = q.Where("peer_user_id = ?", *peerPtr)
	}
	if groupPtr == nil {
		q = q.Where("group_id IS NULL")
	} else {
		q = q.Where("group_id = ?", *groupPtr)
	}
	err := q.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		model.DB.Create(&model.MessageRead{
			UserID:        p.FromUserID,
			PeerUserID:    peerPtr,
			GroupID:       groupPtr,
			LastReadMsgID: lastMsgID,
		})
	} else if err != nil {
		return
	} else if lastMsgID > existing.LastReadMsgID {
		model.DB.Model(&existing).Update("last_read_msg_id", lastMsgID)
	}

	forward := *p
	forward.MessageIDs = validIDs
	if peerPtr != nil {
		forward.PeerUserID = *peerPtr
		forward.GroupID = 0
		s.Hub.SendReadReceipt(*peerPtr, &forward)
	} else if groupPtr != nil {
		forward.PeerUserID = 0
		forward.GroupID = *groupPtr
		var memberIDs []uint
		model.DB.Model(&model.GroupMember{}).
			Where("group_id = ?", *groupPtr).
			Pluck("user_id", &memberIDs)
		s.Hub.SendReadReceiptToUsers(memberIDs, &forward)
	}
}

func (s *ChatService) HandleTyping(p *ws.TypingPayload) {
	if p == nil {
		return
	}
	if p.GroupID > 0 {
		var memberIDs []uint
		model.DB.Model(&model.GroupMember{}).
			Where("group_id = ?", p.GroupID).
			Pluck("user_id", &memberIDs)
		s.Hub.SendTypingToUsers(memberIDs, p)
	} else if p.ToUserID > 0 {
		s.Hub.SendTyping(p.ToUserID, p)
	}
}

func (s *ChatService) Send(msg *model.Message) error {
	if err := model.DB.Create(msg).Error; err != nil {
		return err
	}

	model.DB.Preload("FromUser").First(msg, msg.ID)

	switch {
	case msg.IsBroadcast():
		s.Hub.Broadcast(msg)
	case msg.IsToGroup():
		var memberIDs []uint
		model.DB.Model(&model.GroupMember{}).
			Where("group_id = ?", *msg.GroupID).
			Pluck("user_id", &memberIDs)
		s.Hub.SendToUsers(memberIDs, msg)
	case msg.IsToUser():
		s.Hub.SendToUser(*msg.ToUserID, msg)
		if msg.FromUserID != *msg.ToUserID {
			s.Hub.SendToUser(msg.FromUserID, msg)
		}
	}
	return nil
}

func (s *ChatService) GetHistory(userID, peerID uint, page, pageSize int, afterID uint) ([]model.Message, int64, error) {
	var msgs []model.Message
	var total int64

	where := "(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)"
	args := []interface{}{userID, peerID, peerID, userID}

	model.DB.Model(&model.Message{}).Where(where, args...).Count(&total)

	db := model.DB.Where(where, args...)
	if afterID > 0 {
		err := db.Where("id > ?", afterID).
			Preload("FromUser").Order("id ASC").Find(&msgs).Error
		if err != nil {
			return nil, 0, err
		}
	} else {
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
		err := db.Where("id > ?", afterID).
			Preload("FromUser").Order("id ASC").Find(&msgs).Error
		if err != nil {
			return nil, 0, err
		}
	} else {
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

func (s *ChatService) SearchMessages(userID uint, scope string, targetID uint, keyword string, page, pageSize int) ([]model.Message, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []model.Message{}, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}

	like := "%" + escapeLikePattern(strings.ToLower(keyword)) + "%"
	query := model.DB.Model(&model.Message{}).
		Where("(LOWER(content) LIKE ? ESCAPE '\\' OR LOWER(file_name) LIKE ? ESCAPE '\\')", like, like)

	switch scope {
	case "user":
		if targetID == 0 {
			return nil, 0, errors.New("无效的搜索目标")
		}
		query = query.Where(
			"(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
			userID, targetID, targetID, userID,
		)
	case "group":
		if targetID == 0 {
			return nil, 0, errors.New("无效的搜索目标")
		}
		var memberCount int64
		model.DB.Model(&model.GroupMember{}).
			Where("group_id = ? AND user_id = ?", targetID, userID).
			Count(&memberCount)
		if memberCount == 0 {
			return nil, 0, errors.New("不属于该群组")
		}
		query = query.Where("group_id = ?", targetID)
	case "broadcast":
		query = query.Where("to_user_id IS NULL AND group_id IS NULL")
	default:
		return nil, 0, errors.New("无效的搜索范围")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var msgs []model.Message
	if err := query.Preload("FromUser").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&msgs).Error; err != nil {
		return nil, 0, err
	}
	return msgs, total, nil
}

func escapeLikePattern(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

type ReadInfo struct {
	MyLastRead   uint `json:"my_last_read"`
	PeerLastRead uint `json:"peer_last_read"`
}

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
