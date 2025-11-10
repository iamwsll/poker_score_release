package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Session  SessionConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port           string        // 服务器端口
	ReadTimeout    time.Duration // 读取超时
	WriteTimeout   time.Duration // 写入超时
	AllowedOrigins []string      // 允许的跨域来源
	CookieDomain   string        // Cookie域名
	CookieSecure   bool          // Cookie是否只通过HTTPS传输
	CookieSameSite string        // Cookie的SameSite属性
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path            string        // SQLite数据库文件路径
	MaxIdleConns    int           // 最大空闲连接数
	MaxOpenConns    int           // 最大打开连接数
	ConnMaxLifetime time.Duration // 连接最大生命周期
}

// SessionConfig Session配置
type SessionConfig struct {
	CookieName string        // Session Cookie名称
	MaxAge     time.Duration // Session有效期
}

// GetConfig 获取配置
func GetConfig() *Config {
	env := getEnv("APP_ENV", "development")

	defaultOrigins := []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:80", "capacitor://localhost", "https://localhost", "http://localhost"}
	if env == "production" {
		defaultOrigins = []string{"https://poker.iamwsll.cn"}
	}

	cookieDomain := getEnv("SERVER_COOKIE_DOMAIN", "")
	if env == "production" && cookieDomain == "" {
		cookieDomain = "poker.iamwsll.cn"
	}

	cookieSecure := getEnvAsBool("SERVER_COOKIE_SECURE", env == "production")
	cookieSameSite := normalizeSameSite(getEnv("SERVER_COOKIE_SAME_SITE", "Lax"))

	port := normalizePort(getEnv("SERVER_PORT", ":8080"))

	return &Config{
		Server: ServerConfig{
			Port:           port,
			ReadTimeout:    getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:   getEnvAsDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			AllowedOrigins: getEnvAsList("SERVER_ALLOWED_ORIGINS", defaultOrigins),
			CookieDomain:   cookieDomain,
			CookieSecure:   cookieSecure,
			CookieSameSite: cookieSameSite,
		},
		Database: DatabaseConfig{
			Path:            getEnv("DATABASE_PATH", "./database.db"),
			MaxIdleConns:    getEnvAsInt("DATABASE_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getEnvAsInt("DATABASE_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getEnvAsDuration("DATABASE_CONN_MAX_LIFETIME", time.Hour),
		},
		Session: SessionConfig{
			CookieName: getEnv("SESSION_COOKIE_NAME", "poker_session"),
			MaxAge:     getEnvAsDuration("SESSION_MAX_AGE", 3650*24*time.Hour),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsList(key string, defaultValues []string) []string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		parts := strings.Split(value, ",")
		results := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				results = append(results, trimmed)
			}
		}
		if len(results) > 0 {
			return results
		}
	}
	return defaultValues
}

func normalizePort(port string) string {
	trimmed := strings.TrimSpace(port)
	if trimmed == "" {
		return ":8080"
	}
	if strings.HasPrefix(trimmed, ":") {
		return trimmed
	}
	return ":" + trimmed
}

func normalizeSameSite(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return "Strict"
	case "none":
		return "None"
	case "lax":
		fallthrough
	default:
		return "Lax"
	}
}
