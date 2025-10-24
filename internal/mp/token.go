package mp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// TokenManager Token 管理器
type TokenManager struct {
	baseURL  string
	username string
	password string
	token    string
	mu       sync.RWMutex
	client   *http.Client
}

// LoginRequest 登录请求
type LoginRequest struct {
	GrantType string `json:"grant_type"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

// NewTokenManager 创建 Token 管理器
func NewTokenManager(baseURL, username, password string) *TokenManager {
	return &TokenManager{
		baseURL:  baseURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetToken 获取 Token（如果需要会自动刷新）
func (tm *TokenManager) GetToken(ctx context.Context) (string, error) {
	tm.mu.RLock()
	token := tm.token
	tm.mu.RUnlock()

	// 如果已有 token，直接返回
	if token != "" {
		return token, nil
	}

	// 否则获取新 token
	return tm.RefreshToken(ctx)
}

// RefreshToken 刷新 Token
func (tm *TokenManager) RefreshToken(ctx context.Context) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 构建登录请求
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", tm.username)
	data.Set("password", tm.password)

	// 创建请求
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		tm.baseURL+"/api/v1/login/access-token",
		bytes.NewBufferString(data.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	resp, err := tm.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read login response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("parse login response: %w", err)
	}

	if loginResp.AccessToken == "" {
		return "", fmt.Errorf("login response missing access_token")
	}

	// 保存 token
	tm.token = loginResp.AccessToken

	return tm.token, nil
}

// SetToken 手动设置 Token
func (tm *TokenManager) SetToken(token string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.token = token
}
