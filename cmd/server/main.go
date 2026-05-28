// AIM 即时通讯系统服务入口
// 启动 Gin HTTP 服务 + WebSocket Hub + 消息消费协程
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"AIM/config"
	"AIM/internal/handler"
	"AIM/internal/middleware"
	"AIM/internal/model"
	"AIM/internal/service"
	"AIM/internal/ws"
	aipkg "AIM/pkg/ai"
	"AIM/pkg/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.Load("config")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	if err := model.InitDB(cfg.Database.Driver, cfg.Database.DSN); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化 Hub
	hub := ws.NewHub()
	hub.UseClientMessages = true
	hub.StatusRecipients = service.StatusVisibleUserIDs

	// 初始化所有 Service
	authSvc := &service.AuthService{
		JWTSecret:    cfg.JWT.Secret,
		JWTExpireHrs: cfg.JWT.ExpireHours,
	}
	friendSvc := &service.FriendService{}
	groupSvc := &service.GroupService{}
	contactSvc := &service.ContactGroupService{}
	chatSvc := &service.ChatService{Hub: hub}
	settingsSvc := &service.SettingsService{}
	var aiSvc *service.AIService
	if cfg.AI.Enabled {
		aiClient := aipkg.NewOpenAICompatibleClient(time.Duration(cfg.AI.TimeoutSeconds) * time.Second)
		aiSvc = service.NewAIService(aiClient, chatSvc, service.AIConfig{
			BaseURL:            cfg.AI.BaseURL,
			APIKey:             cfg.AI.APIKey,
			APIKeyEncryptKey:   firstNonEmpty(cfg.AI.APIKeyEncryptKey, cfg.JWT.Secret),
			DefaultModel:       cfg.AI.DefaultModel,
			DefaultBotUsername: cfg.AI.DefaultBotUsername,
			DefaultBotNickname: cfg.AI.DefaultBotNickname,
			SystemPrompt:       cfg.AI.SystemPrompt,
			Timeout:            time.Duration(cfg.AI.TimeoutSeconds) * time.Second,
			MaxContextMessages: cfg.AI.MaxContextMessages,
			Temperature:        cfg.AI.Temperature,
			MaxTokens:          cfg.AI.MaxTokens,
		})
		chatSvc.AIResponder = aiSvc
		bot, err := aiSvc.EnsureDefaultBot()
		if err != nil {
			log.Printf("[ai] 默认 AI 用户初始化失败: %v", err)
		} else {
			log.Printf("[ai] 默认 AI 用户已就绪: id=%d username=%s", bot.ID, bot.Username)
			if strings.TrimSpace(cfg.AI.BaseURL) == "" || strings.TrimSpace(cfg.AI.DefaultModel) == "" {
				log.Println("[ai] 默认 AI 用户已创建，未配置 base_url/default_model，聊天时将提示 AI 尚未配置")
			}
		}
	}

	// 启动消息消费协程：从 Hub.OnClientMessage 读取，经 ChatService 持久化并向来源连接回 ack/error。
	go func() {
		for event := range hub.OnClientMessage {
			if event == nil || event.Message == nil {
				continue
			}
			if err := chatSvc.SendFromClient(event.Message); err != nil {
				log.Printf("[msg] 消息发送失败: uid=%d, err=%v", event.Message.FromUserID, err)
				if event.Client != nil {
					event.Client.SendError(event.RequestID, "send_failed", err.Error())
				}
				continue
			}
			if event.Client != nil {
				event.Client.SendAck(event.RequestID, event.Message.ID)
			}
		}
	}()

	// 兼容旧的内部投递通道。
	go func() {
		for msg := range hub.OnMessage {
			if err := chatSvc.SendFromClient(msg); err != nil {
				log.Printf("[msg] 消息发送失败: %v", err)
			}
		}
	}()
	go func() {
		for userID := range hub.OnUserOnline {
			chatSvc.SyncOfflineMessages(userID)
		}
	}()
	// 输入状态消费协程
	go func() {
		for p := range hub.OnTyping {
			chatSvc.HandleTyping(p)
		}
	}()
	// 已读回执消费协程
	go func() {
		for p := range hub.OnReadReceipt {
			chatSvc.MarkRead(p)
		}
	}()

	// 初始化所有 Handler
	authH := &handler.AuthHandler{Svc: authSvc}
	friendH := &handler.FriendHandler{Svc: friendSvc, Hub: hub}
	groupH := &handler.GroupHandler{Svc: groupSvc}
	contactH := &handler.ContactGroupHandler{Svc: contactSvc}
	chatH := &handler.ChatHandler{Svc: chatSvc, Hub: hub, JWTSecret: cfg.JWT.Secret}
	settingsH := &handler.SettingsHandler{Svc: settingsSvc}
	aiH := &handler.AIHandler{Svc: aiSvc}

	// 初始化文件上传器
	uploader := storage.NewLocalUploader("./data/uploads")
	uploadH := &handler.UploadHandler{Uploader: uploader}

	// 配置 Gin 路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 静态文件与模板
	r.Static("/static", "./web/static")
	r.GET("/data/uploads/*filepath", serveUploadedFile)
	r.LoadHTMLGlob("web/templates/*")
	registerEmptyAssetFallback(r, "/favicon.ico")
	registerEmptyAssetFallback(r, "/apple-touch-icon.png")
	registerEmptyAssetFallback(r, "/apple-touch-icon-precomposed.png")

	// 登录注册页面
	registerHTMLPage(r, "/", "login.html")
	registerHTMLPage(r, "/login", "login.html")

	// 公开 API（免鉴权）
	api := r.Group("/api")
	{
		api.POST("/register", authH.Register)
		api.POST("/login", authH.Login)
		api.POST("/logout", authH.Logout)
	}

	// WebSocket — 浏览器 WS 不支持 Header 认证，由 Handler 从 ?token= 自验 JWT
	r.GET("/ws", chatH.HandleWS)

	// 认证 API（需 JWT Authorization Header）
	auth := r.Group("/api")
	auth.Use(middleware.AuthRequired(cfg.JWT.Secret))
	{
		// 好友管理
		auth.POST("/friends", friendH.AddFriend)
		auth.PUT("/friends/:id", friendH.HandleRequest)
		auth.DELETE("/friends/:id", friendH.DeleteFriend)
		auth.PUT("/friends/:id/remark", friendH.UpdateRemark)
		auth.GET("/friends", friendH.FriendList)
		auth.GET("/friends/pending", friendH.PendingRequests)
		auth.GET("/friends/online", friendH.GetOnlineFriends)

		// AI 助手管理
		auth.GET("/ai/bots", aiH.ListBots)
		auth.POST("/ai/bots", aiH.CreateBot)
		auth.PUT("/ai/bots/:id", aiH.UpdateBot)
		auth.DELETE("/ai/bots/:id", aiH.DeleteBot)
		auth.POST("/ai/context/reset", aiH.ResetContext)
		auth.GET("/ai/token-usages", aiH.ListTokenUsages)
		auth.GET("/ai/knowledge-bases", aiH.ListKnowledgeBases)
		auth.POST("/ai/knowledge-bases", aiH.CreateKnowledgeBase)
		auth.PUT("/ai/knowledge-bases/:id", aiH.UpdateKnowledgeBase)
		auth.DELETE("/ai/knowledge-bases/:id", aiH.DeleteKnowledgeBase)
		auth.POST("/ai/knowledge-bases/:id/documents", aiH.AddKnowledgeDocument)
		auth.PUT("/ai/knowledge-documents/:id", aiH.UpdateKnowledgeDocument)
		auth.DELETE("/ai/knowledge-documents/:id", aiH.DeleteKnowledgeDocument)

		// 联系人分组
		auth.POST("/contact-groups", contactH.Create)
		auth.PUT("/contact-groups/:id", contactH.Update)
		auth.DELETE("/contact-groups/:id", contactH.Delete)
		auth.GET("/contact-groups", contactH.List)

		// 群组管理
		auth.POST("/groups", groupH.CreateGroup)
		auth.GET("/groups", groupH.GetUserGroups)
		auth.GET("/groups/:id", groupH.GetGroupDetail)
		auth.PUT("/groups/:id", groupH.UpdateGroup)
		auth.POST("/groups/:id/join", groupH.JoinGroup)
		auth.POST("/groups/:id/add-member", groupH.AddMember)
		auth.POST("/groups/:id/leave", groupH.LeaveGroup)
		auth.POST("/groups/:id/kick", groupH.KickMember)
		auth.PUT("/groups/:id/transfer", groupH.TransferOwner)
		auth.PUT("/groups/:id/admin/:user_id", groupH.SetAdmin)
		auth.POST("/groups/:id/mute", groupH.MuteMember)
		auth.POST("/groups/:id/unmute", groupH.UnmuteMember)
		auth.POST("/groups/:id/dnd", groupH.ToggleDND)
		auth.GET("/groups/:id/members", groupH.GetGroupMembers)
		auth.GET("/groups/:id/reads", chatH.GetGroupReads) // 群已读状态恢复
		auth.POST("/groups/:id/announcements", groupH.CreateAnnouncement)
		auth.GET("/groups/:id/announcements", groupH.GetAnnouncements)

		// 个人信息
		auth.GET("/profile", authH.GetProfile)
		auth.GET("/profile/:id", authH.GetUserProfile)
		auth.PUT("/profile", authH.UpdateProfile)
		auth.PUT("/profile/password", authH.ChangePassword)

		// 用户设置
		auth.GET("/settings", settingsH.Get)
		auth.PUT("/settings", settingsH.Update)

		// 文件上传
		auth.POST("/upload", uploadH.HandleUpload)

		// 消息历史
		auth.GET("/search/messages", chatH.SearchMessages)
		auth.POST("/messages/:id/recall", chatH.RecallMessage)
		auth.DELETE("/messages/:id", chatH.DeleteMessage)
		auth.GET("/history", chatH.GetHistory)
		auth.GET("/history/group/:id", chatH.GetGroupHistory)
		auth.GET("/history/broadcast", chatH.GetBroadcastHistory)
	}

	// 页面路由（无服务端鉴权，由前端 JS 检查 sessionStorage token）
	registerHTMLPage(r, "/contacts", "contacts.html")
	registerHTMLPage(r, "/chat", "chat.html")
	registerHTMLPage(r, "/groups", "groups.html")
	registerHTMLPage(r, "/ai", "ai.html")
	registerHTMLPage(r, "/profile", "profile.html")
	registerHTMLPage(r, "/settings", "settings.html")

	// 优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("[server] 正在关闭服务...")
		os.Exit(0)
	}()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("[server] AIM 服务启动于 http://%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[server] 启动失败: %v", err)
	}
}

func registerHTMLPage(r *gin.Engine, path, template string) {
	r.GET(path, func(c *gin.Context) {
		c.HTML(200, template, nil)
	})
	// 部分浏览器预检、代理健康检查会用 HEAD 探测页面，避免误报 404。
	r.HEAD(path, func(c *gin.Context) {
		c.Status(200)
	})
}

func registerEmptyAssetFallback(r *gin.Engine, path string) {
	r.GET(path, func(c *gin.Context) {
		c.Status(204)
	})
	r.HEAD(path, func(c *gin.Context) {
		c.Status(204)
	})
}

func serveUploadedFile(c *gin.Context) {
	raw := strings.TrimPrefix(c.Param("filepath"), "/")
	clean := filepath.Clean(raw)
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		c.Status(404)
		return
	}

	c.Header("X-Content-Type-Options", "nosniff")
	if isActiveUploadExt(strings.ToLower(filepath.Ext(clean))) {
		c.Header("Content-Disposition", "attachment")
	}
	c.File(filepath.Join("./data/uploads", clean))
}

func isActiveUploadExt(ext string) bool {
	switch ext {
	case ".html", ".htm", ".xhtml", ".svg", ".js", ".mjs":
		return true
	default:
		return false
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
