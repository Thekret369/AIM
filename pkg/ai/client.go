// Package ai 为 LanLine 提供 AI 助手集成能力。
//
// AI 以“虚拟用户”的形态接入系统，聊天服务只关心消息触发和投递，
// 具体大模型厂商通过 Client 接口统一封装。
package ai

import (
	"bufio"
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

// Message 是 AI Agent 接收的 LanLine 消息抽象。
type Message struct {
	Type      string `json:"type"`         // text | image | file | audio
	Content   string `json:"content"`      // 文本内容或文件 URL
	FromID    uint   `json:"from_user_id"` // 发送者 ID
	ToID      *uint  `json:"to_user_id,omitempty"`
	GroupID   *uint  `json:"group_id,omitempty"`
	AtMention bool   `json:"at_mention"` // 是否被 @ 唤起
}

// Agent 描述运行在 LanLine 内的 AI 虚拟用户行为。
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

// ChatStreamChunk 是模型流式返回的一段增量内容。
type ChatStreamChunk struct {
	Delta string
	Done  bool
	Usage Usage
}

// StreamHandler 处理模型流式返回的增量内容。
type StreamHandler func(ChatStreamChunk) error

// Client 封装大模型调用，业务层不直接依赖具体厂商 SDK。
type Client interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

// StreamingClient 表示支持流式 Chat Completions 的客户端。
type StreamingClient interface {
	ChatStream(ctx context.Context, req ChatRequest, onChunk StreamHandler) (*ChatResponse, error)
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
	httpReq, err := c.newChatCompletionRequest(ctx, req, false)
	if err != nil {
		return nil, err
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

// ChatStream 发起一次流式对话请求，并把每段 delta 交给调用方。
func (c *OpenAICompatibleClient) ChatStream(ctx context.Context, req ChatRequest, onChunk StreamHandler) (*ChatResponse, error) {
	httpReq, err := c.newChatCompletionRequest(ctx, req, true)
	if err != nil {
		return nil, err
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

	var usage Usage
	var content strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var out chatCompletionStreamResponse
		if err := json.Unmarshal([]byte(data), &out); err != nil {
			return nil, err
		}
		if out.Error != nil && out.Error.Message != "" {
			return nil, errors.New(out.Error.Message)
		}
		if out.Usage != (Usage{}) {
			usage = out.Usage
		}
		for _, choice := range out.Choices {
			delta := choice.Delta.Content
			if delta == "" {
				continue
			}
			content.WriteString(delta)
			if onChunk != nil {
				if err := onChunk(ChatStreamChunk{Delta: delta}); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	finalContent := strings.TrimSpace(content.String())
	if finalContent == "" {
		return nil, errors.New("ai provider returned empty content")
	}
	return &ChatResponse{Content: finalContent, Usage: usage}, nil
}

func (c *OpenAICompatibleClient) newChatCompletionRequest(ctx context.Context, req ChatRequest, stream bool) (*http.Request, error) {
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
	if stream {
		body["stream"] = true
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
	return httpReq, nil
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

type chatCompletionStreamResponse struct {
	Choices []struct {
		Delta ChatMessage `json:"delta"`
	} `json:"choices"`
	Usage Usage `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
