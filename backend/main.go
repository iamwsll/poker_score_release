package main

import (
	"log"
	"poker_score_backend/app"
	"poker_score_backend/config"
)

func main() {
	// 加载配置
	cfg := config.GetConfig()

	// 初始化服务器
	engine, cleanup, err := app.NewServer(cfg)
	if err != nil {
		log.Fatalf("服务器初始化失败: %v", err)
	}
	defer func() {
		if cleanup != nil {
			if err := cleanup(); err != nil {
				log.Printf("服务器清理失败: %v", err)
			}
		}
	}()

	// 启动服务器
	log.Printf("服务器启动在端口%s", cfg.Server.Port)
	if err := engine.Run(cfg.Server.Port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
