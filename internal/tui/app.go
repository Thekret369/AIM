package tui

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"AIM/internal/model"
	"AIM/internal/ws"

	"github.com/gorilla/websocket"
)

// App 管理终端界面状态、HTTP 数据加载和 WebSocket 实时消息。
type App struct {
	client *Client
	input  *bufio.Reader
	out    io.Writer
	opts   Options

	mu            sync.Mutex
	uiMu          sync.Mutex
	user          model.User
	conversations []Conversation
	activeKey     string
	messages      map[string][]model.Message
	status        string

	wsMu   sync.Mutex
	wsConn *websocket.Conn
}

func NewApp(in io.Reader, out io.Writer, opts Options) (*App, error) {
	client, err := NewClient(opts.Server)
	if err != nil {
		return nil, err
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 30
	}
	return &App{
		client:   client,
		input:    bufio.NewReader(in),
		out:      out,
		opts:     opts,
		messages: make(map[string][]model.Message),
		status:   "输入 /help 查看命令",
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.authenticate(ctx); err != nil {
		return err
	}
	if err := a.refresh(ctx); err != nil {
		a.setStatus("刷新会话失败: " + err.Error())
	}
	if err := a.connectWebSocket(ctx); err != nil {
		a.setStatus("WebSocket 未连接: " + err.Error())
	}
	a.draw()

	for {
		line, err := a.readLine("> ")
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		quit, err := a.handleInput(ctx, strings.TrimSpace(line))
		if err != nil {
			a.setStatus(err.Error())
		}
		if quit {
			a.closeWebSocket()
			return nil
		}
		a.draw()
	}
}

func (a *App) authenticate(ctx context.Context) error {
	for {
		a.drawLogin()
		username := strings.TrimSpace(a.opts.Username)
		password := strings.TrimSpace(a.opts.Password)

		if username == "" {
			value, err := a.readLine("账号（输入 /register 注册）: ")
			if err != nil {
				return err
			}
			username = strings.TrimSpace(value)
		}
		if username == "/register" {
			if err := a.register(ctx); err != nil {
				a.writeNotice("注册失败: " + err.Error())
			}
			continue
		}
		if username == "" {
			a.writeNotice("账号不能为空")
			continue
		}
		if password == "" {
			value, err := a.readLine("密码: ")
			if err != nil {
				return err
			}
			password = strings.TrimSpace(value)
		}

		user, err := a.client.Login(ctx, username, password)
		if err != nil {
			a.writeNotice("登录失败: " + err.Error())
			a.opts.Username = ""
			a.opts.Password = ""
			continue
		}
		a.mu.Lock()
		a.user = user
		a.mu.Unlock()
		return nil
	}
}

func (a *App) register(ctx context.Context) error {
	username, err := a.readLine("新账号: ")
	if err != nil {
		return err
	}
	password, err := a.readLine("新密码: ")
	if err != nil {
		return err
	}
	nickname, err := a.readLine("昵称（可空）: ")
	if err != nil {
		return err
	}
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	nickname = strings.TrimSpace(nickname)
	if username == "" || password == "" {
		return errors.New("账号和密码不能为空")
	}
	if _, err := a.client.Register(ctx, username, password, nickname); err != nil {
		return err
	}
	a.opts.Username = username
	a.opts.Password = password
	a.writeNotice("注册成功，正在登录")
	return nil
}

func (a *App) handleInput(ctx context.Context, line string) (bool, error) {
	if line == "" {
		return false, nil
	}
	if !strings.HasPrefix(line, "/") {
		return false, a.sendActiveMessage(ctx, line)
	}

	fields := strings.Fields(line)
	cmd := strings.ToLower(fields[0])
	args := fields[1:]
	switch cmd {
	case "/quit", "/exit":
		return true, nil
	case "/help":
		a.setStatus("命令: /open <序号|user:id|group:id> /convs /refresh /history [条数] /ai /profile /quit；打开会话后直接输入文本发送")
	case "/convs":
		a.mu.Lock()
		a.activeKey = ""
		a.mu.Unlock()
		a.setStatus("已切回会话列表")
	case "/refresh":
		if err := a.refresh(ctx); err != nil {
			return false, err
		}
		a.setStatus("会话已刷新")
	case "/open":
		if err := a.openConversation(ctx, args); err != nil {
			return false, err
		}
	case "/history":
		pageSize := a.opts.PageSize
		if len(args) > 0 {
			n, err := strconv.Atoi(args[0])
			if err != nil || n <= 0 {
				return false, errors.New("历史条数必须是正整数")
			}
			pageSize = n
		}
		if err := a.loadActiveHistory(ctx, pageSize); err != nil {
			return false, err
		}
	case "/send":
		text := strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		if text == "" {
			return false, errors.New("发送内容不能为空")
		}
		return false, a.sendActiveMessage(ctx, text)
	case "/ai":
		a.setStatus(a.aiSummary())
	case "/profile":
		a.setStatus(a.profileSummary())
	default:
		return false, fmt.Errorf("未知命令: %s", cmd)
	}
	return false, nil
}

func (a *App) refresh(ctx context.Context) error {
	friends, err := a.client.Friends(ctx)
	if err != nil {
		return err
	}
	groups, err := a.client.Groups(ctx)
	if err != nil {
		return err
	}
	bots, botErr := a.client.AIBots(ctx)

	convs := make([]Conversation, 0, len(friends)+len(groups)+len(bots)+1)
	convs = append(convs, Conversation{
		Key:      conversationBroadcast,
		Type:     conversationBroadcast,
		Name:     "广播消息",
		ReadOnly: true,
	})

	seen := map[string]struct{}{conversationBroadcast: {}}
	for _, friend := range friends {
		name := firstNonEmpty(friend.Remark, friend.Nickname, friend.Username, fmt.Sprintf("用户%d", friend.FriendID))
		conv := Conversation{
			Key:  conversationKey(conversationUser, friend.FriendID),
			Type: conversationUser,
			ID:   friend.FriendID,
			Name: name,
		}
		seen[conv.Key] = struct{}{}
		convs = append(convs, conv)
	}
	if botErr == nil {
		for _, bot := range bots {
			if bot.Status == "disabled" {
				continue
			}
			key := conversationKey(conversationUser, bot.UserID)
			if _, ok := seen[key]; ok {
				continue
			}
			name := firstNonEmpty(bot.Nickname, bot.Username, fmt.Sprintf("AI%d", bot.UserID))
			convs = append(convs, Conversation{
				Key:  key,
				Type: conversationUser,
				ID:   bot.UserID,
				Name: name,
				IsAI: true,
			})
			seen[key] = struct{}{}
		}
	}
	for _, group := range groups {
		name := firstNonEmpty(group.Name, fmt.Sprintf("群组%d", group.ID))
		convs = append(convs, Conversation{
			Key:  conversationKey(conversationGroup, group.ID),
			Type: conversationGroup,
			ID:   group.ID,
			Name: name,
		})
	}

	a.mu.Lock()
	a.conversations = convs
	if a.activeKey != "" && !containsConversation(convs, a.activeKey) {
		a.activeKey = ""
	}
	if botErr != nil {
		a.status = "会话已刷新；AI 助手列表不可用: " + botErr.Error()
	}
	a.mu.Unlock()
	return nil
}

func (a *App) openConversation(ctx context.Context, args []string) error {
	conv, err := a.resolveConversation(args)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.activeKey = conv.Key
	a.mu.Unlock()

	if err := a.loadHistory(ctx, conv, a.opts.PageSize); err != nil {
		return err
	}
	a.setStatus("已打开: " + conv.Name)
	return nil
}

func (a *App) loadActiveHistory(ctx context.Context, pageSize int) error {
	conv, ok := a.activeConversation()
	if !ok {
		return errors.New("请先使用 /open 打开一个会话")
	}
	if err := a.loadHistory(ctx, conv, pageSize); err != nil {
		return err
	}
	a.setStatus(fmt.Sprintf("已加载最近 %d 条历史", pageSize))
	return nil
}

func (a *App) loadHistory(ctx context.Context, conv Conversation, pageSize int) error {
	msgs, err := a.client.History(ctx, conv, pageSize)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.messages[conv.Key] = msgs
	a.mu.Unlock()
	return nil
}

func (a *App) sendActiveMessage(ctx context.Context, text string) error {
	conv, ok := a.activeConversation()
	if !ok {
		return errors.New("请先使用 /open 打开一个会话")
	}
	if conv.ReadOnly || conv.Type == conversationBroadcast {
		return errors.New("当前会话只读，不能发送消息")
	}
	if err := a.ensureWebSocket(ctx); err != nil {
		return err
	}

	msg := model.Message{
		Type:       model.MsgText,
		FromUserID: a.currentUserID(),
		Content:    text,
	}
	switch conv.Type {
	case conversationUser:
		to := conv.ID
		msg.ToUserID = &to
	case conversationGroup:
		groupID := conv.ID
		msg.GroupID = &groupID
	default:
		return errors.New("未知会话类型")
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	envelope := ws.WSMessage{
		Type:    ws.WSMChat,
		Payload: payload,
	}

	a.wsMu.Lock()
	conn := a.wsConn
	a.wsMu.Unlock()
	if conn == nil {
		return errors.New("WebSocket 未连接")
	}
	if err := conn.WriteJSON(envelope); err != nil {
		a.closeWebSocket()
		return err
	}
	a.setStatus("消息已发送")
	return nil
}

func (a *App) connectWebSocket(ctx context.Context) error {
	conn, err := a.client.DialWebSocket(ctx)
	if err != nil {
		return err
	}
	a.wsMu.Lock()
	if a.wsConn != nil {
		_ = a.wsConn.Close()
	}
	a.wsConn = conn
	a.wsMu.Unlock()

	go a.readWebSocket(conn)
	return nil
}

func (a *App) ensureWebSocket(ctx context.Context) error {
	a.wsMu.Lock()
	connected := a.wsConn != nil
	a.wsMu.Unlock()
	if connected {
		return nil
	}
	return a.connectWebSocket(ctx)
}

func (a *App) readWebSocket(conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			a.wsMu.Lock()
			if a.wsConn == conn {
				a.wsConn = nil
			}
			a.wsMu.Unlock()
			a.setStatus("WebSocket 已断开: " + err.Error())
			a.draw()
			return
		}
		if err := a.handleWebSocketData(data); err != nil {
			a.setStatus("消息解析失败: " + err.Error())
			a.draw()
		}
	}
}

func (a *App) handleWebSocketData(data []byte) error {
	var envelope ws.WSMessage
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Type != "" {
		switch envelope.Type {
		case ws.WSMChat:
			var msg model.Message
			if err := json.Unmarshal(envelope.Payload, &msg); err != nil {
				return err
			}
			a.addIncomingMessage(msg)
		case ws.WSMTyping:
			a.setStatus("对方正在输入")
		case ws.WSMStatus:
			a.setStatus("在线状态已更新")
		case ws.WSMReadReceipt:
			a.setStatus("已读状态已更新")
		}
		return nil
	}

	var msg model.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	a.addIncomingMessage(msg)
	return nil
}

func (a *App) addIncomingMessage(msg model.Message) {
	conv := a.conversationFromMessage(msg)
	a.mu.Lock()
	if !containsConversation(a.conversations, conv.Key) {
		a.conversations = append(a.conversations, conv)
	}
	a.messages[conv.Key] = appendMessage(a.messages[conv.Key], msg)
	a.status = fmt.Sprintf("收到 %s 的新消息", conv.Name)
	a.mu.Unlock()
	a.draw()
}

func (a *App) conversationFromMessage(msg model.Message) Conversation {
	if msg.GroupID != nil && *msg.GroupID > 0 {
		id := *msg.GroupID
		if conv, ok := a.findConversation(conversationKey(conversationGroup, id)); ok {
			return conv
		}
		return Conversation{
			Key:  conversationKey(conversationGroup, id),
			Type: conversationGroup,
			ID:   id,
			Name: fmt.Sprintf("群组%d", id),
		}
	}
	if msg.ToUserID != nil && *msg.ToUserID > 0 {
		peerID := *msg.ToUserID
		if msg.FromUserID != a.currentUserID() {
			peerID = msg.FromUserID
		}
		if conv, ok := a.findConversation(conversationKey(conversationUser, peerID)); ok {
			return conv
		}
		name := fmt.Sprintf("用户%d", peerID)
		if msg.FromUser.ID == peerID {
			name = displayUserName(msg.FromUser)
		}
		return Conversation{
			Key:  conversationKey(conversationUser, peerID),
			Type: conversationUser,
			ID:   peerID,
			Name: name,
		}
	}
	return Conversation{
		Key:      conversationBroadcast,
		Type:     conversationBroadcast,
		Name:     "广播消息",
		ReadOnly: true,
	}
}

func (a *App) drawLogin() {
	a.uiMu.Lock()
	defer a.uiMu.Unlock()
	fmt.Fprint(a.out, "\033[2J\033[H")
	fmt.Fprintln(a.out, "AIM TUI CLI")
	fmt.Fprintln(a.out, "服务:", a.client.BaseURL())
	fmt.Fprintln(a.out, "----------------------------------------")
}

func (a *App) draw() {
	a.mu.Lock()
	user := a.user
	activeKey := a.activeKey
	status := a.status
	convs := append([]Conversation(nil), a.conversations...)
	msgs := append([]model.Message(nil), a.messages[activeKey]...)
	active, _ := findConversation(convs, activeKey)
	a.mu.Unlock()

	a.uiMu.Lock()
	defer a.uiMu.Unlock()
	fmt.Fprint(a.out, "\033[2J\033[H")
	fmt.Fprintf(a.out, "AIM TUI | %s | %s\n", displayUserName(user), a.client.BaseURL())
	fmt.Fprintln(a.out, "----------------------------------------")
	if status != "" {
		fmt.Fprintln(a.out, status)
		fmt.Fprintln(a.out, "----------------------------------------")
	}
	if activeKey == "" {
		a.drawConversationList(convs)
	} else {
		a.drawActiveConversation(active, msgs)
	}
	fmt.Fprintln(a.out, "----------------------------------------")
	fmt.Fprintln(a.out, "命令: /open <序号|user:id|group:id> /convs /refresh /history [条数] /ai /profile /quit")
	if activeKey != "" && !active.ReadOnly {
		fmt.Fprintln(a.out, "发送: 直接输入文本，或 /send 文本")
	}
}

func (a *App) drawConversationList(convs []Conversation) {
	if len(convs) == 0 {
		fmt.Fprintln(a.out, "暂无会话，使用 /refresh 重试")
		return
	}
	fmt.Fprintln(a.out, "会话列表")
	for i, conv := range convs {
		label := conv.Type
		if conv.IsAI {
			label = "ai"
		}
		if conv.ReadOnly {
			label += "/ro"
		}
		fmt.Fprintf(a.out, "%2d. %-12s #%d %s\n", i+1, label, conv.ID, conv.Name)
	}
}

func (a *App) drawActiveConversation(conv Conversation, messages []model.Message) {
	if conv.Key == "" {
		fmt.Fprintln(a.out, "会话不存在，使用 /convs 返回列表")
		return
	}
	flag := ""
	if conv.IsAI {
		flag = " AI"
	}
	if conv.ReadOnly {
		flag += " 只读"
	}
	fmt.Fprintf(a.out, "[%s%s] %s #%d\n\n", conv.Type, flag, conv.Name, conv.ID)
	if len(messages) == 0 {
		fmt.Fprintln(a.out, "暂无历史消息")
		return
	}
	start := 0
	if len(messages) > 20 {
		start = len(messages) - 20
	}
	for _, msg := range messages[start:] {
		fmt.Fprintln(a.out, formatMessage(a.currentUserID(), msg))
	}
}

func (a *App) drawAIList() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	var items []string
	for i, conv := range a.conversations {
		if conv.IsAI {
			items = append(items, fmt.Sprintf("%d:%s(#%d)", i+1, conv.Name, conv.ID))
		}
	}
	if len(items) == 0 {
		return "当前没有可用 AI 助手"
	}
	return "AI 助手: " + strings.Join(items, " ")
}

func (a *App) aiSummary() string {
	return a.drawAIList()
}

func (a *App) profileSummary() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return fmt.Sprintf("当前用户: %s #%d", displayUserName(a.user), a.user.ID)
}

func (a *App) resolveConversation(args []string) (Conversation, error) {
	if len(args) == 0 {
		return Conversation{}, errors.New("用法: /open <序号|user:id|group:id>")
	}
	a.mu.Lock()
	convs := append([]Conversation(nil), a.conversations...)
	a.mu.Unlock()

	if len(args) == 1 {
		raw := strings.TrimSpace(args[0])
		if idx, err := strconv.Atoi(raw); err == nil {
			if idx < 1 || idx > len(convs) {
				return Conversation{}, errors.New("会话序号不存在")
			}
			return convs[idx-1], nil
		}
		if strings.Contains(raw, ":") {
			parts := strings.SplitN(raw, ":", 2)
			return findByTypeAndID(convs, parts[0], parts[1])
		}
	}
	if len(args) >= 2 {
		return findByTypeAndID(convs, args[0], args[1])
	}
	return Conversation{}, errors.New("用法: /open <序号|user:id|group:id>")
}

func (a *App) activeConversation() (Conversation, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return findConversation(a.conversations, a.activeKey)
}

func (a *App) findConversation(key string) (Conversation, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return findConversation(a.conversations, key)
}

func (a *App) currentUserID() uint {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.user.ID
}

func (a *App) setStatus(status string) {
	a.mu.Lock()
	a.status = status
	a.mu.Unlock()
}

func (a *App) readLine(prompt string) (string, error) {
	a.uiMu.Lock()
	fmt.Fprint(a.out, prompt)
	a.uiMu.Unlock()
	line, err := a.input.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func (a *App) writeNotice(message string) {
	a.uiMu.Lock()
	defer a.uiMu.Unlock()
	fmt.Fprintln(a.out, message)
}

func (a *App) closeWebSocket() {
	a.wsMu.Lock()
	defer a.wsMu.Unlock()
	if a.wsConn != nil {
		_ = a.wsConn.Close()
		a.wsConn = nil
	}
}

func findByTypeAndID(convs []Conversation, typ, rawID string) (Conversation, error) {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "ai" {
		typ = conversationUser
	}
	id, err := strconv.ParseUint(strings.TrimSpace(rawID), 10, 64)
	if err != nil || id == 0 {
		return Conversation{}, errors.New("会话 ID 必须是正整数")
	}
	key := conversationKey(typ, uint(id))
	for _, conv := range convs {
		if conv.Key == key {
			return conv, nil
		}
	}
	return Conversation{}, errors.New("会话不存在，请先 /refresh")
}

func findConversation(convs []Conversation, key string) (Conversation, bool) {
	for _, conv := range convs {
		if conv.Key == key {
			return conv, true
		}
	}
	return Conversation{}, false
}

func containsConversation(convs []Conversation, key string) bool {
	_, ok := findConversation(convs, key)
	return ok
}

func appendMessage(messages []model.Message, msg model.Message) []model.Message {
	if msg.ID > 0 {
		for _, item := range messages {
			if item.ID == msg.ID {
				return messages
			}
		}
	}
	messages = append(messages, msg)
	sort.SliceStable(messages, func(i, j int) bool {
		if messages[i].ID != 0 && messages[j].ID != 0 {
			return messages[i].ID < messages[j].ID
		}
		return messages[i].CreatedAt.Before(messages[j].CreatedAt)
	})
	return messages
}

func conversationKey(typ string, id uint) string {
	if typ == conversationBroadcast {
		return conversationBroadcast
	}
	return fmt.Sprintf("%s:%d", typ, id)
}

func displayUserName(user model.User) string {
	return firstNonEmpty(user.Nickname, user.Username, fmt.Sprintf("用户%d", user.ID))
}

func formatMessage(currentUserID uint, msg model.Message) string {
	sender := fmt.Sprintf("用户%d", msg.FromUserID)
	if msg.FromUserID == currentUserID {
		sender = "我"
	} else if msg.FromUser.ID != 0 {
		sender = displayUserName(msg.FromUser)
	}
	clock := ""
	if !msg.CreatedAt.IsZero() {
		clock = msg.CreatedAt.Local().Format("01-02 15:04")
	} else {
		clock = time.Now().Format("01-02 15:04")
	}
	content := msg.Content
	if msg.Type != model.MsgText {
		content = fmt.Sprintf("[%s] %s", msg.Type, firstNonEmpty(msg.FileName, msg.Content))
	}
	return fmt.Sprintf("%s %s: %s", clock, sender, content)
}
