package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"AIM/internal/model"
	"AIM/pkg/ai"
	appcrypto "AIM/pkg/crypto"
)

type fakeAIClient struct {
	reply string
	err   error
	usage ai.Usage
	calls chan ai.ChatRequest
}

type routedAIClient struct {
	calls chan routedAICall
}

type streamingAIClient struct {
	chunks         []string
	usage          ai.Usage
	err            error
	calls          chan ai.ChatRequest
	nonStreamCalls chan ai.ChatRequest
}

type routedAICall struct {
	req   ai.ChatRequest
	reply chan string
}

func (f *fakeAIClient) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	if f.calls != nil {
		f.calls <- req
	}
	if f.err != nil {
		return nil, f.err
	}
	return &ai.ChatResponse{Content: f.reply, Usage: f.usage}, nil
}

func (f *routedAIClient) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	call := routedAICall{req: req, reply: make(chan string, 1)}
	select {
	case f.calls <- call:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case reply := <-call.reply:
		return &ai.ChatResponse{Content: reply}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *streamingAIClient) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	if f.nonStreamCalls != nil {
		f.nonStreamCalls <- req
	}
	return nil, errors.New("non-stream chat should not be called")
}

func (f *streamingAIClient) ChatStream(ctx context.Context, req ai.ChatRequest, onChunk ai.StreamHandler) (*ai.ChatResponse, error) {
	if f.calls != nil {
		select {
		case f.calls <- req:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	var content strings.Builder
	for _, chunk := range f.chunks {
		content.WriteString(chunk)
		if onChunk != nil {
			if err := onChunk(ai.ChatStreamChunk{Delta: chunk}); err != nil {
				return nil, err
			}
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return &ai.ChatResponse{Content: content.String(), Usage: f.usage}, nil
}

func TestAIUserDirectMessageCreatesReply(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	user := createSecurityUser(t, "ai_direct_user")
	bot := createAIUser(t, "ai_direct_bot")

	fake := &fakeAIClient{reply: "AI reply", calls: make(chan ai.ChatRequest, 1)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		MaxContextMessages: 4,
		Timeout:            time.Second,
	})

	toBot := bot.ID
	source := &model.Message{
		Type:       model.MsgText,
		FromUserID: user.ID,
		ToUserID:   &toBot,
		Content:    "hello ai",
	}
	if err := chatSvc.SendFromClient(source); err != nil {
		t.Fatalf("send direct ai message: %v", err)
	}

	req := waitAIRequest(t, fake.calls)
	if req.Model != bot.AIModel {
		t.Fatalf("expected model %s, got %s", bot.AIModel, req.Model)
	}
	if req.BaseURL != bot.AIEndpoint {
		t.Fatalf("expected endpoint %s, got %s", bot.AIEndpoint, req.BaseURL)
	}
	if len(req.Messages) < 2 || req.Messages[len(req.Messages)-1].Content != "hello ai" {
		t.Fatalf("expected current user message in prompt, got %+v", req.Messages)
	}

	reply := waitMessageContent(t, bot.ID, "AI reply")
	if reply.ToUserID == nil || *reply.ToUserID != user.ID {
		t.Fatalf("expected direct reply to user %d, got %+v", user.ID, reply.ToUserID)
	}
	if reply.QuoteMessageID == nil || *reply.QuoteMessageID != source.ID {
		t.Fatalf("expected reply to quote source message %d, got %+v", source.ID, reply.QuoteMessageID)
	}
}

func TestAIUserDirectMessageStreamsReply(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_stream_owner")
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{Timeout: time.Second})
	bot, err := aiSvc.CreateUserBot(owner.ID, AIBotInput{
		Name:    "Stream AI",
		BaseURL: "http://stream-ai.local/v1",
		APIKey:  "stream-key",
		Model:   "stream-model",
	})
	if err != nil {
		t.Fatalf("create stream bot: %v", err)
	}

	fake := &streamingAIClient{
		chunks:         []string{"stream", " reply"},
		usage:          ai.Usage{PromptTokens: 4, CompletionTokens: 3, TotalTokens: 7},
		calls:          make(chan ai.ChatRequest, 1),
		nonStreamCalls: make(chan ai.ChatRequest, 1),
	}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{Timeout: time.Second})

	toBot := bot.UserID
	source := &model.Message{
		Type:       model.MsgText,
		FromUserID: owner.ID,
		ToUserID:   &toBot,
		Content:    "hello stream ai",
	}
	if err := chatSvc.SendFromClient(source); err != nil {
		t.Fatalf("send stream ai message: %v", err)
	}

	req := waitAIRequest(t, fake.calls)
	if req.APIKey != "stream-key" || req.Model != "stream-model" || req.BaseURL != "http://stream-ai.local/v1" {
		t.Fatalf("unexpected stream request: %+v", req)
	}
	assertNoAIRequest(t, fake.nonStreamCalls)

	reply := waitMessageContent(t, bot.UserID, "stream reply")
	if reply.QuoteMessageID == nil || *reply.QuoteMessageID != source.ID {
		t.Fatalf("expected stream reply to quote source message %d, got %+v", source.ID, reply.QuoteMessageID)
	}

	usage := waitTokenUsage(t, owner.ID, bot.ID)
	if usage.TotalTokens != 7 || usage.PromptTokens != 4 || usage.CompletionTokens != 3 {
		t.Fatalf("unexpected stream usage: %+v", usage)
	}
}

func TestAIDirectConcurrentRepliesStayWithSourceMessage(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	userA := createSecurityUser(t, "ai_concurrent_user_a")
	userB := createSecurityUser(t, "ai_concurrent_user_b")
	bot := createAIUser(t, "ai_concurrent_bot")

	fake := &routedAIClient{calls: make(chan routedAICall, 2)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		MaxContextMessages: 4,
		Timeout:            time.Second,
	})

	toBot := bot.ID
	msgA := &model.Message{
		Type:       model.MsgText,
		FromUserID: userA.ID,
		ToUserID:   &toBot,
		Content:    "question from user a",
	}
	msgB := &model.Message{
		Type:       model.MsgText,
		FromUserID: userB.ID,
		ToUserID:   &toBot,
		Content:    "question from user b",
	}
	if err := chatSvc.SendFromClient(msgA); err != nil {
		t.Fatalf("send user a ai message: %v", err)
	}
	if err := chatSvc.SendFromClient(msgB); err != nil {
		t.Fatalf("send user b ai message: %v", err)
	}

	calls := map[string]routedAICall{}
	for len(calls) < 2 {
		call := waitRoutedAICall(t, fake.calls)
		calls[lastAIUserMessage(call.req)] = call
	}

	callA, ok := calls[msgA.Content]
	if !ok {
		t.Fatalf("missing ai request for %q", msgA.Content)
	}
	callB, ok := calls[msgB.Content]
	if !ok {
		t.Fatalf("missing ai request for %q", msgB.Content)
	}

	// 故意让后发消息先返回，验证投递目标仍按源消息隔离。
	callB.reply <- "answer for user b"
	callA.reply <- "answer for user a"

	replyA := waitMessageContent(t, bot.ID, "answer for user a")
	if replyA.ToUserID == nil || *replyA.ToUserID != userA.ID {
		t.Fatalf("expected user a reply to user %d, got %+v", userA.ID, replyA.ToUserID)
	}
	if replyA.QuoteMessageID == nil || *replyA.QuoteMessageID != msgA.ID {
		t.Fatalf("expected user a reply to quote message %d, got %+v", msgA.ID, replyA.QuoteMessageID)
	}

	replyB := waitMessageContent(t, bot.ID, "answer for user b")
	if replyB.ToUserID == nil || *replyB.ToUserID != userB.ID {
		t.Fatalf("expected user b reply to user %d, got %+v", userB.ID, replyB.ToUserID)
	}
	if replyB.QuoteMessageID == nil || *replyB.QuoteMessageID != msgB.ID {
		t.Fatalf("expected user b reply to quote message %d, got %+v", msgB.ID, replyB.QuoteMessageID)
	}
}

func TestAIDirectContextIsIsolatedPerUser(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	userA := createSecurityUser(t, "ai_context_user_a")
	userB := createSecurityUser(t, "ai_context_user_b")
	bot := createAIUser(t, "ai_context_bot")

	toBot := bot.ID
	toUserA := userA.ID
	toUserB := userB.ID
	history := []*model.Message{
		{Type: model.MsgText, FromUserID: userA.ID, ToUserID: &toBot, Content: "user-a-private-context"},
		{Type: model.MsgText, FromUserID: bot.ID, ToUserID: &toUserA, Content: "assistant-a-private-context"},
		{Type: model.MsgText, FromUserID: userB.ID, ToUserID: &toBot, Content: "user-b-private-context"},
		{Type: model.MsgText, FromUserID: bot.ID, ToUserID: &toUserB, Content: "assistant-b-private-context"},
	}
	for _, msg := range history {
		if err := model.DB.Create(msg).Error; err != nil {
			t.Fatalf("seed ai direct history: %v", err)
		}
	}

	fake := &fakeAIClient{reply: "isolated reply", calls: make(chan ai.ChatRequest, 2)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		MaxContextMessages: 8,
		Timeout:            time.Second,
	})

	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: userA.ID,
		ToUserID:   &toBot,
		Content:    "current question from user a",
	}); err != nil {
		t.Fatalf("send user a ai message: %v", err)
	}
	reqA := waitAIRequest(t, fake.calls)
	if !promptContains(reqA, "user-a-private-context") || !promptContains(reqA, "assistant-a-private-context") {
		t.Fatalf("expected user a prompt to include only user a history, got %+v", reqA.Messages)
	}
	if promptContains(reqA, "user-b-private-context") || promptContains(reqA, "assistant-b-private-context") {
		t.Fatalf("expected user a prompt to exclude user b history, got %+v", reqA.Messages)
	}

	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: userB.ID,
		ToUserID:   &toBot,
		Content:    "current question from user b",
	}); err != nil {
		t.Fatalf("send user b ai message: %v", err)
	}
	reqB := waitAIRequest(t, fake.calls)
	if !promptContains(reqB, "user-b-private-context") || !promptContains(reqB, "assistant-b-private-context") {
		t.Fatalf("expected user b prompt to include only user b history, got %+v", reqB.Messages)
	}
	if promptContains(reqB, "user-a-private-context") || promptContains(reqB, "assistant-a-private-context") {
		t.Fatalf("expected user b prompt to exclude user a history, got %+v", reqB.Messages)
	}
}

func TestAIUserGroupMentionCreatesReply(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_group_owner")
	member := createSecurityUser(t, "ai_group_member")
	bot := createAIUser(t, "ai_group_bot")

	group, err := groupSvc.CreateGroup("ai_group", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, member.ID); err != nil {
		t.Fatalf("join member: %v", err)
	}
	if err := groupSvc.AddMember(group.ID, owner.ID, bot.ID); err != nil {
		t.Fatalf("join bot: %v", err)
	}

	fake := &fakeAIClient{reply: "group AI reply", calls: make(chan ai.ChatRequest, 1)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		MaxContextMessages: 4,
		Timeout:            time.Second,
	})

	mentions := fmt.Sprintf("[%d]", bot.ID)
	source := &model.Message{
		Type:       model.MsgText,
		FromUserID: member.ID,
		GroupID:    &group.ID,
		Content:    "@ai_group_bot hello",
		Mentions:   mentions,
	}
	if err := chatSvc.SendFromClient(source); err != nil {
		t.Fatalf("send group ai mention: %v", err)
	}

	req := waitAIRequest(t, fake.calls)
	if !strings.Contains(req.Messages[0].Content, "群聊场景") {
		t.Fatalf("expected group system prompt, got %q", req.Messages[0].Content)
	}

	reply := waitMessageContent(t, bot.ID, "group AI reply")
	if reply.GroupID == nil || *reply.GroupID != group.ID {
		t.Fatalf("expected group reply to group %d, got %+v", group.ID, reply.GroupID)
	}
	if !strings.Contains(reply.Mentions, fmt.Sprint(member.ID)) {
		t.Fatalf("expected reply mentions sender %d, got %q", member.ID, reply.Mentions)
	}
	if reply.QuoteMessageID == nil || *reply.QuoteMessageID != source.ID {
		t.Fatalf("expected group reply to quote source message %d, got %+v", source.ID, reply.QuoteMessageID)
	}
}

func TestAIUserGroupContextOnlyUsesCurrentMentionThread(t *testing.T) {
	chatSvc, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_group_context_owner")
	member := createSecurityUser(t, "ai_group_context_member")
	other := createSecurityUser(t, "ai_group_context_other")
	bot := createAIUser(t, "ai_group_context_bot")

	group, err := groupSvc.CreateGroup("ai_group_context", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	for _, uid := range []uint{member.ID, other.ID, bot.ID} {
		if err := groupSvc.JoinGroup(group.ID, uid); err != nil {
			t.Fatalf("join group member %d: %v", uid, err)
		}
	}

	seed := []*model.Message{
		{Type: model.MsgText, FromUserID: other.ID, GroupID: &group.ID, Content: "@ai_group_context_bot other secret", Mentions: fmt.Sprintf("[%d]", bot.ID)},
		{Type: model.MsgText, FromUserID: member.ID, GroupID: &group.ID, Content: "member unrelated group chatter"},
		{Type: model.MsgText, FromUserID: member.ID, GroupID: &group.ID, Content: "@ai_group_context_bot prior question", Mentions: fmt.Sprintf("[%d]", bot.ID)},
		{Type: model.MsgText, FromUserID: bot.ID, GroupID: &group.ID, Content: "prior answer to member", Mentions: fmt.Sprintf("[%d]", member.ID)},
		{Type: model.MsgText, FromUserID: bot.ID, GroupID: &group.ID, Content: "answer to other", Mentions: fmt.Sprintf("[%d]", other.ID)},
	}
	for _, msg := range seed {
		if err := model.DB.Create(msg).Error; err != nil {
			t.Fatalf("seed group ai context: %v", err)
		}
	}

	fake := &fakeAIClient{reply: "isolated group reply", calls: make(chan ai.ChatRequest, 1)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		MaxContextMessages: 8,
		Timeout:            time.Second,
	})

	source := &model.Message{
		Type:       model.MsgText,
		FromUserID: member.ID,
		GroupID:    &group.ID,
		Content:    "@ai_group_context_bot current question",
		Mentions:   fmt.Sprintf("[%d]", bot.ID),
	}
	if err := chatSvc.SendFromClient(source); err != nil {
		t.Fatalf("send group ai mention: %v", err)
	}

	req := waitAIRequest(t, fake.calls)
	if !promptContains(req, "prior question") || !promptContains(req, "prior answer to member") || !promptContains(req, "current question") {
		t.Fatalf("expected current member ai thread in prompt, got %+v", req.Messages)
	}
	for _, leaked := range []string{"other secret", "member unrelated group chatter", "answer to other"} {
		if promptContains(req, leaked) {
			t.Fatalf("group ai context leaked %q into prompt: %+v", leaked, req.Messages)
		}
	}
}

func TestNormalUserMessageDoesNotTriggerAI(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "normal_trigger_alice")
	bob := createSecurityUser(t, "normal_trigger_bob")
	createAcceptedFriendPair(t, alice.ID, bob.ID)

	fake := &fakeAIClient{reply: "should not call", calls: make(chan ai.ChatRequest, 1)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{Timeout: time.Second})

	toBob := bob.ID
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: alice.ID,
		ToUserID:   &toBob,
		Content:    "hello bob",
	}); err != nil {
		t.Fatalf("send normal message: %v", err)
	}

	assertNoAIRequest(t, fake.calls)
}

func TestAIProviderFailureSendsFallbackMessage(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	user := createSecurityUser(t, "ai_fail_user")
	bot := createAIUser(t, "ai_fail_bot")

	fake := &fakeAIClient{err: errors.New("provider down"), calls: make(chan ai.ChatRequest, 1)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		MaxContextMessages: 4,
		Timeout:            time.Second,
	})

	toBot := bot.ID
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: user.ID,
		ToUserID:   &toBot,
		Content:    "hello ai",
	}); err != nil {
		t.Fatalf("send direct ai message: %v", err)
	}

	waitAIRequest(t, fake.calls)
	waitMessageContent(t, bot.ID, "AI 暂时无法回复，请稍后再试。")
}

func TestAIContextExcludesRecalledMessages(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	user := createSecurityUser(t, "ai_recall_context_user")
	bot := createAIUser(t, "ai_recall_context_bot")

	toBot := bot.ID
	history := []*model.Message{
		{Type: model.MsgText, FromUserID: user.ID, ToUserID: &toBot, Content: "visible-context"},
		{Type: model.MsgText, FromUserID: user.ID, ToUserID: &toBot, Content: "recalled-secret", IsRecalled: true},
	}
	for _, msg := range history {
		if err := model.DB.Create(msg).Error; err != nil {
			t.Fatalf("seed ai recall context: %v", err)
		}
	}

	fake := &fakeAIClient{reply: "recall-safe reply", calls: make(chan ai.ChatRequest, 1)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		MaxContextMessages: 8,
		Timeout:            time.Second,
	})

	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: user.ID,
		ToUserID:   &toBot,
		Content:    "current recall-safe question",
	}); err != nil {
		t.Fatalf("send ai recall context message: %v", err)
	}

	req := waitAIRequest(t, fake.calls)
	if !promptContains(req, "visible-context") {
		t.Fatalf("expected visible context in prompt, got %+v", req.Messages)
	}
	if promptContains(req, "recalled-secret") {
		t.Fatalf("recalled message leaked into prompt: %+v", req.Messages)
	}
}

func TestAIUserCannotLogin(t *testing.T) {
	setupChatSecurityTest(t)
	createAIUser(t, "ai_login_bot")

	authSvc := &AuthService{JWTSecret: "test", JWTExpireHrs: 1}
	if _, _, err := authSvc.Login("ai_login_bot", "anything"); err == nil {
		t.Fatal("expected ai user login to be rejected")
	}
}

func TestEnsureDefaultBotCreatesAIUser(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	existingUser := createSecurityUser(t, "default_ai_friend_user")
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{
		BaseURL:            "http://ai.local/v1",
		DefaultModel:       "test-model",
		DefaultBotUsername: "default_ai",
		DefaultBotNickname: "默认AI",
		SystemPrompt:       "system prompt",
	})

	bot, err := aiSvc.EnsureDefaultBot()
	if err != nil {
		t.Fatalf("ensure default bot: %v", err)
	}
	if !bot.IsAI || bot.Username != "default_ai" || bot.AIModel != "test-model" || bot.AIEndpoint != "http://ai.local/v1" {
		t.Fatalf("unexpected bot: %+v", bot)
	}

	var cfg model.AIBot
	if err := model.DB.Where("user_id = ? AND is_system = ?", bot.ID, true).First(&cfg).Error; err != nil {
		t.Fatalf("expected system ai bot config: %v", err)
	}
	if cfg.OwnerID != nil || cfg.Model != "test-model" {
		t.Fatalf("unexpected system config: %+v", cfg)
	}
	assertAcceptedFriendPair(t, existingUser.ID, bot.ID)
}

func TestEnsureDefaultBotAllowsEmptyProviderConfig(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{
		DefaultBotUsername: "blue_ai",
		DefaultBotNickname: "蓝妹",
	})

	bot, err := aiSvc.EnsureDefaultBot()
	if err != nil {
		t.Fatalf("ensure default bot without provider config: %v", err)
	}
	if !bot.IsAI || bot.Nickname != "蓝妹" {
		t.Fatalf("unexpected bot: %+v", bot)
	}

	var cfg model.AIBot
	if err := model.DB.Where("user_id = ? AND is_system = ?", bot.ID, true).First(&cfg).Error; err != nil {
		t.Fatalf("expected system ai bot config: %v", err)
	}
	if cfg.BaseURL != "" || cfg.Model != "" {
		t.Fatalf("expected empty provider config, got %+v", cfg)
	}
}

func TestUnconfiguredDefaultBotRepliesWithSetupHint(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	user := createSecurityUser(t, "unconfigured_ai_user")
	fake := &fakeAIClient{reply: "should not call", calls: make(chan ai.ChatRequest, 1)}
	aiSvc := NewAIService(fake, chatSvc, AIConfig{
		DefaultBotUsername: "unconfigured_ai",
		DefaultBotNickname: "蓝妹",
		Timeout:            time.Second,
	})
	bot, err := aiSvc.EnsureDefaultBot()
	if err != nil {
		t.Fatalf("ensure default bot: %v", err)
	}
	chatSvc.AIResponder = aiSvc

	toBot := bot.ID
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: user.ID,
		ToUserID:   &toBot,
		Content:    "你好",
	}); err != nil {
		t.Fatalf("send unconfigured ai message: %v", err)
	}

	assertNoAIRequest(t, fake.calls)
	waitMessageContent(t, bot.ID, "AI 尚未配置，请在后端补充 API 地址和模型后再使用。")
}

func TestRegisterAddsSystemAIBotFriend(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{
		BaseURL:            "http://ai.local/v1",
		DefaultModel:       "test-model",
		DefaultBotUsername: "register_ai",
		DefaultBotNickname: "注册AI",
	})

	bot, err := aiSvc.EnsureDefaultBot()
	if err != nil {
		t.Fatalf("ensure default bot: %v", err)
	}

	authSvc := &AuthService{JWTSecret: "test", JWTExpireHrs: 1}
	user, err := authSvc.Register("register_with_ai", "password123", "新用户")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	assertAcceptedFriendPair(t, user.ID, bot.ID)
}

func TestUserAIBotCRUDDoesNotExposeAPIKey(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_crud_owner")
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{})

	created, err := aiSvc.CreateUserBot(owner.ID, AIBotInput{
		Name:         "My AI",
		BaseURL:      "http://user-ai.local/v1",
		APIKey:       "secret-key",
		Model:        "user-model",
		SystemPrompt: "answer as helper",
	})
	if err != nil {
		t.Fatalf("create user bot: %v", err)
	}
	if !created.CanEdit || !created.APIKeySet || created.IsSystem {
		t.Fatalf("unexpected created info: %+v", created)
	}
	if !created.Ready || created.UnavailableReason != "" {
		t.Fatalf("created bot should be ready, got %+v", created)
	}

	bots, err := aiSvc.ListUserBots(owner.ID)
	if err != nil {
		t.Fatalf("list user bots: %v", err)
	}
	if len(bots) != 1 || bots[0].UserID != created.UserID || !bots[0].APIKeySet {
		t.Fatalf("unexpected bot list: %+v", bots)
	}

	newName := "Renamed AI"
	newKey := "new-secret"
	updated, err := aiSvc.UpdateUserBot(owner.ID, created.ID, AIBotUpdateInput{
		Name:   &newName,
		APIKey: &newKey,
	})
	if err != nil {
		t.Fatalf("update user bot: %v", err)
	}
	if updated.Nickname != newName || !updated.APIKeySet {
		t.Fatalf("unexpected updated info: %+v", updated)
	}

	if err := aiSvc.DeleteUserBot(owner.ID, created.ID); err != nil {
		t.Fatalf("delete user bot: %v", err)
	}
	bots, err = aiSvc.ListUserBots(owner.ID)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(bots) != 0 {
		t.Fatalf("deleted bot should be hidden, got %+v", bots)
	}
}

func TestAIBotInfoReportsUnavailableConfig(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_unavailable_owner")
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{})
	bot, err := aiSvc.CreateUserBot(owner.ID, AIBotInput{
		Name:      "Missing Config",
		APISource: model.AIBotAPISourceSystem,
	})
	if err != nil {
		t.Fatalf("create unavailable bot: %v", err)
	}
	if bot.Ready || bot.UnavailableReason == "" {
		t.Fatalf("expected unavailable bot info, got %+v", bot)
	}
}

func TestUserAIBotUsesOwnerAPIKey(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_owner_key")
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{Timeout: time.Second})
	bot, err := aiSvc.CreateUserBot(owner.ID, AIBotInput{
		Name:    "Owner AI",
		BaseURL: "http://owner-ai.local/v1",
		APIKey:  "owner-key",
		Model:   "owner-model",
	})
	if err != nil {
		t.Fatalf("create owner bot: %v", err)
	}

	fake := &fakeAIClient{
		reply: "owner reply",
		usage: ai.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
		calls: make(chan ai.ChatRequest, 1),
	}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{Timeout: time.Second})
	toBot := bot.UserID
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: owner.ID,
		ToUserID:   &toBot,
		Content:    "hello own ai",
	}); err != nil {
		t.Fatalf("send owner ai message: %v", err)
	}

	req := waitAIRequest(t, fake.calls)
	if req.APIKey != "owner-key" || req.Model != "owner-model" || req.BaseURL != "http://owner-ai.local/v1" {
		t.Fatalf("unexpected owner ai request: %+v", req)
	}

	usage := waitTokenUsage(t, owner.ID, bot.ID)
	if usage.Billable || usage.TotalCostCNY != 0 || usage.TotalTokens != 150 {
		t.Fatalf("third-party usage should only track tokens, got %+v", usage)
	}
}

func TestUserAIBotEncryptsAPIKeyAtRest(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_encrypt_owner")
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{
		APIKeyEncryptKey: "test-storage-secret",
		Timeout:          time.Second,
	})
	bot, err := aiSvc.CreateUserBot(owner.ID, AIBotInput{
		Name:    "Encrypted AI",
		BaseURL: "http://encrypted-ai.local/v1",
		APIKey:  "plain-owner-key",
		Model:   "encrypted-model",
	})
	if err != nil {
		t.Fatalf("create encrypted bot: %v", err)
	}

	var stored model.AIBot
	if err := model.DB.First(&stored, bot.ID).Error; err != nil {
		t.Fatalf("load stored bot: %v", err)
	}
	if stored.APIKey == "plain-owner-key" || !appcrypto.IsEncryptedString(stored.APIKey) {
		t.Fatalf("expected encrypted api key at rest, got %q", stored.APIKey)
	}

	fake := &fakeAIClient{reply: "encrypted reply", calls: make(chan ai.ChatRequest, 1)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		APIKeyEncryptKey: "test-storage-secret",
		Timeout:          time.Second,
	})
	toBot := bot.UserID
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: owner.ID,
		ToUserID:   &toBot,
		Content:    "hello encrypted ai",
	}); err != nil {
		t.Fatalf("send encrypted ai message: %v", err)
	}

	req := waitAIRequest(t, fake.calls)
	if req.APIKey != "plain-owner-key" {
		t.Fatalf("expected decrypted api key, got %q", req.APIKey)
	}
}

func TestLegacyPlaintextAIBotAPIKeyIsMigratedOnUse(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_legacy_key_owner")
	botUser := createAIUser(t, "ai_legacy_key_bot")
	ownerID := owner.ID
	if err := model.DB.Create(&model.AIBot{
		UserID:    botUser.ID,
		OwnerID:   &ownerID,
		BaseURL:   "http://legacy-ai.local/v1",
		APIKey:    "legacy-plain-key",
		Model:     "legacy-model",
		Status:    model.AIBotStatusEnabled,
		APISource: model.AIBotAPISourceThirdParty,
	}).Error; err != nil {
		t.Fatalf("create legacy ai bot config: %v", err)
	}

	fake := &fakeAIClient{reply: "legacy reply", calls: make(chan ai.ChatRequest, 1)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		APIKeyEncryptKey: "test-storage-secret",
		Timeout:          time.Second,
	})
	toBot := botUser.ID
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: owner.ID,
		ToUserID:   &toBot,
		Content:    "hello legacy ai",
	}); err != nil {
		t.Fatalf("send legacy ai message: %v", err)
	}

	req := waitAIRequest(t, fake.calls)
	if req.APIKey != "legacy-plain-key" {
		t.Fatalf("expected legacy api key to decrypt as plaintext, got %q", req.APIKey)
	}
	var stored model.AIBot
	if err := model.DB.Where("user_id = ?", botUser.ID).First(&stored).Error; err != nil {
		t.Fatalf("load migrated bot: %v", err)
	}
	if stored.APIKey == "legacy-plain-key" || !appcrypto.IsEncryptedString(stored.APIKey) {
		t.Fatalf("expected legacy api key to be migrated, got %q", stored.APIKey)
	}
}

func TestSystemAPICustomBotBillsTokensAndUsesKnowledge(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_system_bill_owner")
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{
		BaseURL:      "http://system-ai.local/v1",
		APIKey:       "system-key",
		DefaultModel: "system-model",
		Timeout:      time.Second,
	})

	kb, err := aiSvc.CreateKnowledgeBase(owner.ID, AIKnowledgeBaseInput{Name: "蓝妹知识"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}
	if _, err := aiSvc.AddKnowledgeDocument(owner.ID, kb.ID, AIKnowledgeDocumentInput{
		Title:   "称呼",
		Content: "蓝妹应该称呼用户为掌柜的。",
	}); err != nil {
		t.Fatalf("add knowledge document: %v", err)
	}

	bot, err := aiSvc.CreateUserBot(owner.ID, AIBotInput{
		Name:             "系统分身",
		APISource:        model.AIBotAPISourceSystem,
		SystemPrompt:     "你是蓝妹分身。",
		KnowledgeBaseIDs: []uint{kb.ID},
	})
	if err != nil {
		t.Fatalf("create system api bot: %v", err)
	}

	fake := &fakeAIClient{
		reply: "system reply",
		usage: ai.Usage{PromptTokens: 1000, CompletionTokens: 2000, TotalTokens: 3000},
		calls: make(chan ai.ChatRequest, 1),
	}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{
		BaseURL:      "http://system-ai.local/v1",
		APIKey:       "system-key",
		DefaultModel: "system-model",
		Timeout:      time.Second,
	})

	toBot := bot.UserID
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: owner.ID,
		ToUserID:   &toBot,
		Content:    "hello system bot",
	}); err != nil {
		t.Fatalf("send system bot message: %v", err)
	}

	req := waitAIRequest(t, fake.calls)
	if req.BaseURL != "http://system-ai.local/v1" || req.APIKey != "system-key" || req.Model != "system-model" {
		t.Fatalf("unexpected system api request: %+v", req)
	}
	if !strings.Contains(req.Messages[0].Content, "蓝妹应该称呼用户为掌柜的") {
		t.Fatalf("expected knowledge content in system prompt, got %q", req.Messages[0].Content)
	}

	usage := waitTokenUsage(t, owner.ID, bot.ID)
	if !usage.Billable || usage.TotalTokens != 3000 {
		t.Fatalf("expected billable system usage, got %+v", usage)
	}
	if usage.TotalCostCNY != 0.007 {
		t.Fatalf("expected cost 0.007, got %.6f", usage.TotalCostCNY)
	}
}

func TestOtherUserCannotDirectTriggerOwnedAIBot(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_owner_private")
	other := createSecurityUser(t, "ai_other_private")
	aiSvc := NewAIService(&fakeAIClient{}, chatSvc, AIConfig{Timeout: time.Second})
	bot, err := aiSvc.CreateUserBot(owner.ID, AIBotInput{
		Name:    "Private AI",
		BaseURL: "http://private-ai.local/v1",
		APIKey:  "private-key",
		Model:   "private-model",
	})
	if err != nil {
		t.Fatalf("create private bot: %v", err)
	}

	fake := &fakeAIClient{reply: "should not call", calls: make(chan ai.ChatRequest, 1)}
	chatSvc.AIResponder = NewAIService(fake, chatSvc, AIConfig{Timeout: time.Second})
	toBot := bot.UserID
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: other.ID,
		ToUserID:   &toBot,
		Content:    "use other api",
	}); err == nil {
		t.Fatal("expected other user direct message to private ai to be rejected")
	}
	assertNoAIRequest(t, fake.calls)
}

func TestDeletedAIBotRejectsDirectMessage(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "ai_deleted_owner")
	botUser := createAIUser(t, "ai_deleted_bot")
	ownerID := owner.ID
	if err := model.DB.Create(&model.AIBot{
		UserID:    botUser.ID,
		OwnerID:   &ownerID,
		BaseURL:   "http://deleted-ai.local/v1",
		Model:     "deleted-model",
		Status:    model.AIBotStatusDeleted,
		APISource: model.AIBotAPISourceThirdParty,
	}).Error; err != nil {
		t.Fatalf("create deleted ai bot config: %v", err)
	}

	toBot := botUser.ID
	err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: owner.ID,
		ToUserID:   &toBot,
		Content:    "message to deleted bot",
	})
	if err == nil || !strings.Contains(err.Error(), "已删除") {
		t.Fatalf("expected deleted bot send rejection, got %v", err)
	}
}

func createAIUser(t *testing.T, username string) *model.User {
	t.Helper()

	user := &model.User{
		Username:       username,
		Password:       disabledAIPassword,
		Nickname:       username,
		IsAI:           true,
		AIModel:        "test-model",
		AIEndpoint:     "http://ai.local/v1",
		AISystemPrompt: "system prompt",
	}
	if err := model.DB.Create(user).Error; err != nil {
		t.Fatalf("create ai user %s: %v", username, err)
	}
	return user
}

func waitAIRequest(t *testing.T, calls <-chan ai.ChatRequest) ai.ChatRequest {
	t.Helper()

	select {
	case req := <-calls:
		return req
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ai request")
	}
	return ai.ChatRequest{}
}

func waitRoutedAICall(t *testing.T, calls <-chan routedAICall) routedAICall {
	t.Helper()

	select {
	case call := <-calls:
		return call
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for routed ai request")
	}
	return routedAICall{}
}

func lastAIUserMessage(req ai.ChatRequest) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			return req.Messages[i].Content
		}
	}
	return ""
}

func promptContains(req ai.ChatRequest, text string) bool {
	for _, msg := range req.Messages {
		if strings.Contains(msg.Content, text) {
			return true
		}
	}
	return false
}

func assertNoAIRequest(t *testing.T, calls <-chan ai.ChatRequest) {
	t.Helper()

	select {
	case req := <-calls:
		t.Fatalf("unexpected ai request: %+v", req)
	case <-time.After(150 * time.Millisecond):
	}
}

func waitMessageContent(t *testing.T, fromUserID uint, content string) *model.Message {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var msg model.Message
		err := model.DB.Where("from_user_id = ? AND content = ?", fromUserID, content).
			First(&msg).Error
		if err == nil {
			return &msg
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for message from %d with content %q", fromUserID, content)
	return nil
}

func waitTokenUsage(t *testing.T, userID, botID uint) *model.AITokenUsage {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var usage model.AITokenUsage
		err := model.DB.Where("user_id = ? AND bot_id = ?", userID, botID).
			First(&usage).Error
		if err == nil {
			return &usage
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for token usage user=%d bot=%d", userID, botID)
	return nil
}

func assertAcceptedFriendPair(t *testing.T, userID, friendID uint) {
	t.Helper()

	var count int64
	if err := model.DB.Model(&model.FriendRelation{}).
		Where("((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)) AND status = ?",
			userID, friendID, friendID, userID, "accepted").
		Count(&count).Error; err != nil {
		t.Fatalf("count ai friend pair: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected accepted friend pair between %d and %d, got %d rows", userID, friendID, count)
	}
}
