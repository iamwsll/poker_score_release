package testutil

import (
	"fmt"
	"time"

	"poker_score_backend/app"
	"poker_score_backend/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestConfig 构造测试环境使用的配置。
func TestConfig() *config.Config {
	dbDSN := fmt.Sprintf("file:test-%s.db?mode=memory&cache=shared&_fk=1", uuid.NewString())

	return &config.Config{
		Server: config.ServerConfig{
			Port:           ":0",
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   5 * time.Second,
			AllowedOrigins: []string{"http://localhost"},
			CookieDomain:   "",
			CookieSecure:   false,
			CookieSameSite: "Lax",
		},
		Database: config.DatabaseConfig{
			Path:            dbDSN,
			MaxIdleConns:    1,
			MaxOpenConns:    1,
			ConnMaxLifetime: time.Minute,
		},
		Session: config.SessionConfig{
			CookieName: "poker_test_session",
			MaxAge:     24 * time.Hour,
		},
	}
}

// NewTestServer 创建一个用于测试的 Gin 引擎以及清理函数。
func NewTestServer(cfg *config.Config) (*gin.Engine, func() error, error) {
	gin.SetMode(gin.TestMode)

	if cfg == nil {
		cfg = TestConfig()
	}

	engine, cleanup, err := app.NewServer(cfg)
	if err != nil {
		return nil, nil, err
	}

	return engine, cleanup, nil
}
