package main

import (
	"log"
	"poker_score_backend/config"
	"poker_score_backend/controllers"
	"poker_score_backend/middlewares"
	"poker_score_backend/models"
	"poker_score_backend/services"
	"poker_score_backend/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.GetConfig()

	// 初始化数据库
	err := models.InitDatabase(
		cfg.Database.Path,
		cfg.Database.MaxIdleConns,
		cfg.Database.MaxOpenConns,
		cfg.Database.ConnMaxLifetime,
	)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer func() {
		err := models.CloseDatabase()
		if err != nil {
			log.Printf("数据库关闭失败: %v", err)
		}
	}()

	// 创建WebSocket Hub并启动
	hub := websocket.NewHub()
	go hub.Run()

	// 创建服务层
	authService := services.NewAuthService(cfg.Session.MaxAge)
	roomService := services.NewRoomService(hub)
	operationService := services.NewOperationService(roomService)
	settlementService := services.NewSettlementService(roomService)
	recordService := services.NewRecordService()
	adminService := services.NewAdminService()

	// 创建控制器
	authController := controllers.NewAuthController(authService, cfg)
	roomController := controllers.NewRoomController(roomService)
	operationController := controllers.NewOperationController(operationService)
	settlementController := controllers.NewSettlementController(settlementService)
	recordController := controllers.NewRecordController(recordService)
	adminController := controllers.NewAdminController(adminService)
	wsController := controllers.NewWebSocketController(hub)

	// 创建Gin引擎
	r := gin.Default()

	// 添加CORS中间件
	r.Use(middlewares.CORSMiddleware(cfg.Server.AllowedOrigins))

	// API路由组
	api := r.Group("/api")
	{
		// 认证相关接口
		auth := api.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
			auth.POST("/logout", middlewares.AuthMiddleware(cfg.Session.CookieName), authController.Logout)
			auth.GET("/me", middlewares.AuthMiddleware(cfg.Session.CookieName), authController.GetMe)
			auth.PUT("/nickname", middlewares.AuthMiddleware(cfg.Session.CookieName), authController.UpdateNickname)
			auth.PUT("/password", middlewares.AuthMiddleware(cfg.Session.CookieName), authController.UpdatePassword)
		}

		// 房间相关接口（需要认证）
		rooms := api.Group("/rooms", middlewares.AuthMiddleware(cfg.Session.CookieName))
		{
			rooms.POST("", roomController.CreateRoom)
			rooms.POST("/join", roomController.JoinRoom)
			rooms.GET("/last", roomController.GetLastRoom)
			rooms.GET("/:room_id", roomController.GetRoomDetails)
			rooms.POST("/:room_id/return", roomController.ReturnToRoom)
			rooms.POST("/:room_id/leave", roomController.LeaveRoom)
			rooms.POST("/:room_id/kick", roomController.KickUser)

			// 房间操作
			rooms.POST("/:room_id/bet", operationController.Bet)
			rooms.POST("/:room_id/withdraw", operationController.Withdraw)
			rooms.POST("/:room_id/force-transfer", operationController.ForceTransfer)
			rooms.POST("/:room_id/niuniu-bet", operationController.NiuniuBet)
			rooms.GET("/:room_id/operations", operationController.GetOperations)
			rooms.GET("/:room_id/history-amounts", operationController.GetHistoryAmounts)

			// 结算
			rooms.POST("/:room_id/settlement/initiate", settlementController.InitiateSettlement)
			rooms.POST("/:room_id/settlement/confirm", settlementController.ConfirmSettlement)
		}

		// 战绩统计（需要认证）
		records := api.Group("/records", middlewares.AuthMiddleware(cfg.Session.CookieName))
		{
			records.GET("/tonight", recordController.GetTonightRecords)
		}

		// 后台管理（需要认证和管理员权限）
		admin := api.Group("/admin", middlewares.AuthMiddleware(cfg.Session.CookieName), middlewares.AdminMiddleware())
		{
			admin.GET("/users", adminController.GetUsers)
			admin.GET("/rooms", adminController.GetRooms)
			admin.GET("/rooms/:room_id", adminController.GetRoomDetails)
			admin.GET("/users/:user_id/settlements", adminController.GetUserSettlements)
			admin.GET("/room-member-history", adminController.GetRoomMemberHistory)
		}

		// WebSocket接口（需要认证）
		api.GET("/ws/room/:room_id", middlewares.AuthMiddleware(cfg.Session.CookieName), wsController.HandleWebSocket)
	}

	// 健康检查接口
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// 启动服务器
	log.Printf("服务器启动在端口%s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
