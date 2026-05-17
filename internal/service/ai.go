package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"AIM/internal/model"
	"AIM/pkg/ai"

	"gorm.io/gorm"
)

const disabledAIPassword = "AI_USER_DISABLED_LOGIN"

// AIConfig 是 service 层使用的 AI 运行配置。
type AIConfig struct {
	BaseURL            string
	APIKey             string
	DefaultModel       string
	DefaultBotUsername string
	DefaultBotNickname string
	SystemPrompt       string
	Timeout            time.Duration
	MaxContextMessages int
	Temperature        float64
	MaxTokens          int
}

// AIService 负责识别 AI 用户、组装上下文并以 AI 用户身份写回消息。
type AIService struct {
	Client ai.Client
	Chat   *ChatService
	Config AIConfig
}

type aiTrigger struct {
	Bot       model.User
	AtMention bool
}

func NewAIService(client ai.Client, chat *ChatService, cfg AIConfig) *AIService {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxContextMessages <= 0 {
		cfg.MaxContextMessages = 12
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 1024
	}
	return &AIService{Client: client, Chat: chat, Config: cfg}
}

// EnsureDefaultBot 创建或更新默认 AI 虚拟用户。
func (s *AIService) EnsureDefaultBot() (*model.User, error) {
	username := strings.TrimSpace(s.Config.DefaultBotUsername)
	if username == "" {
		return nil, errors.New("default ai bot username is empty")
	}

	var user model.User
	err := model.DB.Where("username = ?", username).First(&user).Error
	if err == nil {
		if !user.IsAI {
			return nil, fmt.Errorf("username %s already belongs to a normal user", username)
		}
		updates := s.defaultBotUpdates()
		if err := model.DB.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
		if err := model.DB.First(&user, user.ID).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	user = model.User{
		Username:       username,
		Password:       disabledAIPassword,
		Nickname:       firstNonEmpty(s.Config.DefaultBotNickname, username),
		IsAI:           true,
		AIModel:        strings.TrimSpace(s.Config.DefaultModel),
		AISystemPrompt: strings.TrimSpace(s.Config.SystemPrompt),
		AIEndpoint:     strings.TrimSpace(s.Config.BaseURL),
	}
	if err := model.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// HandleMessage 在普通消息入库后异步触发 AI 回复。
func (s *AIService) HandleMessage(msg *model.Message) {
	if s == nil || s.Client == nil || s.Chat == nil || msg == nil {
		return
	}
	if msg.Type != model.MsgText || strings.TrimSpace(msg.Content) == "" {
		return
	}

	triggers, err := s.resolveTriggers(msg)
	if err != nil {
		log.Printf("[ai] resolve trigger failed: %v", err)
		return
	}
	for _, trigger := range triggers {
		reply, err := s.generateReply(context.Background(), trigger.Bot, msg)
		if err != nil {
			log.Printf("[ai] generate reply failed: bot=%d msg=%d err=%v", trigger.Bot.ID, msg.ID, err)
			reply = "AI 暂时无法回复，请稍后再试。"
		}
		if err := s.sendReply(trigger.Bot, msg, reply); err != nil {
			log.Printf("[ai] send reply failed: bot=%d msg=%d err=%v", trigger.Bot.ID, msg.ID, err)
		}
	}
}

func (s *AIService) defaultBotUpdates() map[string]interface{} {
	updates := map[string]interface{}{
		"is_ai": true,
	}
	if nickname := strings.TrimSpace(s.Config.DefaultBotNickname); nickname != "" {
		updates["nickname"] = nickname
	}
	if modelName := strings.TrimSpace(s.Config.DefaultModel); modelName != "" {
		updates["ai_model"] = modelName
	}
	if prompt := strings.TrimSpace(s.Config.SystemPrompt); prompt != "" {
		updates["ai_system_prompt"] = prompt
	}
	if endpoint := strings.TrimSpace(s.Config.BaseURL); endpoint != "" {
		updates["ai_endpoint"] = endpoint
	}
	return updates
}

func (s *AIService) resolveTriggers(msg *model.Message) ([]aiTrigger, error) {
	var sender model.User
	if err := model.DB.First(&sender, msg.FromUserID).Error; err != nil {
		return nil, err
	}
	if sender.IsAI {
		return nil, nil
	}

	switch {
	case msg.IsToUser():
		var bot model.User
		if err := model.DB.First(&bot, *msg.ToUserID).Error; err != nil {
			return nil, err
		}
		if !bot.IsAI {
			return nil, nil
		}
		return []aiTrigger{{Bot: bot}}, nil
	case msg.IsToGroup():
		return s.resolveGroupTriggers(msg)
	default:
		return nil, nil
	}
}

func (s *AIService) resolveGroupTriggers(msg *model.Message) ([]aiTrigger, error) {
	if msg.GroupID == nil {
		return nil, nil
	}

	bots, err := s.groupMentionedBots(*msg.GroupID, msg.Mentions, msg.Content)
	if err != nil {
		return nil, err
	}
	triggers := make([]aiTrigger, 0, len(bots))
	for _, bot := range bots {
		if bot.ID == msg.FromUserID || !s.isGroupMember(*msg.GroupID, bot.ID) {
			continue
		}
		triggers = append(triggers, aiTrigger{Bot: bot, AtMention: true})
	}
	return triggers, nil
}

func (s *AIService) groupMentionedBots(groupID uint, rawMentions, content string) ([]model.User, error) {
	mentionIDs := parseMentionIDs(rawMentions)
	if len(mentionIDs) > 0 {
		var bots []model.User
		err := model.DB.Where("id IN ? AND is_ai = ?", mentionIDs, true).Find(&bots).Error
		return bots, err
	}
	return s.groupBotsMentionedByName(groupID, content)
}

func (s *AIService) groupBotsMentionedByName(groupID uint, content string) ([]model.User, error) {
	var memberIDs []uint
	if err := model.DB.Model(&model.GroupMember{}).
		Where("group_id = ?", groupID).
		Pluck("user_id", &memberIDs).Error; err != nil {
		return nil, err
	}
	if len(memberIDs) == 0 {
		return nil, nil
	}

	var bots []model.User
	if err := model.DB.Where("id IN ? AND is_ai = ?", memberIDs, true).Find(&bots).Error; err != nil {
		return nil, err
	}

	matched := make([]model.User, 0, len(bots))
	for _, bot := range bots {
		if containsMention(content, bot.Username) || containsMention(content, bot.Nickname) {
			matched = append(matched, bot)
		}
	}
	return matched, nil
}

func (s *AIService) isGroupMember(groupID, userID uint) bool {
	var count int64
	model.DB.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count)
	return count > 0
}

func (s *AIService) generateReply(parent context.Context, bot model.User, source *model.Message) (string, error) {
	baseURL := firstNonEmpty(bot.AIEndpoint, s.Config.BaseURL)
	modelName := firstNonEmpty(bot.AIModel, s.Config.DefaultModel)
	if baseURL == "" {
		return "", errors.New("ai endpoint is empty")
	}
	if modelName == "" {
		return "", errors.New("ai model is empty")
	}

	messages, err := s.buildPrompt(bot, source)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(parent, s.Config.Timeout)
	defer cancel()

	temperature := s.Config.Temperature
	resp, err := s.Client.Chat(ctx, ai.ChatRequest{
		BaseURL:     baseURL,
		APIKey:      s.Config.APIKey,
		Model:       modelName,
		Messages:    messages,
		Temperature: &temperature,
		MaxTokens:   s.Config.MaxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (s *AIService) buildPrompt(bot model.User, source *model.Message) ([]ai.ChatMessage, error) {
	systemPrompt := firstNonEmpty(bot.AISystemPrompt, s.Config.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = "你是 AIM 内置 AI 助手，回答要简洁、准确。"
	}
	if source.IsToGroup() {
		systemPrompt += "\n当前是群聊场景，请只回复本次 @ 你的用户，并避免主动读取无关隐私。"
	}

	history, err := s.loadContextMessages(bot.ID, source)
	if err != nil {
		return nil, err
	}

	messages := []ai.ChatMessage{{Role: "system", Content: systemPrompt}}
	for _, msg := range history {
		if msg.Type != model.MsgText || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		role := "user"
		content := msg.Content
		if msg.FromUserID == bot.ID {
			role = "assistant"
		} else if source.IsToGroup() {
			content = fmt.Sprintf("%s: %s", displayUserName(msg.FromUser), msg.Content)
		}
		messages = append(messages, ai.ChatMessage{Role: role, Content: content})
	}
	return messages, nil
}

func (s *AIService) loadContextMessages(botID uint, source *model.Message) ([]model.Message, error) {
	limit := s.Config.MaxContextMessages
	var messages []model.Message

	query := model.DB.Preload("FromUser").
		Where("id <= ?", source.ID).
		Order("id DESC").
		Limit(limit)

	if source.IsToUser() && source.ToUserID != nil {
		query = query.Where(
			"(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
			source.FromUserID, botID, botID, source.FromUserID,
		)
	} else if source.IsToGroup() && source.GroupID != nil {
		query = query.Where("group_id = ?", *source.GroupID)
	} else {
		return nil, nil
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	reverseMessages(messages)
	return messages, nil
}

func (s *AIService) sendReply(bot model.User, source *model.Message, content string) error {
	reply := &model.Message{
		Type:       model.MsgText,
		FromUserID: bot.ID,
		Content:    strings.TrimSpace(content),
	}
	if source.IsToGroup() && source.GroupID != nil {
		groupID := *source.GroupID
		reply.GroupID = &groupID
		reply.Mentions = marshalMention(source.FromUserID)
	} else {
		toUserID := source.FromUserID
		reply.ToUserID = &toUserID
	}
	return s.Chat.Send(reply)
}

func parseMentionIDs(raw string) []uint {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func containsMention(content, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	return strings.Contains(strings.ToLower(content), "@"+strings.ToLower(name))
}

func marshalMention(userID uint) string {
	data, err := json.Marshal([]uint{userID})
	if err != nil {
		return ""
	}
	return string(data)
}

func displayUserName(user model.User) string {
	if strings.TrimSpace(user.Nickname) != "" {
		return user.Nickname
	}
	if strings.TrimSpace(user.Username) != "" {
		return user.Username
	}
	return fmt.Sprintf("用户%d", user.ID)
}

func reverseMessages(messages []model.Message) {
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
