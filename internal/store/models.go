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

// TrackingStatus 订阅跟踪状态
type TrackingStatus string

const (
	TrackingPending      TrackingStatus = "pending"      // 待订阅
	TrackingSubscribed   TrackingStatus = "subscribed"   // 已订阅
	TrackingDownloading  TrackingStatus = "downloading"  // 下载中
	TrackingDownloaded   TrackingStatus = "downloaded"   // 下载完成
	TrackingTransferred  TrackingStatus = "transferred"  // 已入库
	TrackingFailed       TrackingStatus = "failed"       // 失败
	TrackingManualSearch TrackingStatus = "manual_search" // 手动搜索
)

// SubscriptionTracking 订阅跟踪记录
type SubscriptionTracking struct {
	ID                 int64          `json:"id"`
	SourceRequestID    string         `json:"source_request_id"`
	TMDBID             int            `json:"tmdb_id"`
	Title              string         `json:"title"`
	MediaType          MediaType      `json:"media_type"`
	SubscribeStatus    TrackingStatus `json:"subscribe_status"`
	SubscribeTime      *time.Time     `json:"subscribe_time,omitempty"`
	DownloadStartTime  *time.Time     `json:"download_start_time,omitempty"`
	DownloadFinishTime *time.Time     `json:"download_finish_time,omitempty"`
	TransferTime       *time.Time     `json:"transfer_time,omitempty"`
	RetryCount         int            `json:"retry_count"`
	LastRetryTime      *time.Time     `json:"last_retry_time,omitempty"`
	ErrorMessage       string         `json:"error_message,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// EventType 事件类型
type EventType string

const (
	EventSubscribed       EventType = "subscribed"        // 订阅成功
	EventDownloadStarted  EventType = "download_started"  // 开始下载
	EventDownloadComplete EventType = "download_complete" // 下载完成
	EventTransferComplete EventType = "transfer_complete" // 入库完成
	EventFailed           EventType = "failed"            // 失败
	EventManualSearch     EventType = "manual_search"     // 手动搜索
)

// DownloadEvent 下载事件记录
type DownloadEvent struct {
	ID              int64     `json:"id"`
	SourceRequestID string    `json:"source_request_id"`
	EventType       EventType `json:"event_type"`
	EventData       string    `json:"event_data"` // JSON 格式的事件数据
	CreatedAt       time.Time `json:"created_at"`
}

// DailyReport 每日报告
type DailyReport struct {
	ID               int64     `json:"id"`
	ReportDate       string    `json:"report_date"` // YYYY-MM-DD 格式
	TotalSubscribed  int       `json:"total_subscribed"`
	TotalDownloaded  int       `json:"total_downloaded"`
	TotalTransferred int       `json:"total_transferred"`
	TotalFailed      int       `json:"total_failed"`
	ReportContent    string    `json:"report_content"` // JSON 格式的详细报告
	CreatedAt        time.Time `json:"created_at"`
}
