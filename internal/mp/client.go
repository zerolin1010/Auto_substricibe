package mp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// AuthScheme 认证方案
type AuthScheme string

const (
	AuthSchemeBearer     AuthScheme = "bearer"
	AuthSchemeXAPIToken  AuthScheme = "x-api-token"
	AuthSchemeQueryToken AuthScheme = "query-token"
)

// Client MoviePilot API 客户端
type Client struct {
	baseURL      string
	tokenManager *TokenManager
	authScheme   AuthScheme
	httpClient   *http.Client
	limiter      *rate.Limiter
	maxRetries   int
	dryRun       bool
}

// ClientConfig 客户端配置
type ClientConfig struct {
	BaseURL      string
	Username     string // MoviePilot 用户名（必需）
	Password     string // MoviePilot 密码（必需）
	AuthScheme   string
	RateLimitPS  int  // 每秒请求数
	MaxRetries   int  // 最大重试次数
	DryRun       bool // 干跑模式
	TokenRefresh int  // Token 刷新间隔（小时）
}

// NewClient 创建客户端
func NewClient(cfg ClientConfig, ctx context.Context) (*Client, error) {
	// 设置速率限制器
	limiter := rate.NewLimiter(rate.Limit(cfg.RateLimitPS), cfg.RateLimitPS)

	// 创建 token 管理器
	tokenManager := NewTokenManager(cfg.BaseURL, cfg.Username, cfg.Password)

	// 立即获取 token
	_, err := tokenManager.RefreshToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("initial token fetch: %w", err)
	}

	client := &Client{
		baseURL:      cfg.BaseURL,
		tokenManager: tokenManager,
		authScheme:   AuthScheme(cfg.AuthScheme),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter:    limiter,
		maxRetries: cfg.MaxRetries,
		dryRun:     cfg.DryRun,
	}

	return client, nil
}

// Subscribe 创建订阅
func (c *Client) Subscribe(ctx context.Context, req *SubscribeRequest) (*SubscribeResponse, error) {
	// 干跑模式：仅打印请求
	if c.dryRun {
		data, _ := json.MarshalIndent(req, "", "  ")
		fmt.Printf("[DRY-RUN] Would create subscription:\n%s\n", string(data))
		return &SubscribeResponse{
			Success: true,
			Message: "dry-run mode",
			Data: &SubscribeData{
				ID: 99999, // 干跑模式的模拟 ID
			},
		}, nil
	}

	// 执行请求（带重试）
	var resp *SubscribeResponse
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, lastErr = c.doSubscribe(ctx, req)
		if lastErr == nil {
			return resp, nil
		}

		// 检查是否应该重试
		if !c.shouldRetry(lastErr) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doSubscribe 执行订阅请求
func (c *Client) doSubscribe(ctx context.Context, req *SubscribeRequest) (*SubscribeResponse, error) {
	// 等待速率限制
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	// 构建请求体
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 构建 URL（注意末尾的斜杠）
	url := c.baseURL + "/api/v1/subscribe/"

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置认证
	if err := c.setAuth(httpReq, ctx); err != nil {
		return nil, err
	}

	// 设置头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, &RetryableError{Err: err}
	}
	defer httpResp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 检查状态码
	if httpResp.StatusCode == http.StatusTooManyRequests {
		return nil, &RetryableError{
			Err:        fmt.Errorf("rate limited (429)"),
			StatusCode: httpResp.StatusCode,
		}
	}

	if httpResp.StatusCode >= 500 {
		return nil, &RetryableError{
			Err:        fmt.Errorf("server error (%d): %s", httpResp.StatusCode, string(respBody)),
			StatusCode: httpResp.StatusCode,
		}
	}

	if httpResp.StatusCode >= 400 {
		return nil, &NonRetryableError{
			Err:        fmt.Errorf("client error (%d): %s", httpResp.StatusCode, string(respBody)),
			StatusCode: httpResp.StatusCode,
		}
	}

	// 解析响应
	var response SubscribeResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}

	// 记录响应内容用于调试
	// 注意：生产环境可以设置为 Debug 级别
	// log.Printf("[MP Subscribe] Success=%v, Message=%s, Code=%d, Body=%s",
	// 	response.Success, response.Message, response.Code, string(respBody))

	// 检查业务状态
	if !response.Success {
		return nil, &NonRetryableError{
			Err:     fmt.Errorf("subscribe failed: %s", response.Message),
			Message: response.Message,
		}
	}

	return &response, nil
}

// SearchMedia 搜索媒体
func (c *Client) SearchMedia(ctx context.Context, req *MediaSearchRequest) (*MediaSearchResponse, error) {
	// 等待速率限制
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	// 构建请求体
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 构建 URL（注意末尾的斜杠）
	url := c.baseURL + "/api/v1/media/search/"

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置认证
	if err := c.setAuth(httpReq, ctx); err != nil {
		return nil, err
	}

	// 设置头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 检查状态码
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// 解析响应
	var response MediaSearchResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &response, nil
}

// setAuth 设置认证
func (c *Client) setAuth(req *http.Request, ctx context.Context) error {
	// 获取最新 token
	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	switch c.authScheme {
	case AuthSchemeBearer:
		req.Header.Set("Authorization", "Bearer "+token)
	case AuthSchemeXAPIToken:
		req.Header.Set("X-API-Token", token)
	case AuthSchemeQueryToken:
		q := req.URL.Query()
		q.Set("token", token)
		req.URL.RawQuery = q.Encode()
	}

	return nil
}

// shouldRetry 判断是否应该重试
func (c *Client) shouldRetry(err error) bool {
	_, ok := err.(*RetryableError)
	return ok
}

// RetryableError 可重试错误
type RetryableError struct {
	Err        error
	StatusCode int
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NonRetryableError 不可重试错误
type NonRetryableError struct {
	Err        error
	StatusCode int
	Message    string
}

func (e *NonRetryableError) Error() string {
	return e.Err.Error()
}

func (e *NonRetryableError) Unwrap() error {
	return e.Err
}

// DownloadHistoryItem 下载历史项
type DownloadHistoryItem struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Type         string    `json:"type"`         // "电影" 或 "电视剧"
	Year         int       `json:"year"`
	TMDBID       int       `json:"tmdbid"`       // 注意：MP API 使用小写 tmdbid
	Season       int       `json:"season,omitempty"`
	Episode      int       `json:"episode,omitempty"`
	Status       string    `json:"status"`       // "downloading", "completed", "failed"
	DownloadHash string    `json:"download_hash,omitempty"`
	Torrent      string    `json:"torrent,omitempty"`
	CreatedAt    string    `json:"date"`         // MP 使用 date 字段
}

// TransferHistoryItem 入库历史项
type TransferHistoryItem struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"`          // "电影" 或 "电视剧"
	Year      int    `json:"year"`
	TMDBID    int    `json:"tmdbid"`        // 注意：MP API 使用小写 tmdbid
	Season    int    `json:"season,omitempty"`
	Episode   int    `json:"episode,omitempty"`
	Path      string `json:"path"`
	Dest      string `json:"dest,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Status    string `json:"status"`        // "success", "failed"
	CreatedAt string `json:"date"`          // MP 使用 date 字段
}

// GetDownloadHistory 获取下载历史
// 注意：MoviePilot API 直接返回数组，不是带分页的对象
func (c *Client) GetDownloadHistory(ctx context.Context, page, pageSize int) ([]DownloadHistoryItem, error) {
	// 等待速率限制
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/history/download?page=%d&count=%d", c.baseURL, page, pageSize)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置认证
	if err := c.setAuth(httpReq, ctx); err != nil {
		return nil, err
	}

	httpReq.Header.Set("Accept", "application/json")

	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer httpResp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", httpResp.StatusCode, string(respBody))
	}

	// MP API 直接返回数组
	var items []DownloadHistoryItem
	if err := json.Unmarshal(respBody, &items); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}

	return items, nil
}

// GetTransferHistory 获取入库历史
// 注意：MoviePilot API 直接返回数组，不是带分页的对象
func (c *Client) GetTransferHistory(ctx context.Context, page, pageSize int) ([]TransferHistoryItem, error) {
	// 等待速率限制
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/history/transfer?page=%d&count=%d", c.baseURL, page, pageSize)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置认证
	if err := c.setAuth(httpReq, ctx); err != nil {
		return nil, err
	}

	httpReq.Header.Set("Accept", "application/json")

	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer httpResp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", httpResp.StatusCode, string(respBody))
	}

	// MP API 直接返回数组
	var items []TransferHistoryItem
	if err := json.Unmarshal(respBody, &items); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}

	return items, nil
}
