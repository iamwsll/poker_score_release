package middlewares

import (
	"poker_score_backend/models"
	"poker_score_backend/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
// 支持从Cookie、Authorization Header或Query参数中获取Session ID
func AuthMiddleware(sessionCookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var sessionID string
		var err error

		// 1. 优先从Authorization Header获取（移动端更可靠）
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// 支持 "Bearer <session_id>" 和直接传 session_id 两种格式
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				sessionID = authHeader[7:]
			} else {
				sessionID = authHeader
			}
		}

		// 2. 如果Header中没有，尝试从Query参数获取（用于WebSocket）
		if sessionID == "" {
			sessionID = c.Query("session_id")
		}

		// 3. 如果Query参数中没有，尝试从Cookie获取
		if sessionID == "" {
			sessionID, err = c.Cookie(sessionCookieName)
			if err != nil || sessionID == "" {
				utils.Unauthorized(c, "未登录或Session已过期，请重新登录")
				c.Abort()
				return
			}
		}

		// 查询Session
		var session models.Session
		err = models.DB.Where("session_id = ?", sessionID).First(&session).Error
		if err != nil {
			utils.Unauthorized(c, "未登录或Session已过期，请重新登录")
			c.Abort()
			return
		}

		// 检查Session是否过期
		if session.IsExpired() {
			// 删除过期的Session
			models.DB.Delete(&session)
			utils.Unauthorized(c, "Session已过期，请重新登录")
			c.Abort()
			return
		}

		// 查询用户信息
		var user models.User
		err = models.DB.First(&user, session.UserID).Error
		if err != nil {
			utils.Unauthorized(c, "用户不存在")
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", user.ID)
		c.Set("user", user)

		c.Next()
	}
}

// AdminMiddleware 管理员权限中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userInterface, exists := c.Get("user")
		if !exists {
			utils.Forbidden(c, "权限不足，需要管理员权限")
			c.Abort()
			return
		}

		user, ok := userInterface.(models.User)
		if !ok {
			utils.Forbidden(c, "权限不足，需要管理员权限")
			c.Abort()
			return
		}

		// 检查是否为管理员
		if user.Role != "admin" {
			utils.Forbidden(c, "权限不足，需要管理员权限")
			c.Abort()
			return
		}

		c.Next()
	}
}
