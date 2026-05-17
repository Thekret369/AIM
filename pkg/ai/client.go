// Package ai 为 AIM 提供 AI 助手集成能力。
//
// AI 以“虚拟用户”的形态接入系统，聊天服务只关心消息触发和投递，
// 具体大模型厂商通过 Client 接口统一封装。
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message 是 AI Agent 接收的 AIM 消息抽象。
type Message struct {
	Type      string `json:"type"`         // text | image | file | audio
	Content   string `json:"content"`      // 文本内容或文件 URL
	FromID    uint   `json:"from_user_id"` // 发送者 ID
	ToID      *uint  `json:"to_user_id,omitempty"`
	GroupID   *uint  `json:"group_id,omitempty"`
	AtMention bool   `json:"at_mention"` // 是否被 @ 唤起
}

// Agent 描述运行在 AIM 内的 AI 虚拟用户行为。
type Agent interface {
	ID() uint
	Process(ctx context.Context, msg *Message) (reply string, err error)
	Connect(wsURL, token string) error
}

// ChatMessage 是发送给大模型的标准聊天消息。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 是一次大模型对话请求。
type ChatRequest struct {
	BaseURL     string
	APIKey      string
	Model       string
	Messages    []ChatMessage
	Temperature *float64
	MaxTokens   int
}

// Usage 记录模型侧返回的 token 用量。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse 是统一后的模型回复。
type ChatResponse struct {
	Content string
	Usage   Usage
}

// Client 封装大模型调用，业务层不直接依赖具体厂商 SDK。
type Client interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

// OpenAICompatibleClient 调用兼容 OpenAI Chat Completions 协议的模型服务。
type OpenAICompatibleClient struct {
	httpClient *http.Client
}

// NewOpenAICompatibleClient 创建 OpenAI-compatible 客户端。
func NewOpenAICompatibleClient(timeout time.Duration) *OpenAICompatibleClient {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &OpenAICompatibleClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Chat 发起一次非流式对话请求。
func (c *OpenAICompatibleClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if c == nil {
		return nil, errors.New("ai client is nil")
	}
	if strings.TrimSpace(req.BaseURL) == "" {
		return nil, errors.New("ai base url is empty")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, errors.New("ai model is empty")
	}
	if len(req.Messages) == 0 {
		return nil, errors.New("ai messages is empty")
	}

	body := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionsURL(req.BaseURL), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(req.APIKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(req.APIKey))
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("ai provider status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var out chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != nil && out.Error.Message != "" {
		return nil, errors.New(out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return nil, errors.New("ai provider returned empty choices")
	}

	content := strings.TrimSpace(out.Choices[0].Message.Content)
	if content == "" {
		return nil, errors.New("ai provider returned empty content")
	}
	return &ChatResponse{Content: content, Usage: out.Usage}, nil
}

func chatCompletionsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

type chatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
