package service

import (
	"errors"
	"sort"
	"strings"
	"time"

	"AIM/internal/model"
	"AIM/internal/ws"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChatService struct {
	Hub         *ws.Hub
	AIResponder AIResponder
}

const offlineSyncBatchSize = 100

type MessageSearchParams struct {
	Scope     string
	TargetID  uint
	Keyword   string
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PageSize  int
}

// AIResponder 由 AI 服务实现，聊天服务只负责在消息入库后通知它。
type AIResponder interface {
	HandleMessage(msg *model.Message)
}

func (s *ChatService) SendFromClient(msg *model.Message) error {
	if err := s.validateClientMessage(msg); err != nil {
		return err
	}
	if err := s.Send(msg); err != nil {
		return err
	}
	s.dispatchAI(msg)
	return nil
}

func (s *ChatService) validateClientMessage(msg *model.Message) error {
	if msg == nil {
		return errors.New("消息不能为空")
	}
	if !isClientMessageTypeAllowed(msg.Type) {
		return errors.New("不支持的消息类型")
	}
	hasUser := msg.ToUserID != nil && *msg.ToUserID > 0
	hasGroup := msg.GroupID != nil && *msg.GroupID > 0
	if hasUser == hasGroup {
		return errors.New("消息目标无效")
	}
	if hasUser {
		if err := s.validateDirectMessagePermission(msg.FromUserID, *msg.ToUserID); err != nil {
			return err
		}
	}
	if hasGroup {
		member, err := s.getGroupMember(*msg.GroupID, msg.FromUserID)
		if err != nil {
			return errors.New("不是群成员")
		}
		if member.IsMuted() {
			return errors.New("已被禁言")
		}
	}
	if err := s.validateQuoteMessage(msg); err != nil {
		return err
	}
	return nil
}

func isClientMessageTypeAllowed(messageType model.MessageType) bool {
	switch messageType {
	case model.MsgText, model.MsgImage, model.MsgFile, model.MsgAudio:
		return true
	default:
		return false
	}
}

func (s *ChatService) validateDirectMessagePermission(fromUserID, toUserID uint) error {
	if fromUserID == 0 || toUserID == 0 {
		return errors.New("消息目标无效")
	}
	if fromUserID == toUserID {
		return nil
	}

	var target model.User
	if err := model.DB.First(&target, toUserID).Error; err != nil {
		return errors.New("接收用户不存在")
	}
	if target.IsAI {
		return s.validateAIDirectMessagePermission(fromUserID, target.ID)
	}

	var blockedCount int64
	if err := model.DB.Model(&model.FriendRelation{}).
		Where("((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)) AND status = ?",
			fromUserID, toUserID, toUserID, fromUserID, "blocked").
		Count(&blockedCount).Error; err != nil {
		return err
	}
	if blockedCount > 0 {
		return errors.New("已被拉黑，不能发送消息")
	}

	var friendCount int64
	if err := model.DB.Model(&model.FriendRelation{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", fromUserID, toUserID, "accepted").
		Count(&friendCount).Error; err != nil {
		return err
	}
	if friendCount == 0 {
		return errors.New("不是好友，不能发送消息")
	}
	return nil
}

func (s *ChatService) validateAIDirectMessagePermission(fromUserID, botUserID uint) error {
	var bot model.AIBot
	err := model.DB.Where("user_id = ? AND status <> ?", botUserID, model.AIBotStatusDeleted).
		First(&bot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if bot.Status != model.AIBotStatusEnabled {
		return errors.New("AI 助手未启用")
	}
	if bot.IsSystem {
		return nil
	}
	if bot.OwnerID != nil && *bot.OwnerID == fromUserID {
		return nil
	}
	return errors.New("无权向该 AI 助手发送消息")
}

func (s *ChatService) validateQuoteMessage(msg *model.Message) error {
	if msg == nil || msg.QuoteMessageID == nil {
		return nil
	}
	if *msg.QuoteMessageID == 0 {
		return errors.New("引用消息无效")
	}

	var quote model.Message
	if err := model.DB.First(&quote, *msg.QuoteMessageID).Error; err != nil {
		return errors.New("引用消息不存在")
	}

	switch {
	case msg.IsToUser():
		if !quote.IsToUser() || quote.ToUserID == nil || msg.ToUserID == nil {
			return errors.New("引用消息不属于当前会话")
		}
		sameDirection := quote.FromUserID == msg.FromUserID && *quote.ToUserID == *msg.ToUserID
		reverseDirection := quote.FromUserID == *msg.ToUserID && *quote.ToUserID == msg.FromUserID
		if !sameDirection && !reverseDirection {
			return errors.New("引用消息不属于当前会话")
		}
	case msg.IsToGroup():
		if !quote.IsToGroup() || quote.GroupID == nil || msg.GroupID == nil || *quote.GroupID != *msg.GroupID {
			return errors.New("引用消息不属于当前群聊")
		}
	default:
		return errors.New("引用消息目标无效")
	}
	return nil
}

func (s *ChatService) dispatchAI(msg *model.Message) {
	if s == nil || s.AIResponder == nil || msg == nil {
		return
	}
	go s.AIResponder.HandleMessage(msg)
}

func (s *ChatService) MarkRead(p *ws.ReadReceiptPayload) {
	if p == nil {
		return
	}
	if p.GroupID > 0 && p.LastReadMsgID > 0 {
		s.markGroupReadWatermark(p)
		return
	}
	if len(p.MessageIDs) == 0 {
		return
	}
	s.markReadByMessageIDs(p)
}

func (s *ChatService) markReadByMessageIDs(p *ws.ReadReceiptPayload) {
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

	convType := model.ConversationUser
	convID := peerUserID
	if hasGroup {
		convType = model.ConversationGroup
		convID = groupID
	}
	if err := s.saveReadProgress(p.FromUserID, convType, convID, peerPtr, groupPtr, lastMsgID); err != nil {
		return
	}

	forward := *p
	forward.MessageIDs = validIDs
	forward.LastReadMsgID = lastMsgID
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

func (s *ChatService) markGroupReadWatermark(p *ws.ReadReceiptPayload) {
	if _, err := s.getGroupMember(p.GroupID, p.FromUserID); err != nil {
		return
	}

	var msg model.Message
	err := model.DB.Where("id = ? AND group_id = ? AND from_user_id <> ?", p.LastReadMsgID, p.GroupID, p.FromUserID).
		First(&msg).Error
	if err != nil {
		return
	}

	groupPtr := p.GroupID
	if err := s.saveReadProgress(p.FromUserID, model.ConversationGroup, p.GroupID, nil, &groupPtr, msg.ID); err != nil {
		return
	}

	forward := *p
	forward.PeerUserID = 0
	forward.GroupID = p.GroupID
	forward.MessageIDs = nil
	forward.LastReadMsgID = msg.ID

	var memberIDs []uint
	model.DB.Model(&model.GroupMember{}).
		Where("group_id = ?", p.GroupID).
		Pluck("user_id", &memberIDs)
	s.Hub.SendReadReceiptToUsers(memberIDs, &forward)
}

func (s *ChatService) saveReadProgress(userID uint, convType string, convID uint, peerPtr, groupPtr *uint, lastMsgID uint) error {
	if userID == 0 || convType == "" || convID == 0 || lastMsgID == 0 {
		return errors.New("无效的已读进度")
	}

	read := model.MessageRead{
		UserID:           userID,
		ConversationType: convType,
		ConversationID:   convID,
		PeerUserID:       peerPtr,
		GroupID:          groupPtr,
		LastReadMsgID:    lastMsgID,
	}
	return model.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "conversation_type"},
			{Name: "conversation_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"last_read_msg_id": gorm.Expr("CASE WHEN last_read_msg_id < ? THEN ? ELSE last_read_msg_id END", lastMsgID, lastMsgID),
			"peer_user_id":     peerPtr,
			"group_id":         groupPtr,
		}),
	}).Create(&read).Error
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
	if msg.IsToGroup() && msg.RecipientCount == 0 {
		var count int64
		model.DB.Model(&model.GroupMember{}).
			Where("group_id = ? AND user_id <> ?", *msg.GroupID, msg.FromUserID).
			Count(&count)
		msg.RecipientCount = int(count)
	}

	if err := model.DB.Create(msg).Error; err != nil {
		return err
	}

	preloadMessageRelations(model.DB).First(msg, msg.ID)

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

// SyncOfflineMessages pushes unread messages after a user reconnects.
func (s *ChatService) SyncOfflineMessages(userID uint) {
	if s == nil || s.Hub == nil || userID == 0 {
		return
	}
	msgs, err := s.getOfflineSyncMessages(userID)
	if err != nil {
		return
	}
	for i := range msgs {
		msg := msgs[i]
		s.Hub.SendToUser(userID, &msg)
	}
}

func (s *ChatService) getOfflineSyncMessages(userID uint) ([]model.Message, error) {
	if userID == 0 {
		return nil, nil
	}

	msgs, err := s.getOfflineUserMessages(userID)
	if err != nil {
		return nil, err
	}
	groupMsgs, err := s.getOfflineGroupMessages(userID)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, groupMsgs...)
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].ID < msgs[j].ID
	})
	return msgs, nil
}

func (s *ChatService) getOfflineUserMessages(userID uint) ([]model.Message, error) {
	type peerRow struct {
		PeerID uint
	}
	var peers []peerRow
	if err := model.DB.Model(&model.Message{}).
		Select("from_user_id AS peer_id").
		Where("group_id IS NULL AND to_user_id = ? AND from_user_id <> ?", userID, userID).
		Group("from_user_id").
		Find(&peers).Error; err != nil {
		return nil, err
	}

	var result []model.Message
	for _, peer := range peers {
		lastReadID := s.readWatermark(userID, model.ConversationUser, peer.PeerID)
		var msgs []model.Message
		if err := preloadMessageRelations(model.DB.Where("group_id IS NULL AND to_user_id = ? AND from_user_id = ? AND id > ?", userID, peer.PeerID, lastReadID)).
			Order("id ASC").
			Limit(offlineSyncBatchSize).
			Find(&msgs).Error; err != nil {
			return nil, err
		}
		result = append(result, msgs...)
	}
	return result, nil
}

func (s *ChatService) getOfflineGroupMessages(userID uint) ([]model.Message, error) {
	var memberships []model.GroupMember
	if err := model.DB.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
		return nil, err
	}

	var result []model.Message
	for _, member := range memberships {
		lastReadID := s.readWatermark(userID, model.ConversationGroup, member.GroupID)
		var msgs []model.Message
		if err := preloadMessageRelations(model.DB.Where("group_id = ? AND from_user_id <> ? AND id > ? AND created_at >= ?", member.GroupID, userID, lastReadID, member.CreatedAt)).
			Order("id ASC").
			Limit(offlineSyncBatchSize).
			Find(&msgs).Error; err != nil {
			return nil, err
		}
		result = append(result, msgs...)
	}
	return result, nil
}

func (s *ChatService) readWatermark(userID uint, convType string, convID uint) uint {
	var read model.MessageRead
	if err := model.DB.Select("last_read_msg_id").
		Where("user_id = ? AND conversation_type = ? AND conversation_id = ?", userID, convType, convID).
		First(&read).Error; err != nil {
		return 0
	}
	return read.LastReadMsgID
}

func preloadMessageRelations(db *gorm.DB) *gorm.DB {
	return db.Preload("FromUser").
		Preload("QuoteMessage").
		Preload("QuoteMessage.FromUser")
}

func (s *ChatService) GetHistory(userID, peerID uint, page, pageSize int, afterID uint) ([]model.Message, int64, error) {
	var msgs []model.Message
	var total int64

	where := "group_id IS NULL AND to_user_id IS NOT NULL AND ((from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?))"
	args := []interface{}{userID, peerID, peerID, userID}

	model.DB.Model(&model.Message{}).Where(where, args...).Count(&total)

	db := model.DB.Where(where, args...)
	if afterID > 0 {
		err := preloadMessageRelations(db.Where("id > ?", afterID)).
			Order("id ASC").Find(&msgs).Error
		if err != nil {
			return nil, 0, err
		}
	} else {
		err := preloadMessageRelations(db).
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
	member, err := s.getGroupMember(groupID, userID)
	if err != nil {
		return nil, 0, err
	}

	var msgs []model.Message
	var total int64

	baseWhere := "group_id = ? AND created_at >= ?"
	baseArgs := []interface{}{groupID, member.CreatedAt}

	model.DB.Model(&model.Message{}).Where(baseWhere, baseArgs...).Count(&total)

	db := model.DB.Where(baseWhere, baseArgs...)
	if afterID > 0 {
		err := preloadMessageRelations(db.Where("id > ?", afterID)).
			Order("id ASC").Find(&msgs).Error
		if err != nil {
			return nil, 0, err
		}
	} else {
		err := preloadMessageRelations(db).
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
	return s.SearchMessagesWithParams(userID, MessageSearchParams{
		Scope:    scope,
		TargetID: targetID,
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	})
}

func (s *ChatService) SearchMessagesWithParams(userID uint, params MessageSearchParams) ([]model.Message, int64, error) {
	params.Keyword = strings.TrimSpace(params.Keyword)
	if params.Keyword == "" && params.StartTime == nil && params.EndTime == nil {
		return []model.Message{}, 0, nil
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	if params.PageSize > 50 {
		params.PageSize = 50
	}

	query := model.DB.Model(&model.Message{})
	if params.Keyword != "" {
		like := "%" + escapeLikePattern(strings.ToLower(params.Keyword)) + "%"
		query = query.Where("(LOWER(messages.content) LIKE ? ESCAPE '\\' OR LOWER(messages.file_name) LIKE ? ESCAPE '\\')", like, like)
	}
	if params.StartTime != nil {
		query = query.Where("messages.created_at >= ?", *params.StartTime)
	}
	if params.EndTime != nil {
		query = query.Where("messages.created_at <= ?", *params.EndTime)
	}

	scope := strings.TrimSpace(params.Scope)
	if scope == "" {
		scope = "user"
	}
	switch scope {
	case "all", "global":
		query = query.Joins("LEFT JOIN group_members search_gm ON search_gm.group_id = messages.group_id AND search_gm.user_id = ?", userID).
			Where(`(
				(messages.group_id IS NULL AND messages.to_user_id IS NOT NULL AND (messages.from_user_id = ? OR messages.to_user_id = ?))
				OR (messages.group_id IS NOT NULL AND search_gm.id IS NOT NULL AND messages.created_at >= search_gm.created_at)
				OR (messages.to_user_id IS NULL AND messages.group_id IS NULL)
			)`, userID, userID)
	case "user":
		if params.TargetID == 0 {
			return nil, 0, errors.New("无效的搜索目标")
		}
		query = query.Where(
			"messages.group_id IS NULL AND messages.to_user_id IS NOT NULL AND ((messages.from_user_id = ? AND messages.to_user_id = ?) OR (messages.from_user_id = ? AND messages.to_user_id = ?))",
			userID, params.TargetID, params.TargetID, userID,
		)
	case "group":
		if params.TargetID == 0 {
			return nil, 0, errors.New("无效的搜索目标")
		}
		member, err := s.getGroupMember(params.TargetID, userID)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where("messages.group_id = ? AND messages.created_at >= ?", params.TargetID, member.CreatedAt)
	case "broadcast":
		query = query.Where("messages.to_user_id IS NULL AND messages.group_id IS NULL")
	default:
		return nil, 0, errors.New("无效的搜索范围")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var msgs []model.Message
	if err := preloadMessageRelations(query).
		Order("messages.created_at DESC, messages.id DESC").
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
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
		model.DB.Where("user_id = ? AND conversation_type = ? AND conversation_id = ?", userID, model.ConversationGroup, *groupID).First(&r)
		info.MyLastRead = r.LastReadMsgID
	} else {
		var myRead model.MessageRead
		model.DB.Where("user_id = ? AND conversation_type = ? AND conversation_id = ?", userID, model.ConversationUser, peerID).First(&myRead)
		info.MyLastRead = myRead.LastReadMsgID
		var peerRead model.MessageRead
		model.DB.Where("user_id = ? AND conversation_type = ? AND conversation_id = ?", peerID, model.ConversationUser, userID).First(&peerRead)
		info.PeerLastRead = peerRead.LastReadMsgID
	}
	return info
}

func (s *ChatService) GetGroupReads(userID, groupID uint) (map[uint]uint, error) {
	if _, err := s.getGroupMember(groupID, userID); err != nil {
		return nil, err
	}

	type row struct {
		UserID uint
		MaxID  uint
	}
	var rows []row
	if err := model.DB.Model(&model.MessageRead{}).
		Select("message_reads.user_id, MAX(message_reads.last_read_msg_id) AS max_id").
		Joins("JOIN group_members ON group_members.user_id = message_reads.user_id AND group_members.group_id = ?", groupID).
		Where("message_reads.conversation_type = ? AND message_reads.conversation_id = ?", model.ConversationGroup, groupID).
		Group("message_reads.user_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]uint)
	for _, r := range rows {
		result[r.UserID] = r.MaxID
	}
	return result, nil
}

func (s *ChatService) getGroupMember(groupID, userID uint) (*model.GroupMember, error) {
	var member model.GroupMember
	if err := model.DB.Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error; err != nil {
		return nil, errors.New("不属于该群组")
	}
	return &member, nil
}

func (s *ChatService) GetBroadcastHistory(page, pageSize int) ([]model.Message, int64, error) {
	var msgs []model.Message
	var total int64

	query := model.DB.Model(&model.Message{}).
		Where("to_user_id IS NULL AND group_id IS NULL")
	query.Count(&total)

	if err := preloadMessageRelations(query).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&msgs).Error; err != nil {
		return nil, 0, err
	}
	return msgs, total, nil
}
