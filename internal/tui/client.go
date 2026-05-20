package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"AIM/internal/model"

	"github.com/gorilla/websocket"
)

const defaultServerURL = "http://127.0.0.1:8080"

// Client 封装 AIM HTTP API 与 WebSocket 地址生成。
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(server string) (*Client, error) {
	baseURL, err := normalizeServerURL(server)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) SetToken(token string) {
	c.token = strings.TrimSpace(token)
}

func (c *Client) Login(ctx context.Context, username, password string) (model.User, error) {
	var resp struct {
		Token string     `json:"token"`
		User  model.User `json:"user"`
	}
	err := c.doJSON(ctx, http.MethodPost, "/api/login", map[string]string{
		"username": username,
		"password": password,
	}, &resp)
	if err != nil {
		return model.User{}, err
	}
	c.SetToken(resp.Token)
	return resp.User, nil
}

func (c *Client) Register(ctx context.Context, username, password, nickname string) (model.User, error) {
	var resp struct {
		User model.User `json:"user"`
	}
	err := c.doJSON(ctx, http.MethodPost, "/api/register", map[string]string{
		"username": username,
		"password": password,
		"nickname": nickname,
	}, &resp)
	if err != nil {
		return model.User{}, err
	}
	return resp.User, nil
}

func (c *Client) Profile(ctx context.Context) (model.User, error) {
	var resp struct {
		User model.User `json:"user"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/profile", nil, &resp); err != nil {
		return model.User{}, err
	}
	return resp.User, nil
}

func (c *Client) Friends(ctx context.Context) ([]friendInfo, error) {
	var resp struct {
		Friends []friendInfo `json:"friends"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/friends", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Friends, nil
}

func (c *Client) Groups(ctx context.Context) ([]model.Group, error) {
	var resp struct {
		Groups []model.Group `json:"groups"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/groups", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Groups, nil
}

func (c *Client) AIBots(ctx context.Context) ([]aiBotInfo, error) {
	var resp struct {
		Bots []aiBotInfo `json:"bots"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/ai/bots", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Bots, nil
}

func (c *Client) History(ctx context.Context, conv Conversation, pageSize int) ([]model.Message, error) {
	if pageSize <= 0 {
		pageSize = 30
	}

	path := ""
	switch conv.Type {
	case conversationUser:
		path = fmt.Sprintf("/api/history?peer_id=%d&page=1&page_size=%d", conv.ID, pageSize)
	case conversationGroup:
		path = fmt.Sprintf("/api/history/group/%d?page=1&page_size=%d", conv.ID, pageSize)
	case conversationBroadcast:
		path = fmt.Sprintf("/api/history/broadcast?page=1&page_size=%d", pageSize)
	default:
		return nil, fmt.Errorf("未知会话类型: %s", conv.Type)
	}

	var resp struct {
		Messages []model.Message `json:"messages"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	reverseMessages(resp.Messages)
	return resp.Messages, nil
}

func (c *Client) DialWebSocket(ctx context.Context) (*websocket.Conn, error) {
	if c.token == "" {
		return nil, errors.New("缺少登录 token")
	}
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/ws"
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	return conn, err
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var payload *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	} else {
		payload = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Error   string `json:"error"`
			Message string `json:"message"`
			Msg     string `json:"msg"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		msg := firstNonEmpty(apiErr.Error, apiErr.Message, apiErr.Msg, resp.Status)
		return errors.New(msg)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func normalizeServerURL(server string) (string, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		server = defaultServerURL
	}
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}
	u, err := url.Parse(server)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("不支持的服务地址协议: %s", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("服务地址缺少 host")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func reverseMessages(messages []model.Message) {
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
