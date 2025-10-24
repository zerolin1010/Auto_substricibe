package jelly

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client Jellyseerr/Overseerr API 客户端
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient 创建客户端
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListRequests 列出请求（带分页）
func (c *Client) ListRequests(ctx context.Context, filter string, take, skip int) (*RequestsResponse, error) {
	// 构建 URL
	u, err := url.Parse(c.baseURL + "/api/v1/request")
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	// 添加查询参数
	q := u.Query()
	q.Set("take", strconv.Itoa(take))
	q.Set("skip", strconv.Itoa(skip))
	if filter != "" {
		q.Set("filter", filter)
	}
	// 按请求时间排序
	q.Set("sort", "added")
	u.RawQuery = q.Encode()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置认证头
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response RequestsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &response, nil
}

// GetRequest 获取单个请求详情
func (c *Client) GetRequest(ctx context.Context, requestID int) (*MediaRequestV2, error) {
	// 构建 URL
	u := fmt.Sprintf("%s/api/v1/request/%d", c.baseURL, requestID)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置认证头
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("request not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var request MediaRequestV2
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &request, nil
}

// GetMediaDetails 获取媒体详情（含标题等）
func (c *Client) GetMediaDetails(ctx context.Context, mediaType string, tmdbID int) (*MediaDetails, error) {
	// 构建 URL
	u := fmt.Sprintf("%s/api/v1/%s/%d", c.baseURL, mediaType, tmdbID)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置认证头
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("media not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var details MediaDetails
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &details, nil
}

// MediaDetails 媒体详情
type MediaDetails struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`          // 剧集名称
	OriginalName  string `json:"originalName"`  // 原始名称
	Title         string `json:"title"`         // 电影标题
	OriginalTitle string `json:"originalTitle"` // 原始标题
	Overview      string `json:"overview"`
	PosterPath    string `json:"posterPath"`
	BackdropPath  string `json:"backdropPath"`
	MediaType     string `json:"mediaType"`
}

// GetTitle 获取标题
func (m *MediaDetails) GetTitle() string {
	if m.Title != "" {
		return m.Title
	}
	if m.Name != "" {
		return m.Name
	}
	if m.OriginalTitle != "" {
		return m.OriginalTitle
	}
	return m.OriginalName
}

// FetchAllApprovedRequests 获取所有已批准的请求（自动分页）
func (c *Client) FetchAllApprovedRequests(ctx context.Context, pageSize int) ([]*MediaRequestV2, error) {
	var allRequests []*MediaRequestV2
	skip := 0

	for {
		resp, err := c.ListRequests(ctx, "approved", pageSize, skip)
		if err != nil {
			return nil, fmt.Errorf("list requests (skip=%d): %w", skip, err)
		}

		// 本地二次过滤（确保是已批准状态）
		for _, req := range resp.Results {
			if req.Status.IsApproved() {
				allRequests = append(allRequests, &req)
			}
		}

		// 检查是否还有更多数据
		if len(resp.Results) < pageSize {
			break
		}

		skip += pageSize

		// 防止无限循环
		if skip > 10000 {
			return nil, fmt.Errorf("too many requests (> 10000), possible infinite loop")
		}
	}

	return allRequests, nil
}
