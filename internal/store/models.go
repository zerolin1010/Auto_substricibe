package store

import (
	"encoding/json"
	"time"
)

// MediaType 媒体类型
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

// SyncStatus 同步状态
type SyncStatus string

const (
	StatusPending    SyncStatus = "pending"    // 待同步
	StatusProcessing SyncStatus = "processing" // 同步中
	StatusSynced     SyncStatus = "synced"     // 已同步
	StatusFailed     SyncStatus = "failed"     // 同步失败
	StatusRetrying   SyncStatus = "retrying"   // 重试中
)

// Request 存储在本地的请求记录
type Request struct {
	ID              int64      `json:"id"`
	SourceRequestID string     `json:"source_request_id"` // Jellyseerr 请求 ID
	MediaType       MediaType  `json:"media_type"`
	TMDBID          int        `json:"tmdb_id"`
	Title           string     `json:"title"`
	SeasonsJSON     string     `json:"seasons_json"`  // JSON 数组，如 [1,2,3]
	EpisodesJSON    string     `json:"episodes_json"` // JSON 对象，如 {"1":[1,2,3]}
	Status          SyncStatus `json:"status"`
	RequestedAt     time.Time  `json:"requested_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// GetSeasons 解析季数组
func (r *Request) GetSeasons() ([]int, error) {
	if r.SeasonsJSON == "" {
		return []int{}, nil
	}
	var seasons []int
	if err := json.Unmarshal([]byte(r.SeasonsJSON), &seasons); err != nil {
		return nil, err
	}
	return seasons, nil
}

// SetSeasons 设置季数组
func (r *Request) SetSeasons(seasons []int) error {
	data, err := json.Marshal(seasons)
	if err != nil {
		return err
	}
	r.SeasonsJSON = string(data)
	return nil
}

// GetEpisodes 解析剧集映射 (season -> episodes)
func (r *Request) GetEpisodes() (map[int][]int, error) {
	if r.EpisodesJSON == "" {
		return map[int][]int{}, nil
	}
	var episodes map[int][]int
	if err := json.Unmarshal([]byte(r.EpisodesJSON), &episodes); err != nil {
		return nil, err
	}
	return episodes, nil
}

// SetEpisodes 设置剧集映射
func (r *Request) SetEpisodes(episodes map[int][]int) error {
	data, err := json.Marshal(episodes)
	if err != nil {
		return err
	}
	r.EpisodesJSON = string(data)
	return nil
}

// MPLink MoviePilot 订阅链接记录
type MPLink struct {
	ID              int64      `json:"id"`
	SourceRequestID string     `json:"source_request_id"` // 对应 Request.SourceRequestID
	MPSubscribeID   string     `json:"mp_subscribe_id"`   // MoviePilot 返回的订阅 ID
	State           SyncStatus `json:"state"`
	LastError       string     `json:"last_error"`  // 最后一次错误信息（不含敏感信息）
	RetryCount      int        `json:"retry_count"` // 重试次数
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
