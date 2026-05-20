package service

import (
	"testing"
	"time"

	"AIM/internal/model"
	"AIM/internal/ws"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupChatSecurityTest(t *testing.T) (*ChatService, *GroupService) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&model.User{},
		&model.Message{},
		&model.FriendRelation{},
		&model.ContactGroup{},
		&model.Group{},
		&model.GroupMember{},
		&model.Announcement{},
		&model.MessageRead{},
		&model.AIBot{},
		&model.AIKnowledgeBase{},
		&model.AIKnowledgeDocument{},
		&model.AIBotKnowledgeBase{},
		&model.AITokenUsage{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	model.DB = db
	if err := model.EnsureIndexes(); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}

	return &ChatService{Hub: ws.NewHub()}, &GroupService{}
}

func createSecurityUser(t *testing.T, username string) *model.User {
	t.Helper()

	user := &model.User{Username: username, Password: "test-password", Nickname: username}
	if err := model.DB.Create(user).Error; err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return user
}

func createAcceptedFriendPair(t *testing.T, userID, friendID uint) {
	t.Helper()

	rels := []model.FriendRelation{
		{UserID: userID, FriendID: friendID, Status: "accepted"},
		{UserID: friendID, FriendID: userID, Status: "accepted"},
	}
	for i := range rels {
		if err := model.DB.Create(&rels[i]).Error; err != nil {
			t.Fatalf("create accepted friend relation: %v", err)
		}
	}
}

func TestSendFromClientRejectsBroadcast(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	user := createSecurityUser(t, "ws_broadcast_user")

	msg := &model.Message{
		Type:       model.MsgText,
		FromUserID: user.ID,
		Content:    "should not broadcast from client",
	}

	if err := chatSvc.SendFromClient(msg); err == nil {
		t.Fatal("expected client broadcast to be rejected")
	}

	var count int64
	model.DB.Model(&model.Message{}).Where("content = ?", msg.Content).Count(&count)
	if count != 0 {
		t.Fatalf("rejected client broadcast was persisted, count=%d", count)
	}
}

func TestSendFromClientRejectsUnknownMessageType(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "type_alice")
	bob := createSecurityUser(t, "type_bob")
	createAcceptedFriendPair(t, alice.ID, bob.ID)

	toBob := bob.ID
	msg := &model.Message{
		Type:       model.MessageType("admin"),
		FromUserID: alice.ID,
		ToUserID:   &toBob,
		Content:    "invalid type",
	}

	if err := chatSvc.SendFromClient(msg); err == nil {
		t.Fatal("expected unknown message type to be rejected")
	}
}

func TestSendFromClientRejectsNonFriendDirectMessage(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "direct_alice")
	bob := createSecurityUser(t, "direct_bob")

	toBob := bob.ID
	msg := &model.Message{
		Type:       model.MsgText,
		FromUserID: alice.ID,
		ToUserID:   &toBob,
		Content:    "non friend direct",
	}

	if err := chatSvc.SendFromClient(msg); err == nil {
		t.Fatal("expected non-friend direct message to be rejected")
	}

	var count int64
	model.DB.Model(&model.Message{}).Where("content = ?", msg.Content).Count(&count)
	if count != 0 {
		t.Fatalf("rejected direct message was persisted, count=%d", count)
	}
}

func TestSendFromClientAcceptsFriendDirectMessage(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "friend_alice")
	bob := createSecurityUser(t, "friend_bob")
	createAcceptedFriendPair(t, alice.ID, bob.ID)

	toBob := bob.ID
	msg := &model.Message{
		Type:       model.MsgText,
		FromUserID: alice.ID,
		ToUserID:   &toBob,
		Content:    "friend direct",
	}

	if err := chatSvc.SendFromClient(msg); err != nil {
		t.Fatalf("send friend direct message: %v", err)
	}
	if msg.ID == 0 {
		t.Fatal("expected direct message to be persisted")
	}
}

func TestSendFromClientRejectsNonMemberGroupMessage(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "group_owner_sec")
	outsider := createSecurityUser(t, "group_outsider_sec")
	group, err := groupSvc.CreateGroup("secure_group", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	msg := &model.Message{
		Type:       model.MsgText,
		FromUserID: outsider.ID,
		GroupID:    &group.ID,
		Content:    "should not enter group",
	}

	if err := chatSvc.SendFromClient(msg); err == nil {
		t.Fatal("expected non-member group message to be rejected")
	}

	var count int64
	model.DB.Model(&model.Message{}).Where("content = ?", msg.Content).Count(&count)
	if count != 0 {
		t.Fatalf("rejected group message was persisted, count=%d", count)
	}
}

func TestSendFromClientRejectsMutedGroupMember(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "group_owner_mute")
	member := createSecurityUser(t, "group_member_mute")
	group, err := groupSvc.CreateGroup("muted_group", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, member.ID); err != nil {
		t.Fatalf("join group: %v", err)
	}
	if err := groupSvc.MuteMember(group.ID, owner.ID, member.ID, 10); err != nil {
		t.Fatalf("mute member: %v", err)
	}

	msg := &model.Message{
		Type:       model.MsgText,
		FromUserID: member.ID,
		GroupID:    &group.ID,
		Content:    "muted message",
	}

	if err := chatSvc.SendFromClient(msg); err == nil {
		t.Fatal("expected muted member message to be rejected")
	}
}

func TestMarkReadIgnoresMessagesNotAddressedToReader(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	sender := createSecurityUser(t, "read_sender")
	reader := createSecurityUser(t, "read_reader")
	third := createSecurityUser(t, "read_third")

	toReader := reader.ID
	valid := &model.Message{
		Type:       model.MsgText,
		FromUserID: sender.ID,
		ToUserID:   &toReader,
		Content:    "valid read",
	}
	if err := model.DB.Create(valid).Error; err != nil {
		t.Fatalf("create valid message: %v", err)
	}

	toSender := sender.ID
	invalid := &model.Message{
		Type:       model.MsgText,
		FromUserID: third.ID,
		ToUserID:   &toSender,
		Content:    "invalid read",
	}
	if err := model.DB.Create(invalid).Error; err != nil {
		t.Fatalf("create invalid message: %v", err)
	}

	chatSvc.MarkRead(&ws.ReadReceiptPayload{
		FromUserID: reader.ID,
		MessageIDs: []uint{valid.ID, invalid.ID},
	})

	var read model.MessageRead
	if err := model.DB.Where("user_id = ? AND peer_user_id = ? AND group_id IS NULL", reader.ID, sender.ID).
		First(&read).Error; err != nil {
		t.Fatalf("expected valid read record: %v", err)
	}
	if read.LastReadMsgID != valid.ID {
		t.Fatalf("expected last read id %d, got %d", valid.ID, read.LastReadMsgID)
	}
	if read.ConversationType != model.ConversationUser || read.ConversationID != sender.ID {
		t.Fatalf("expected user conversation %d, got %s/%d", sender.ID, read.ConversationType, read.ConversationID)
	}

	var count int64
	model.DB.Model(&model.MessageRead{}).
		Where("user_id = ? AND peer_user_id = ?", reader.ID, third.ID).
		Count(&count)
	if count != 0 {
		t.Fatalf("invalid peer read record was created, count=%d", count)
	}
}

func TestMarkReadRejectsNonMemberGroupReader(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "read_group_owner")
	member := createSecurityUser(t, "read_group_member")
	outsider := createSecurityUser(t, "read_group_outsider")
	group, err := groupSvc.CreateGroup("read_group", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, member.ID); err != nil {
		t.Fatalf("join group: %v", err)
	}

	msg := &model.Message{
		Type:       model.MsgText,
		FromUserID: member.ID,
		GroupID:    &group.ID,
		Content:    "group read",
	}
	if err := model.DB.Create(msg).Error; err != nil {
		t.Fatalf("create group message: %v", err)
	}

	chatSvc.MarkRead(&ws.ReadReceiptPayload{
		FromUserID: outsider.ID,
		MessageIDs: []uint{msg.ID},
	})

	var count int64
	model.DB.Model(&model.MessageRead{}).Where("user_id = ?", outsider.ID).Count(&count)
	if count != 0 {
		t.Fatalf("non-member group read record was created, count=%d", count)
	}
}

func TestMarkReadUsesAtomicHighWatermark(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	sender := createSecurityUser(t, "read_sender_highwater")
	reader := createSecurityUser(t, "read_reader_highwater")
	toReader := reader.ID

	first := &model.Message{Type: model.MsgText, FromUserID: sender.ID, ToUserID: &toReader, Content: "first"}
	second := &model.Message{Type: model.MsgText, FromUserID: sender.ID, ToUserID: &toReader, Content: "second"}
	if err := model.DB.Create(first).Error; err != nil {
		t.Fatalf("create first message: %v", err)
	}
	if err := model.DB.Create(second).Error; err != nil {
		t.Fatalf("create second message: %v", err)
	}

	chatSvc.MarkRead(&ws.ReadReceiptPayload{FromUserID: reader.ID, MessageIDs: []uint{second.ID}})
	chatSvc.MarkRead(&ws.ReadReceiptPayload{FromUserID: reader.ID, MessageIDs: []uint{first.ID}})

	var reads []model.MessageRead
	if err := model.DB.Where("user_id = ? AND conversation_type = ? AND conversation_id = ?", reader.ID, model.ConversationUser, sender.ID).
		Find(&reads).Error; err != nil {
		t.Fatalf("query reads: %v", err)
	}
	if len(reads) != 1 {
		t.Fatalf("expected one read row, got %d", len(reads))
	}
	if reads[0].LastReadMsgID != second.ID {
		t.Fatalf("expected high watermark %d, got %d", second.ID, reads[0].LastReadMsgID)
	}
}

func TestMarkGroupReadWatermark(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "read_group_watermark_owner")
	member := createSecurityUser(t, "read_group_watermark_member")
	group, err := groupSvc.CreateGroup("read_group_watermark", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, member.ID); err != nil {
		t.Fatalf("join group: %v", err)
	}

	msg := &model.Message{
		Type:       model.MsgText,
		FromUserID: owner.ID,
		GroupID:    &group.ID,
		Content:    "group watermark",
	}
	if err := model.DB.Create(msg).Error; err != nil {
		t.Fatalf("create group message: %v", err)
	}

	chatSvc.MarkRead(&ws.ReadReceiptPayload{
		FromUserID:    member.ID,
		GroupID:       group.ID,
		LastReadMsgID: msg.ID,
	})

	var read model.MessageRead
	if err := model.DB.Where("user_id = ? AND conversation_type = ? AND conversation_id = ?", member.ID, model.ConversationGroup, group.ID).
		First(&read).Error; err != nil {
		t.Fatalf("expected group read record: %v", err)
	}
	if read.LastReadMsgID != msg.ID {
		t.Fatalf("expected last read id %d, got %d", msg.ID, read.LastReadMsgID)
	}
}

func TestGetGroupReadsRejectsNonMember(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "reads_group_owner")
	outsider := createSecurityUser(t, "reads_group_outsider")
	group, err := groupSvc.CreateGroup("reads_group", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	if _, err := chatSvc.GetGroupReads(outsider.ID, group.ID); err == nil {
		t.Fatal("expected non-member group reads query to be rejected")
	}
}

func TestSearchMessagesUserScope(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "search_alice")
	bob := createSecurityUser(t, "search_bob")
	charlie := createSecurityUser(t, "search_charlie")

	bobID := bob.ID
	aliceID := alice.ID
	matches := []*model.Message{
		{Type: model.MsgText, FromUserID: alice.ID, ToUserID: &bobID, Content: "needle from alice"},
		{Type: model.MsgText, FromUserID: bob.ID, ToUserID: &aliceID, Content: "reply with NEEDLE"},
	}
	for _, msg := range matches {
		if err := model.DB.Create(msg).Error; err != nil {
			t.Fatalf("create matching message: %v", err)
		}
	}

	if err := model.DB.Create(&model.Message{
		Type:       model.MsgText,
		FromUserID: charlie.ID,
		ToUserID:   &bobID,
		Content:    "needle from charlie",
	}).Error; err != nil {
		t.Fatalf("create out-of-scope message: %v", err)
	}

	got, total, err := chatSvc.SearchMessages(bob.ID, "user", alice.ID, "needle", 1, 20)
	if err != nil {
		t.Fatalf("search user messages: %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Fatalf("expected 2 scoped results, total=%d len=%d", total, len(got))
	}
	for _, msg := range got {
		if msg.FromUserID != alice.ID && msg.FromUserID != bob.ID {
			t.Fatalf("search returned out-of-scope sender %d", msg.FromUserID)
		}
	}
}

func TestSearchMessagesEscapesLikeWildcards(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "search_escape_alice")
	bob := createSecurityUser(t, "search_escape_bob")

	bobID := bob.ID
	if err := model.DB.Create(&model.Message{
		Type:       model.MsgText,
		FromUserID: alice.ID,
		ToUserID:   &bobID,
		Content:    "literal % match",
	}).Error; err != nil {
		t.Fatalf("create percent message: %v", err)
	}
	if err := model.DB.Create(&model.Message{
		Type:       model.MsgText,
		FromUserID: alice.ID,
		ToUserID:   &bobID,
		Content:    "ordinary message",
	}).Error; err != nil {
		t.Fatalf("create ordinary message: %v", err)
	}

	got, total, err := chatSvc.SearchMessages(bob.ID, "user", alice.ID, "%", 1, 20)
	if err != nil {
		t.Fatalf("search wildcard literal: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].Content != "literal % match" {
		t.Fatalf("expected only literal percent message, total=%d len=%d", total, len(got))
	}
}

func TestSearchMessagesGroupScopeRequiresMembership(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "search_group_owner")
	member := createSecurityUser(t, "search_group_member")
	outsider := createSecurityUser(t, "search_group_outsider")
	group, err := groupSvc.CreateGroup("search_group", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, member.ID); err != nil {
		t.Fatalf("join group: %v", err)
	}

	if err := model.DB.Create(&model.Message{
		Type:       model.MsgText,
		FromUserID: owner.ID,
		GroupID:    &group.ID,
		Content:    "group needle",
	}).Error; err != nil {
		t.Fatalf("create group message: %v", err)
	}

	got, total, err := chatSvc.SearchMessages(member.ID, "group", group.ID, "needle", 1, 20)
	if err != nil {
		t.Fatalf("search group messages: %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Fatalf("expected 1 group result, total=%d len=%d", total, len(got))
	}

	if _, _, err := chatSvc.SearchMessages(outsider.ID, "group", group.ID, "needle", 1, 20); err == nil {
		t.Fatal("expected non-member group search to be rejected")
	}
}

func TestSendFromClientAcceptsSameConversationQuote(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "quote_alice")
	bob := createSecurityUser(t, "quote_bob")
	createAcceptedFriendPair(t, alice.ID, bob.ID)

	bobID := bob.ID
	quote := &model.Message{Type: model.MsgText, FromUserID: alice.ID, ToUserID: &bobID, Content: "original quote"}
	if err := model.DB.Create(quote).Error; err != nil {
		t.Fatalf("create quote message: %v", err)
	}

	aliceID := alice.ID
	reply := &model.Message{
		Type:           model.MsgText,
		FromUserID:     bob.ID,
		ToUserID:       &aliceID,
		QuoteMessageID: &quote.ID,
		Content:        "reply with quote",
	}
	if err := chatSvc.SendFromClient(reply); err != nil {
		t.Fatalf("send quoted reply: %v", err)
	}
	if reply.QuoteMessage == nil || reply.QuoteMessage.ID != quote.ID {
		t.Fatalf("expected quote message to be preloaded, got %+v", reply.QuoteMessage)
	}
}

func TestSendFromClientRejectsCrossConversationQuote(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "quote_reject_alice")
	bob := createSecurityUser(t, "quote_reject_bob")
	charlie := createSecurityUser(t, "quote_reject_charlie")

	charlieID := charlie.ID
	quote := &model.Message{Type: model.MsgText, FromUserID: alice.ID, ToUserID: &charlieID, Content: "other conversation"}
	if err := model.DB.Create(quote).Error; err != nil {
		t.Fatalf("create quote message: %v", err)
	}

	bobID := bob.ID
	reply := &model.Message{
		Type:           model.MsgText,
		FromUserID:     alice.ID,
		ToUserID:       &bobID,
		QuoteMessageID: &quote.ID,
		Content:        "invalid quote",
	}
	if err := chatSvc.SendFromClient(reply); err == nil {
		t.Fatal("expected cross-conversation quote to be rejected")
	}
}

func TestSearchMessagesGlobalAndTimeRange(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "global_search_alice")
	bob := createSecurityUser(t, "global_search_bob")
	charlie := createSecurityUser(t, "global_search_charlie")
	group, err := groupSvc.CreateGroup("global_search_group", "", alice.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, bob.ID); err != nil {
		t.Fatalf("join group: %v", err)
	}

	base := time.Now().Add(time.Minute)
	bobID := bob.ID
	aliceID := alice.ID
	messages := []*model.Message{
		{Type: model.MsgText, FromUserID: alice.ID, ToUserID: &bobID, Content: "needle direct", CreatedAt: base.Add(10 * time.Minute)},
		{Type: model.MsgText, FromUserID: bob.ID, ToUserID: &aliceID, Content: "needle reply", CreatedAt: base.Add(20 * time.Minute)},
		{Type: model.MsgText, FromUserID: bob.ID, GroupID: &group.ID, Content: "needle group", CreatedAt: base.Add(30 * time.Minute)},
		{Type: model.MsgText, FromUserID: charlie.ID, ToUserID: &bobID, Content: "needle private outsider", CreatedAt: base.Add(40 * time.Minute)},
		{Type: model.MsgText, FromUserID: charlie.ID, Content: "needle broadcast", CreatedAt: base.Add(50 * time.Minute)},
		{Type: model.MsgText, FromUserID: alice.ID, ToUserID: &bobID, Content: "needle too old", CreatedAt: base.Add(-10 * time.Minute)},
	}
	for _, msg := range messages {
		if err := model.DB.Create(msg).Error; err != nil {
			t.Fatalf("create search message: %v", err)
		}
	}

	start := base
	end := base.Add(45 * time.Minute)
	got, total, err := chatSvc.SearchMessagesWithParams(alice.ID, MessageSearchParams{
		Scope:     "all",
		Keyword:   "needle",
		StartTime: &start,
		EndTime:   &end,
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("global search: %v", err)
	}
	if total != 3 || len(got) != 3 {
		t.Fatalf("expected 3 accessible in-range messages, total=%d len=%d", total, len(got))
	}
	for _, msg := range got {
		if msg.Content == "needle private outsider" || msg.Content == "needle too old" {
			t.Fatalf("search returned out-of-scope message: %q", msg.Content)
		}
	}
}

func TestOfflineSyncMessagesUseReadWatermarks(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	sender := createSecurityUser(t, "offline_sender")
	receiver := createSecurityUser(t, "offline_receiver")

	receiverID := receiver.ID
	first := &model.Message{Type: model.MsgText, FromUserID: sender.ID, ToUserID: &receiverID, Content: "already read"}
	second := &model.Message{Type: model.MsgText, FromUserID: sender.ID, ToUserID: &receiverID, Content: "offline unread"}
	if err := model.DB.Create(first).Error; err != nil {
		t.Fatalf("create first direct message: %v", err)
	}
	if err := model.DB.Create(second).Error; err != nil {
		t.Fatalf("create second direct message: %v", err)
	}

	peerID := sender.ID
	if err := chatSvc.saveReadProgress(receiver.ID, model.ConversationUser, sender.ID, &peerID, nil, first.ID); err != nil {
		t.Fatalf("save direct read progress: %v", err)
	}

	group, err := groupSvc.CreateGroup("offline_group", "", sender.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, receiver.ID); err != nil {
		t.Fatalf("join group: %v", err)
	}
	selfGroupMsg := &model.Message{Type: model.MsgText, FromUserID: receiver.ID, GroupID: &group.ID, Content: "self group"}
	groupMsg := &model.Message{Type: model.MsgText, FromUserID: sender.ID, GroupID: &group.ID, Content: "group offline"}
	if err := model.DB.Create(selfGroupMsg).Error; err != nil {
		t.Fatalf("create self group message: %v", err)
	}
	if err := model.DB.Create(groupMsg).Error; err != nil {
		t.Fatalf("create group message: %v", err)
	}

	msgs, err := chatSvc.getOfflineSyncMessages(receiver.ID)
	if err != nil {
		t.Fatalf("get offline sync messages: %v", err)
	}

	got := make(map[uint]bool)
	for _, msg := range msgs {
		got[msg.ID] = true
	}
	if got[first.ID] {
		t.Fatalf("already read direct message was included")
	}
	if got[selfGroupMsg.ID] {
		t.Fatalf("receiver's own group message was included")
	}
	if !got[second.ID] || !got[groupMsg.ID] {
		t.Fatalf("expected unread direct and group messages, got ids=%v", got)
	}
}
