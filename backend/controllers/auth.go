package controllers

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"poker_score_backend/config"
	"poker_score_backend/models"
	"poker_score_backend/services"
	"poker_score_backend/utils"

	"github.com/gin-gonic/gin"
)

// AuthController 认证控制器
type AuthController struct {
	authService *services.AuthService
	config      *config.Config
}

// NewAuthController 创建认证控制器
func NewAuthController(authService *services.AuthService, cfg *config.Config) *AuthController {
	return &AuthController{
		authService: authService,
		config:      cfg,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Phone    string `json:"phone" binding:"required,len=11"`
	Nickname string `json:"nickname" binding:"required,min=1,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register 用户注册
func (ctrl *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 调用服务层注册
	user, session, err := ctrl.authService.Register(req.Phone, req.Nickname, req.Password)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 设置Session Cookie
	ctrl.setSessionCookie(c, session.SessionID)

	// 返回响应
	utils.SuccessWithMessage(c, "注册成功", gin.H{
		"user": gin.H{
			"id":         user.ID,
			"phone":      user.Phone,
			"nickname":   user.Nickname,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
		"session_id": session.SessionID,
	})
}

// LoginRequest 登录请求
type LoginRequest struct {
	Phone    string `json:"phone" binding:"required,len=11"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录
func (ctrl *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 调用服务层登录
	user, session, err := ctrl.authService.Login(req.Phone, req.Password)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 设置Session Cookie
	ctrl.setSessionCookie(c, session.SessionID)

	// 返回响应
	utils.SuccessWithMessage(c, "登录成功", gin.H{
		"user": gin.H{
			"id":         user.ID,
			"phone":      user.Phone,
			"nickname":   user.Nickname,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
		"session_id": session.SessionID,
	})
}

// Logout 用户登出
func (ctrl *AuthController) Logout(c *gin.Context) {
	// 获取Session ID
	sessionID, err := c.Cookie(ctrl.config.Session.CookieName)
	if err == nil {
		// 删除Session
		ctrl.authService.Logout(sessionID)
	}

	expiredCookie := &http.Cookie{
		Name:     ctrl.config.Session.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	}

	secure, sameSite := ctrl.resolveCookieSecurity(c)
	expiredCookie.Secure = secure
	expiredCookie.SameSite = sameSite

	if domain := ctrl.resolveCookieDomain(c); domain != "" {
		expiredCookie.Domain = domain
	}

	http.SetCookie(c.Writer, expiredCookie)

	utils.SuccessWithMessage(c, "登出成功", nil)
}

// GetMe 获取当前用户信息
func (ctrl *AuthController) GetMe(c *gin.Context) {
	// 从上下文中获取用户信息
	userInterface, _ := c.Get("user")
	user := userInterface.(models.User)

	utils.Success(c, gin.H{
		"id":         user.ID,
		"phone":      user.Phone,
		"nickname":   user.Nickname,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}

// UpdateNicknameRequest 修改昵称请求
type UpdateNicknameRequest struct {
	Nickname string `json:"nickname" binding:"required,min=1,max=50"`
}

// UpdateNickname 修改昵称
func (ctrl *AuthController) UpdateNickname(c *gin.Context) {
	var req UpdateNicknameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 调用服务层修改昵称
	err := ctrl.authService.UpdateNickname(userID.(uint), req.Nickname)
	if err != nil {
		utils.InternalServerError(c, "修改昵称失败")
		return
	}

	utils.SuccessWithMessage(c, "修改成功", gin.H{
		"nickname": req.Nickname,
	})
}

// UpdatePasswordRequest 修改密码请求
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// UpdatePassword 修改密码
func (ctrl *AuthController) UpdatePassword(c *gin.Context) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 调用服务层修改密码
	err := ctrl.authService.UpdatePassword(userID.(uint), req.OldPassword, req.NewPassword)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "密码修改成功", nil)
}

// setSessionCookie 设置Session Cookie
func (ctrl *AuthController) setSessionCookie(c *gin.Context, sessionID string) {
	maxAge := int(ctrl.config.Session.MaxAge.Seconds())

	cookie := &http.Cookie{
		Name:     ctrl.config.Session.CookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  time.Now().Add(ctrl.config.Session.MaxAge),
		HttpOnly: true,
	}

	secure, sameSite := ctrl.resolveCookieSecurity(c)
	cookie.Secure = secure
	cookie.SameSite = sameSite

	// 移动浏览器建议不设置Domain，让浏览器自动处理
	ua := c.Request.UserAgent()
	if !ctrl.isMobileWebView(ua) {
		// 只在非移动浏览器中设置Domain
		if domain := ctrl.resolveCookieDomain(c); domain != "" {
			cookie.Domain = domain
		}
	}

	http.SetCookie(c.Writer, cookie)
}

func (ctrl *AuthController) resolveCookieSecurity(c *gin.Context) (bool, http.SameSite) {
	secure := ctrl.config.Server.CookieSecure
	if !secure {
		secure = c.Request.TLS != nil
	}

	if !secure {
		if proto := c.Request.Header.Get("X-Forwarded-Proto"); strings.EqualFold(proto, "https") {
			secure = true
		}
	}

	if !secure {
		if scheme := c.Request.Header.Get("X-Forwarded-Scheme"); strings.EqualFold(scheme, "https") {
			secure = true
		}
	}

	// 使用配置的 SameSite 值
	sameSite := ctrl.parseSameSiteFromConfig()

	// 针对移动浏览器的特殊处理
	ua := c.Request.UserAgent()
	if ctrl.isMobileWebView(ua) {
		// 移动浏览器（微信、夸克等）对 SameSite=None 支持不佳
		// 强制使用 Lax，即使在HTTPS环境下
		if sameSite == http.SameSiteNoneMode {
			sameSite = http.SameSiteLaxMode
		}
	}

	return secure, sameSite
}

// 移动浏览器和WebView检测（包括微信、QQ、夸克、支付宝等）
var mobileWebViewRegexp = regexp.MustCompile(`(?i)MicroMessenger|WeChat|QQ/|Quark|UCBrowser|Alipay|DingTalk`)

func (ctrl *AuthController) isMobileWebView(ua string) bool {
	if ua == "" {
		return false
	}

	return mobileWebViewRegexp.MatchString(ua)
}

// 从配置解析 SameSite 值
func (ctrl *AuthController) parseSameSiteFromConfig() http.SameSite {
	switch strings.ToLower(strings.TrimSpace(ctrl.config.Server.CookieSameSite)) {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (ctrl *AuthController) resolveCookieDomain(c *gin.Context) string {
	configured := strings.TrimSpace(ctrl.config.Server.CookieDomain)
	if configured == "" {
		return ""
	}

	host := c.Request.Host
	if host == "" {
		return configured
	}

	if idx := strings.Index(host, ":"); idx >= 0 {
		host = host[:idx]
	}

	if strings.EqualFold(host, configured) {
		return configured
	}

	if strings.HasSuffix(host, configured) {
		prefix := strings.TrimSuffix(host, configured)
		if strings.HasSuffix(prefix, ".") {
			return configured
		}
	}

	return ""
}
