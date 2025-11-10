package testutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIClient 封装 httptest + Gin Engine，简化接口测试调用。
type APIClient struct {
	engine        *gin.Engine
	cookies       map[string]*http.Cookie
	authorization string
}

// NewAPIClient 创建新的测试客户端。
func NewAPIClient(engine *gin.Engine) *APIClient {
	return &APIClient{
		engine:  engine,
		cookies: make(map[string]*http.Cookie),
	}
}

// SetAuthorization 设置 Authorization 头。
func (c *APIClient) SetAuthorization(value string) {
	c.authorization = value
}

// Authorization 获取当前 Authorization。
func (c *APIClient) Authorization() string {
	return c.authorization
}

// Cookie 获取保存的 Cookie。
func (c *APIClient) Cookie(name string) *http.Cookie {
	return c.cookies[name]
}

// Cookies 返回当前已保存 Cookie 的副本。
func (c *APIClient) Cookies() map[string]*http.Cookie {
	result := make(map[string]*http.Cookie, len(c.cookies))
	for name, cookie := range c.cookies {
		result[name] = cookie
	}
	return result
}

// Do 发送 HTTP 请求，自动处理 JSON 序列化与 Cookie。
func (c *APIClient) Do(method, path string, body interface{}) (*httptest.ResponseRecorder, error) {
	if c.engine == nil {
		return nil, errors.New("engine 未初始化")
	}

	var reader io.Reader
	contentType := ""

	switch v := body.(type) {
	case nil:
	case io.Reader:
		reader = v
	case []byte:
		reader = bytes.NewReader(v)
	case string:
		reader = strings.NewReader(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
		contentType = "application/json"
	}

	req := httptest.NewRequest(method, path, reader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if c.authorization != "" {
		req.Header.Set("Authorization", c.authorization)
	}

	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}

	recorder := httptest.NewRecorder()
	c.engine.ServeHTTP(recorder, req)

	result := recorder.Result()
	if result != nil {
		for _, cookie := range result.Cookies() {
			c.cookies[cookie.Name] = cookie
		}
		result.Body.Close()
	}

	return recorder, nil
}
