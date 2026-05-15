// AIM 即时通讯系统服务入口
// 启动 Gin HTTP 服务 + WebSocket Hub + 消息消费协程
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"AIM/config"
	"AIM/internal/handler"
	"AIM/internal/middleware"
	"AIM/internal/model"
	"AIM/internal/service"
	"AIM/internal/ws"
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
	if err := model.InitDB(cfg.Database.DSN); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化 Hub
	hub := ws.NewHub()

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

	// 启动消息消费协程：从 Hub.OnMessage 读取，经 ChatService 持久化并路由
	go func() {
		for msg := range hub.OnMessage {
			if err := chatSvc.Send(msg); err != nil {
				log.Printf("[msg] 消息发送失败: %v", err)
			}
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

	// 初始化文件上传器
	uploader := storage.NewLocalUploader("./data/uploads")
	uploadH := &handler.UploadHandler{Uploader: uploader}

	// 配置 Gin 路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 静态文件与模板
	r.Static("/static", "./web/static")
	r.Static("/data/uploads", "./data/uploads")
	r.LoadHTMLGlob("web/templates/*")

	// 登录注册页面
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "login.html", nil)
	})
	r.GET("/login", func(c *gin.Context) {
		c.HTML(200, "login.html", nil)
	})

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
		auth.GET("/history", chatH.GetHistory)
		auth.GET("/history/group/:id", chatH.GetGroupHistory)
		auth.GET("/history/broadcast", chatH.GetBroadcastHistory)
	}

	// 页面路由（无服务端鉴权，由前端 JS 检查 sessionStorage token）
	r.GET("/contacts", func(c *gin.Context) {
		c.HTML(200, "contacts.html", nil)
	})
	r.GET("/chat", func(c *gin.Context) {
		c.HTML(200, "chat.html", nil)
	})
	r.GET("/groups", func(c *gin.Context) {
		c.HTML(200, "groups.html", nil)
	})
	r.GET("/profile", func(c *gin.Context) {
		c.HTML(200, "profile.html", nil)
	})
	r.GET("/settings", func(c *gin.Context) {
		c.HTML(200, "settings.html", nil)
	})

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
