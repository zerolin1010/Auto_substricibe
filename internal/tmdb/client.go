package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client TMDB API 客户端
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient 创建 TMDB 客户端
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		return nil // 如果没有配置 API key，返回 nil
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.themoviedb.org/3",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// MovieDetails 电影详情
type MovieDetails struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	OriginalTitle string `json:"original_title"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
	Overview     string `json:"overview"`
	ReleaseDate  string `json:"release_date"`
}

// TVDetails 剧集详情
type TVDetails struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	OriginalName string `json:"original_name"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
	Overview     string `json:"overview"`
	FirstAirDate string `json:"first_air_date"`
}

// GetMovieDetails 获取电影详情
func (c *Client) GetMovieDetails(ctx context.Context, movieID int) (*MovieDetails, error) {
	if c == nil {
		return nil, fmt.Errorf("tmdb client not initialized")
	}

	url := fmt.Sprintf("%s/movie/%d?api_key=%s&language=zh-CN", c.baseURL, movieID, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var details MovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &details, nil
}

// GetTVDetails 获取剧集详情
func (c *Client) GetTVDetails(ctx context.Context, tvID int) (*TVDetails, error) {
	if c == nil {
		return nil, fmt.Errorf("tmdb client not initialized")
	}

	url := fmt.Sprintf("%s/tv/%d?api_key=%s&language=zh-CN", c.baseURL, tvID, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var details TVDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &details, nil
}

// GetPosterPath 根据媒体类型获取海报路径
func (c *Client) GetPosterPath(ctx context.Context, mediaType string, tmdbID int) (string, error) {
	if c == nil {
		return "", nil // 没有配置 TMDB API，返回空字符串
	}

	if mediaType == "movie" {
		details, err := c.GetMovieDetails(ctx, tmdbID)
		if err != nil {
			return "", err
		}
		return details.PosterPath, nil
	} else if mediaType == "tv" {
		details, err := c.GetTVDetails(ctx, tmdbID)
		if err != nil {
			return "", err
		}
		return details.PosterPath, nil
	}

	return "", fmt.Errorf("unknown media type: %s", mediaType)
}
