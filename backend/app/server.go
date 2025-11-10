package app

import (
	"fmt"
	"poker_score_backend/config"
	"poker_score_backend/controllers"
	"poker_score_backend/middlewares"
	"poker_score_backend/models"
	"poker_score_backend/services"
	"poker_score_backend/websocket"

	"github.com/gin-gonic/gin"
)

// NewServer 根据给定配置创建并初始化 Gin 引擎。
// 返回的 cleanup 函数会在适当的时候关闭数据库连接。
func NewServer(cfg *config.Config) (*gin.Engine, func() error, error) {
	if cfg == nil {
		cfg = config.GetConfig()
	}

	if err := models.InitDatabase(
		cfg.Database.Path,
		cfg.Database.MaxIdleConns,
		cfg.Database.MaxOpenConns,
		cfg.Database.ConnMaxLifetime,
	); err != nil {
		return nil, nil, fmt.Errorf("初始化数据库失败: %w", err)
	}

	cleanup := func() error {
		return models.CloseDatabase()
	}

	hub := websocket.NewHub()
	go hub.Run()

	authService := services.NewAuthService(cfg.Session.MaxAge)
	roomService := services.NewRoomService(hub)
	operationService := services.NewOperationService(roomService)
	settlementService := services.NewSettlementService(roomService)
	recordService := services.NewRecordService()
	adminService := services.NewAdminService()

	authController := controllers.NewAuthController(authService, cfg)
	roomController := controllers.NewRoomController(roomService, settlementService)
	operationController := controllers.NewOperationController(operationService)
	settlementController := controllers.NewSettlementController(settlementService)
	recordController := controllers.NewRecordController(recordService)
	adminController := controllers.NewAdminController(adminService)
	wsController := controllers.NewWebSocketController(hub)

	engine := gin.Default()
	engine.Use(middlewares.CORSMiddleware(cfg.Server.AllowedOrigins))

	api := engine.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
			auth.POST("/logout", middlewares.AuthMiddleware(cfg.Session.CookieName), authController.Logout)
			auth.GET("/me", middlewares.AuthMiddleware(cfg.Session.CookieName), authController.GetMe)
			auth.PUT("/nickname", middlewares.AuthMiddleware(cfg.Session.CookieName), authController.UpdateNickname)
			auth.PUT("/password", middlewares.AuthMiddleware(cfg.Session.CookieName), authController.UpdatePassword)
		}

		rooms := api.Group("/rooms", middlewares.AuthMiddleware(cfg.Session.CookieName))
		{
			rooms.POST("", roomController.CreateRoom)
			rooms.POST("/join", roomController.JoinRoom)
			rooms.GET("/last", roomController.GetLastRoom)
			rooms.GET("/:room_id", roomController.GetRoomDetails)
			rooms.POST("/:room_id/return", roomController.ReturnToRoom)
			rooms.POST("/:room_id/leave", roomController.LeaveRoom)
			rooms.POST("/:room_id/kick", roomController.KickUser)
			rooms.POST("/:room_id/dissolve", roomController.DissolveRoom)

			rooms.POST("/:room_id/bet", operationController.Bet)
			rooms.POST("/:room_id/withdraw", operationController.Withdraw)
			rooms.POST("/:room_id/force-transfer", operationController.ForceTransfer)
			rooms.POST("/:room_id/niuniu-bet", operationController.NiuniuBet)
			rooms.GET("/:room_id/operations", operationController.GetOperations)
			rooms.GET("/:room_id/history-amounts", operationController.GetHistoryAmounts)

			rooms.POST("/:room_id/settlement/initiate", settlementController.InitiateSettlement)
			rooms.POST("/:room_id/settlement/confirm", settlementController.ConfirmSettlement)
		}

		records := api.Group("/records", middlewares.AuthMiddleware(cfg.Session.CookieName))
		{
			records.GET("/tonight", recordController.GetTonightRecords)
		}

		admin := api.Group("/admin", middlewares.AuthMiddleware(cfg.Session.CookieName), middlewares.AdminMiddleware())
		{
			admin.GET("/users", adminController.GetUsers)
			admin.PUT("/users/:user_id", adminController.UpdateUser)
			admin.GET("/rooms", adminController.GetRooms)
			admin.GET("/rooms/:room_id", adminController.GetRoomDetails)
			admin.GET("/users/:user_id/settlements", adminController.GetUserSettlements)
			admin.GET("/room-member-history", adminController.GetRoomMemberHistory)
		}

		api.GET("/ws/room/:room_id", middlewares.AuthMiddleware(cfg.Session.CookieName), wsController.HandleWebSocket)
	}

	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	return engine, cleanup, nil
}
