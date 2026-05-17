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
)

type fakeAIClient struct {
	reply string
	err   error
	calls chan ai.ChatRequest
}

func (f *fakeAIClient) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	if f.calls != nil {
		f.calls <- req
	}
	if f.err != nil {
		return nil, f.err
	}
	return &ai.ChatResponse{Content: f.reply}, nil
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
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: user.ID,
		ToUserID:   &toBot,
		Content:    "hello ai",
	}); err != nil {
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
	if err := chatSvc.SendFromClient(&model.Message{
		Type:       model.MsgText,
		FromUserID: member.ID,
		GroupID:    &group.ID,
		Content:    "@ai_group_bot hello",
		Mentions:   mentions,
	}); err != nil {
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
}

func TestNormalUserMessageDoesNotTriggerAI(t *testing.T) {
	chatSvc, _ := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "normal_trigger_alice")
	bob := createSecurityUser(t, "normal_trigger_bob")

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
