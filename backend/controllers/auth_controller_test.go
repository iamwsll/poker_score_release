package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"poker_score_backend/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const testSessionCookieName = "poker_test_session"

var phoneCounter uint64

func uniquePhone() string {
	n := atomic.AddUint64(&phoneCounter, 1)
	return fmt.Sprintf("13%09d", n%1_000_000_000)
}

func newTestEnv(t *testing.T) (*gin.Engine, *testutil.APIClient) {
	t.Helper()

	engine, cleanup, err := testutil.NewTestServer(nil)
	require.NoError(t, err)

	if cleanup != nil {
		t.Cleanup(func() {
			require.NoError(t, cleanup())
		})
	}

	client := testutil.NewAPIClient(engine)
	return engine, client
}

func decodeResponse(t *testing.T, resp *httptest.ResponseRecorder, out interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), out))
}

func TestRegister_Success(t *testing.T) {
	_, client := newTestEnv(t)

	phone := uniquePhone()

	resp, err := client.Do(http.MethodPost, "/api/auth/register", map[string]string{
		"phone":    phone,
		"nickname": "测试用户",
		"password": "123456",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			User struct {
				ID       uint   `json:"id"`
				Phone    string `json:"phone"`
				Nickname string `json:"nickname"`
				Role     string `json:"role"`
			} `json:"user"`
			SessionID string `json:"session_id"`
		} `json:"data"`
	}

	decodeResponse(t, resp, &response)
	require.Equal(t, 0, response.Code)
	require.Equal(t, "注册成功", response.Message)
	require.Equal(t, phone, response.Data.User.Phone)
	require.Equal(t, "测试用户", response.Data.User.Nickname)
	require.Equal(t, "user", response.Data.User.Role)
	require.NotEmpty(t, response.Data.User.ID)
	require.NotEmpty(t, response.Data.SessionID)
	require.NotNil(t, client.Cookie(testSessionCookieName))
}

func TestLogin_Success(t *testing.T) {
	engine, client := newTestEnv(t)

	phone := uniquePhone()
	password := "123456"

	resp, err := client.Do(http.MethodPost, "/api/auth/register", map[string]string{
		"phone":    phone,
		"nickname": "测试登录用户",
		"password": password,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	loginClient := testutil.NewAPIClient(engine)

	resp, err = loginClient.Do(http.MethodPost, "/api/auth/login", map[string]string{
		"phone":    phone,
		"password": password,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			User struct {
				Phone string `json:"phone"`
				Role  string `json:"role"`
			} `json:"user"`
			SessionID string `json:"session_id"`
		} `json:"data"`
	}

	decodeResponse(t, resp, &response)
	require.Equal(t, 0, response.Code)
	require.Equal(t, "登录成功", response.Message)
	require.Equal(t, phone, response.Data.User.Phone)
	require.Equal(t, "user", response.Data.User.Role)
	require.NotEmpty(t, response.Data.SessionID)
	require.NotNil(t, loginClient.Cookie(testSessionCookieName))
}

func TestGetMe_Success(t *testing.T) {
	_, client := newTestEnv(t)

	phone := uniquePhone()

	resp, err := client.Do(http.MethodPost, "/api/auth/register", map[string]string{
		"phone":    phone,
		"nickname": "查询用户",
		"password": "123456",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	resp, err = client.Do(http.MethodGet, "/api/auth/me", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Phone    string `json:"phone"`
			Nickname string `json:"nickname"`
			Role     string `json:"role"`
		} `json:"data"`
	}

	decodeResponse(t, resp, &response)
	require.Equal(t, 0, response.Code)
	require.Equal(t, phone, response.Data.Phone)
	require.Equal(t, "查询用户", response.Data.Nickname)
	require.Equal(t, "user", response.Data.Role)
}

func TestUpdateNickname_Success(t *testing.T) {
	_, client := newTestEnv(t)

	phone := uniquePhone()

	resp, err := client.Do(http.MethodPost, "/api/auth/register", map[string]string{
		"phone":    phone,
		"nickname": "旧昵称",
		"password": "123456",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	resp, err = client.Do(http.MethodPut, "/api/auth/nickname", map[string]string{
		"nickname": "新昵称",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var updateResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Nickname string `json:"nickname"`
		} `json:"data"`
	}

	decodeResponse(t, resp, &updateResp)
	require.Equal(t, 0, updateResp.Code)
	require.Equal(t, "修改成功", updateResp.Message)
	require.Equal(t, "新昵称", updateResp.Data.Nickname)

	resp, err = client.Do(http.MethodGet, "/api/auth/me", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var meResp struct {
		Data struct {
			Nickname string `json:"nickname"`
		} `json:"data"`
	}

	decodeResponse(t, resp, &meResp)
	require.Equal(t, "新昵称", meResp.Data.Nickname)
}

func TestUpdatePassword_Success(t *testing.T) {
	engine, client := newTestEnv(t)

	phone := uniquePhone()
	oldPassword := "123456"
	newPassword := "654321"

	resp, err := client.Do(http.MethodPost, "/api/auth/register", map[string]string{
		"phone":    phone,
		"nickname": "密码用户",
		"password": oldPassword,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	resp, err = client.Do(http.MethodPut, "/api/auth/password", map[string]string{
		"old_password": oldPassword,
		"new_password": newPassword,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var updateResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	decodeResponse(t, resp, &updateResp)
	require.Equal(t, 0, updateResp.Code)
	require.Equal(t, "密码修改成功", updateResp.Message)

	// 新密码应当可用于登录
	loginClient := testutil.NewAPIClient(engine)
	resp, err = loginClient.Do(http.MethodPost, "/api/auth/login", map[string]string{
		"phone":    phone,
		"password": newPassword,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var loginResp struct {
		Code int `json:"code"`
	}
	decodeResponse(t, resp, &loginResp)
	require.Equal(t, 0, loginResp.Code)

	// 旧密码应当失败
	oldClient := testutil.NewAPIClient(engine)
	resp, err = oldClient.Do(http.MethodPost, "/api/auth/login", map[string]string{
		"phone":    phone,
		"password": oldPassword,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.Code)

	var failResp struct {
		Code int `json:"code"`
	}
	decodeResponse(t, resp, &failResp)
	require.Equal(t, 400, failResp.Code)
}

func TestLogout_ClearsSession(t *testing.T) {
	_, client := newTestEnv(t)

	resp, err := client.Do(http.MethodPost, "/api/auth/register", map[string]string{
		"phone":    uniquePhone(),
		"nickname": "登出用户",
		"password": "123456",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	resp, err = client.Do(http.MethodPost, "/api/auth/logout", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var logoutResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	decodeResponse(t, resp, &logoutResp)
	require.Equal(t, 0, logoutResp.Code)
	require.Equal(t, "登出成功", logoutResp.Message)

	// 再次访问需要登录的接口应当返回未授权
	resp, err = client.Do(http.MethodGet, "/api/auth/me", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	var meResp struct {
		Code int `json:"code"`
	}
	decodeResponse(t, resp, &meResp)
	require.Equal(t, 401, meResp.Code)
}

func TestGetMe_Unauthorized(t *testing.T) {
	_, client := newTestEnv(t)

	resp, err := client.Do(http.MethodGet, "/api/auth/me", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	decodeResponse(t, resp, &response)
	require.Equal(t, 401, response.Code)
	require.NotEmpty(t, response.Message)
}
