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

// AIConfig 是 service 层使用的全局 AI 运行配置。
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

// AIBotInput 是创建用户自建 AI 时的输入。
type AIBotInput struct {
	Name         string
	Avatar       string
	BaseURL      string
	APIKey       string
	Model        string
	SystemPrompt string
	ContextLimit int
	Temperature  float64
	MaxTokens    int
}

// AIBotUpdateInput 使用指针区分“未传字段”和“传空值”。
type AIBotUpdateInput struct {
	Name         *string
	Avatar       *string
	BaseURL      *string
	APIKey       *string
	Model        *string
	SystemPrompt *string
	ContextLimit *int
	Temperature  *float64
	MaxTokens    *int
	Status       *string
}

// AIBotInfo 是返回给前端的 AI 用户信息，不包含 API Key 明文。
type AIBotInfo struct {
	ID           uint      `json:"id"`
	UserID       uint      `json:"user_id"`
	OwnerID      *uint     `json:"owner_id,omitempty"`
	Username     string    `json:"username"`
	Nickname     string    `json:"nickname"`
	Avatar       string    `json:"avatar"`
	BaseURL      string    `json:"base_url"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt"`
	ContextLimit int       `json:"context_limit"`
	Temperature  float64   `json:"temperature"`
	MaxTokens    int       `json:"max_tokens"`
	Status       string    `json:"status"`
	IsSystem     bool      `json:"is_system"`
	CanEdit      bool      `json:"can_edit"`
	APIKeySet    bool      `json:"api_key_set"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AIService 负责识别 AI 用户、组装上下文并以 AI 用户身份写回消息。
type AIService struct {
	Client ai.Client
	Chat   *ChatService
	Config AIConfig
}

type aiTrigger struct {
	Bot model.User
}

type aiRuntime struct {
	User               model.User
	Bot                *model.AIBot
	BaseURL            string
	APIKey             string
	Model              string
	SystemPrompt       string
	ContextLimit       int
	Temperature        float64
	MaxTokens          int
	OwnedByCurrentUser bool
	SystemManaged      bool
	HasDedicatedConfig bool
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

// EnsureDefaultBot 创建或更新后端托管的内置 AI 虚拟用户。
func (s *AIService) EnsureDefaultBot() (*model.User, error) {
	username := strings.TrimSpace(s.Config.DefaultBotUsername)
	if username == "" {
		return nil, errors.New("default ai bot username is empty")
	}

	var user model.User
	err := model.DB.Where("username = ?", username).First(&user).Error
	if err == nil && !user.IsAI {
		return nil, fmt.Errorf("username %s already belongs to a normal user", username)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
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
	} else {
		if err := model.DB.Model(&user).Updates(s.defaultBotUserUpdates()).Error; err != nil {
			return nil, err
		}
		if err := model.DB.First(&user, user.ID).Error; err != nil {
			return nil, err
		}
	}

	if err := s.ensureSystemBotConfig(user.ID); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUserBots 返回当前用户可使用的内置 AI 和自己创建的 AI。
func (s *AIService) ListUserBots(ownerID uint) ([]AIBotInfo, error) {
	var bots []model.AIBot
	if err := model.DB.Preload("User").
		Where("status <> ? AND (is_system = ? OR owner_id = ?)", model.AIBotStatusDeleted, true, ownerID).
		Order("is_system DESC, updated_at DESC").
		Find(&bots).Error; err != nil {
		return nil, err
	}

	result := make([]AIBotInfo, 0, len(bots))
	for _, bot := range bots {
		result = append(result, toAIBotInfo(bot, ownerID))
	}
	return result, nil
}

// CreateUserBot 创建归属于当前用户的 AI 虚拟用户。
func (s *AIService) CreateUserBot(ownerID uint, input AIBotInput) (*AIBotInfo, error) {
	if err := normalizeCreateAIBotInput(&input); err != nil {
		return nil, err
	}

	var info *AIBotInfo
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		user := model.User{
			Username:       fmt.Sprintf("ai_%d_%d", ownerID, time.Now().UnixNano()),
			Password:       disabledAIPassword,
			Nickname:       input.Name,
			Avatar:         input.Avatar,
			IsAI:           true,
			AIModel:        input.Model,
			AISystemPrompt: input.SystemPrompt,
			AIEndpoint:     input.BaseURL,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		bot := model.AIBot{
			UserID:       user.ID,
			OwnerID:      &ownerID,
			IsSystem:     false,
			BaseURL:      input.BaseURL,
			APIKey:       input.APIKey,
			Model:        input.Model,
			SystemPrompt: input.SystemPrompt,
			ContextLimit: input.ContextLimit,
			Temperature:  input.Temperature,
			MaxTokens:    input.MaxTokens,
			Status:       model.AIBotStatusEnabled,
			User:         user,
		}
		if err := tx.Create(&bot).Error; err != nil {
			return err
		}
		value := toAIBotInfo(bot, ownerID)
		info = &value
		return nil
	})
	if err != nil {
		return nil, err
	}
	return info, nil
}

// UpdateUserBot 只允许创建者修改自己的非内置 AI。
func (s *AIService) UpdateUserBot(ownerID, botID uint, input AIBotUpdateInput) (*AIBotInfo, error) {
	var bot model.AIBot
	if err := model.DB.Preload("User").
		Where("id = ? AND owner_id = ? AND is_system = ? AND status <> ?", botID, ownerID, false, model.AIBotStatusDeleted).
		First(&bot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("AI 助手不存在或无权修改")
		}
		return nil, err
	}
	if err := validateAIBotUpdate(input); err != nil {
		return nil, err
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		userUpdates := map[string]interface{}{}
		botUpdates := map[string]interface{}{}

		if input.Name != nil {
			name := strings.TrimSpace(*input.Name)
			userUpdates["nickname"] = name
		}
		if input.Avatar != nil {
			userUpdates["avatar"] = strings.TrimSpace(*input.Avatar)
		}
		if input.BaseURL != nil {
			baseURL := strings.TrimSpace(*input.BaseURL)
			botUpdates["base_url"] = baseURL
			userUpdates["ai_endpoint"] = baseURL
		}
		if input.APIKey != nil {
			botUpdates["api_key"] = strings.TrimSpace(*input.APIKey)
		}
		if input.Model != nil {
			modelName := strings.TrimSpace(*input.Model)
			botUpdates["model"] = modelName
			userUpdates["ai_model"] = modelName
		}
		if input.SystemPrompt != nil {
			prompt := strings.TrimSpace(*input.SystemPrompt)
			botUpdates["system_prompt"] = prompt
			userUpdates["ai_system_prompt"] = prompt
		}
		if input.ContextLimit != nil {
			botUpdates["context_limit"] = *input.ContextLimit
		}
		if input.Temperature != nil {
			botUpdates["temperature"] = *input.Temperature
		}
		if input.MaxTokens != nil {
			botUpdates["max_tokens"] = *input.MaxTokens
		}
		if input.Status != nil {
			botUpdates["status"] = strings.TrimSpace(*input.Status)
		}

		if len(userUpdates) > 0 {
			if err := tx.Model(&model.User{}).Where("id = ?", bot.UserID).Updates(userUpdates).Error; err != nil {
				return err
			}
		}
		if len(botUpdates) > 0 {
			if err := tx.Model(&model.AIBot{}).Where("id = ?", bot.ID).Updates(botUpdates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := model.DB.Preload("User").First(&bot, bot.ID).Error; err != nil {
		return nil, err
	}
	info := toAIBotInfo(bot, ownerID)
	return &info, nil
}

// DeleteUserBot 采用软删除，保留历史消息中的 AI 用户外键。
func (s *AIService) DeleteUserBot(ownerID, botID uint) error {
	var bot model.AIBot
	if err := model.DB.Where("id = ? AND owner_id = ? AND is_system = ? AND status <> ?", botID, ownerID, false, model.AIBotStatusDeleted).
		First(&bot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("AI 助手不存在或无权删除")
		}
		return err
	}
	return model.DB.Model(&bot).Update("status", model.AIBotStatusDeleted).Error
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

func (s *AIService) defaultBotUserUpdates() map[string]interface{} {
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

func (s *AIService) ensureSystemBotConfig(userID uint) error {
	var bot model.AIBot
	err := model.DB.Where("user_id = ?", userID).First(&bot).Error
	updates := map[string]interface{}{
		"is_system":     true,
		"owner_id":      nil,
		"base_url":      strings.TrimSpace(s.Config.BaseURL),
		"model":         strings.TrimSpace(s.Config.DefaultModel),
		"system_prompt": strings.TrimSpace(s.Config.SystemPrompt),
		"context_limit": s.Config.MaxContextMessages,
		"temperature":   s.Config.Temperature,
		"max_tokens":    s.Config.MaxTokens,
		"status":        model.AIBotStatusEnabled,
	}
	if strings.TrimSpace(s.Config.APIKey) != "" {
		updates["api_key"] = strings.TrimSpace(s.Config.APIKey)
	}
	if err == nil {
		return model.DB.Model(&bot).Updates(updates).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	apiKey := strings.TrimSpace(s.Config.APIKey)
	bot = model.AIBot{
		UserID:       userID,
		IsSystem:     true,
		BaseURL:      strings.TrimSpace(s.Config.BaseURL),
		APIKey:       apiKey,
		Model:        strings.TrimSpace(s.Config.DefaultModel),
		SystemPrompt: strings.TrimSpace(s.Config.SystemPrompt),
		ContextLimit: s.Config.MaxContextMessages,
		Temperature:  s.Config.Temperature,
		MaxTokens:    s.Config.MaxTokens,
		Status:       model.AIBotStatusEnabled,
	}
	return model.DB.Create(&bot).Error
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
		if !bot.IsAI || !s.canTriggerDirect(bot.ID, msg.FromUserID) {
			return nil, nil
		}
		return []aiTrigger{{Bot: bot}}, nil
	case msg.IsToGroup():
		return s.resolveGroupTriggers(msg)
	default:
		return nil, nil
	}
}

func (s *AIService) canTriggerDirect(botUserID, senderID uint) bool {
	runtime, err := s.loadRuntimeByUserID(botUserID)
	if err != nil {
		return false
	}
	if runtime.Bot == nil {
		return true
	}
	if runtime.Bot.IsSystem {
		return true
	}
	return runtime.Bot.OwnerID != nil && *runtime.Bot.OwnerID == senderID
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
		if _, err := s.loadRuntimeByUserID(bot.ID); err != nil {
			continue
		}
		triggers = append(triggers, aiTrigger{Bot: bot})
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
	runtime, err := s.loadRuntime(bot)
	if err != nil {
		return "", err
	}
	if runtime.BaseURL == "" {
		return "", errors.New("ai endpoint is empty")
	}
	if runtime.Model == "" {
		return "", errors.New("ai model is empty")
	}

	messages, err := s.buildPrompt(runtime, source)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(parent, s.Config.Timeout)
	defer cancel()

	temperature := runtime.Temperature
	resp, err := s.Client.Chat(ctx, ai.ChatRequest{
		BaseURL:     runtime.BaseURL,
		APIKey:      runtime.APIKey,
		Model:       runtime.Model,
		Messages:    messages,
		Temperature: &temperature,
		MaxTokens:   runtime.MaxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (s *AIService) loadRuntime(bot model.User) (*aiRuntime, error) {
	return s.loadRuntimeByUser(bot)
}

func (s *AIService) loadRuntimeByUserID(userID uint) (*aiRuntime, error) {
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return s.loadRuntimeByUser(user)
}

func (s *AIService) loadRuntimeByUser(user model.User) (*aiRuntime, error) {
	if !user.IsAI {
		return nil, errors.New("user is not ai")
	}

	var bot model.AIBot
	err := model.DB.Where("user_id = ?", user.ID).First(&bot).Error
	if err == nil {
		if !bot.IsEnabled() {
			return nil, errors.New("ai bot is disabled")
		}
		return &aiRuntime{
			User:               user,
			Bot:                &bot,
			BaseURL:            strings.TrimSpace(bot.BaseURL),
			APIKey:             strings.TrimSpace(bot.APIKey),
			Model:              strings.TrimSpace(bot.Model),
			SystemPrompt:       firstNonEmpty(bot.SystemPrompt, s.Config.SystemPrompt),
			ContextLimit:       defaultInt(bot.ContextLimit, s.Config.MaxContextMessages),
			Temperature:        bot.Temperature,
			MaxTokens:          defaultInt(bot.MaxTokens, s.Config.MaxTokens),
			SystemManaged:      bot.IsSystem,
			HasDedicatedConfig: true,
		}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &aiRuntime{
		User:          user,
		BaseURL:       firstNonEmpty(user.AIEndpoint, s.Config.BaseURL),
		APIKey:        s.Config.APIKey,
		Model:         firstNonEmpty(user.AIModel, s.Config.DefaultModel),
		SystemPrompt:  firstNonEmpty(user.AISystemPrompt, s.Config.SystemPrompt),
		ContextLimit:  s.Config.MaxContextMessages,
		Temperature:   s.Config.Temperature,
		MaxTokens:     s.Config.MaxTokens,
		SystemManaged: false,
	}, nil
}

func (s *AIService) buildPrompt(runtime *aiRuntime, source *model.Message) ([]ai.ChatMessage, error) {
	systemPrompt := runtime.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是 AIM 内置 AI 助手，回答要简洁、准确。"
	}
	if source.IsToGroup() {
		systemPrompt += "\n当前是群聊场景，请只回复本次 @ 你的用户，并避免主动读取无关隐私。"
	}

	history, err := s.loadContextMessages(runtime.User.ID, source, runtime.ContextLimit)
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
		if msg.FromUserID == runtime.User.ID {
			role = "assistant"
		} else if source.IsToGroup() {
			content = fmt.Sprintf("%s: %s", displayUserName(msg.FromUser), msg.Content)
		}
		messages = append(messages, ai.ChatMessage{Role: role, Content: content})
	}
	return messages, nil
}

func (s *AIService) loadContextMessages(botID uint, source *model.Message, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 12
	}
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

func normalizeCreateAIBotInput(input *AIBotInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Avatar = strings.TrimSpace(input.Avatar)
	input.BaseURL = strings.TrimSpace(input.BaseURL)
	input.APIKey = strings.TrimSpace(input.APIKey)
	input.Model = strings.TrimSpace(input.Model)
	input.SystemPrompt = strings.TrimSpace(input.SystemPrompt)
	if input.Name == "" {
		return errors.New("AI 名称不能为空")
	}
	if input.BaseURL == "" {
		return errors.New("API 地址不能为空")
	}
	if input.Model == "" {
		return errors.New("模型名称不能为空")
	}
	if input.ContextLimit <= 0 {
		input.ContextLimit = 12
	}
	if input.MaxTokens <= 0 {
		input.MaxTokens = 1024
	}
	if input.Temperature == 0 {
		input.Temperature = 0.7
	}
	return nil
}

func validateAIBotUpdate(input AIBotUpdateInput) error {
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		return errors.New("AI 名称不能为空")
	}
	if input.BaseURL != nil && strings.TrimSpace(*input.BaseURL) == "" {
		return errors.New("API 地址不能为空")
	}
	if input.Model != nil && strings.TrimSpace(*input.Model) == "" {
		return errors.New("模型名称不能为空")
	}
	if input.ContextLimit != nil && *input.ContextLimit <= 0 {
		return errors.New("上下文条数必须大于 0")
	}
	if input.MaxTokens != nil && *input.MaxTokens <= 0 {
		return errors.New("max_tokens 必须大于 0")
	}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if status != model.AIBotStatusEnabled && status != model.AIBotStatusDisabled {
			return errors.New("AI 状态无效")
		}
	}
	return nil
}

func toAIBotInfo(bot model.AIBot, currentUserID uint) AIBotInfo {
	canEdit := !bot.IsSystem && bot.OwnerID != nil && *bot.OwnerID == currentUserID
	return AIBotInfo{
		ID:           bot.ID,
		UserID:       bot.UserID,
		OwnerID:      bot.OwnerID,
		Username:     bot.User.Username,
		Nickname:     bot.User.Nickname,
		Avatar:       bot.User.Avatar,
		BaseURL:      bot.BaseURL,
		Model:        bot.Model,
		SystemPrompt: bot.SystemPrompt,
		ContextLimit: bot.ContextLimit,
		Temperature:  bot.Temperature,
		MaxTokens:    bot.MaxTokens,
		Status:       bot.Status,
		IsSystem:     bot.IsSystem,
		CanEdit:      canEdit,
		APIKeySet:    strings.TrimSpace(bot.APIKey) != "",
		CreatedAt:    bot.CreatedAt,
		UpdatedAt:    bot.UpdatedAt,
	}
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

func defaultInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
