package service

import (
	"time"

	"AIM/internal/model"
	"AIM/pkg/ai"
)

type AITokenUsageInfo struct {
	ID               uint      `json:"id"`
	UserID           uint      `json:"user_id"`
	CallerID         uint      `json:"caller_id"`
	BotID            uint      `json:"bot_id"`
	BotUserID        uint      `json:"bot_user_id"`
	APISource        string    `json:"api_source"`
	Model            string    `json:"model"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	Billable         bool      `json:"billable"`
	InputCostCNY     float64   `json:"input_cost_cny"`
	OutputCostCNY    float64   `json:"output_cost_cny"`
	TotalCostCNY     float64   `json:"total_cost_cny"`
	CreatedAt        time.Time `json:"created_at"`
}

func (s *AIService) ListTokenUsages(userID uint, page, pageSize int) ([]AITokenUsageInfo, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := model.DB.Model(&model.AITokenUsage{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.AITokenUsage
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	result := make([]AITokenUsageInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, toAITokenUsageInfo(row))
	}
	return result, total, nil
}

func (s *AIService) recordTokenUsage(runtime *aiRuntime, source, reply *model.Message, usage ai.Usage) error {
	if runtime == nil || runtime.Bot == nil || source == nil {
		return nil
	}

	totalTokens := usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	accountUserID := source.FromUserID
	if runtime.Bot.OwnerID != nil {
		accountUserID = *runtime.Bot.OwnerID
	}
	billable := runtime.APISource == model.AIBotAPISourceSystem && !runtime.SystemManaged && runtime.Bot.OwnerID != nil

	inputCost := 0.0
	outputCost := 0.0
	if billable {
		inputCost = float64(usage.PromptTokens) * aiInputPricePerMillionCNY / 1000000
		outputCost = float64(usage.CompletionTokens) * aiOutputPricePerMillionCNY / 1000000
	}

	messageID := source.ID
	var replyID *uint
	if reply != nil {
		id := reply.ID
		replyID = &id
	}
	return model.DB.Create(&model.AITokenUsage{
		UserID:           accountUserID,
		CallerID:         source.FromUserID,
		BotID:            runtime.Bot.ID,
		BotUserID:        runtime.User.ID,
		MessageID:        &messageID,
		ReplyMessageID:   replyID,
		APISource:        runtime.APISource,
		ProviderBaseURL:  runtime.BaseURL,
		Model:            runtime.Model,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      totalTokens,
		Billable:         billable,
		InputCostCNY:     inputCost,
		OutputCostCNY:    outputCost,
		TotalCostCNY:     inputCost + outputCost,
	}).Error
}

func (s *AIService) loadBotUsageSummary(userID, botID uint) (AIBotUsageSummary, error) {
	var summary AIBotUsageSummary
	err := model.DB.Model(&model.AITokenUsage{}).
		Select(`
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(CASE WHEN billable THEN total_tokens ELSE 0 END), 0) AS billable_tokens,
			COALESCE(SUM(total_cost_cny), 0) AS total_cost_cny`).
		Where("user_id = ? AND bot_id = ?", userID, botID).
		Scan(&summary).Error
	return summary, err
}

func toAITokenUsageInfo(row model.AITokenUsage) AITokenUsageInfo {
	return AITokenUsageInfo{
		ID:               row.ID,
		UserID:           row.UserID,
		CallerID:         row.CallerID,
		BotID:            row.BotID,
		BotUserID:        row.BotUserID,
		APISource:        row.APISource,
		Model:            row.Model,
		PromptTokens:     row.PromptTokens,
		CompletionTokens: row.CompletionTokens,
		TotalTokens:      row.TotalTokens,
		Billable:         row.Billable,
		InputCostCNY:     row.InputCostCNY,
		OutputCostCNY:    row.OutputCostCNY,
		TotalCostCNY:     row.TotalCostCNY,
		CreatedAt:        row.CreatedAt,
	}
}
