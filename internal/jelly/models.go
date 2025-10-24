package jelly

import "time"

// RequestsResponse API 响应
type RequestsResponse struct {
	PageInfo PageInfo         `json:"pageInfo"`
	Results  []MediaRequestV2 `json:"results"`
}

// PageInfo 分页信息
type PageInfo struct {
	Pages   int `json:"pages"`
	Results int `json:"results"`
	Page    int `json:"page"`
}

// MediaRequestV2 媒体请求（兼容 Jellyseerr/Overseerr）
type MediaRequestV2 struct {
	ID            int             `json:"id"`
	Status        RequestStatus   `json:"status"`
	Media         MediaInfo       `json:"media"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
	RequestedBy   RequestedByInfo `json:"requestedBy,omitempty"`
	ModifiedBy    RequestedByInfo `json:"modifiedBy,omitempty"`
	IsAutoRequest bool            `json:"is4k,omitempty"`    // 4K 请求标识
	Seasons       []SeasonRequest `json:"seasons,omitempty"` // 剧集季请求
}

// RequestStatus 请求状态
type RequestStatus int

const (
	StatusPendingApproval RequestStatus = 1
	StatusApproved        RequestStatus = 2
	StatusDeclined        RequestStatus = 3
)

// String 返回状态字符串
func (s RequestStatus) String() string {
	switch s {
	case StatusPendingApproval:
		return "pending"
	case StatusApproved:
		return "approved"
	case StatusDeclined:
		return "declined"
	default:
		return "unknown"
	}
}

// IsApproved 是否已批准
func (s RequestStatus) IsApproved() bool {
	return s == StatusApproved
}

// MediaInfo 媒体信息
type MediaInfo struct {
	ID                  int    `json:"id"`
	TMDBID              int    `json:"tmdbId"`
	TVDBId              int    `json:"tvdbId,omitempty"`
	Status              int    `json:"status"`
	MediaType           string `json:"mediaType"` // movie 或 tv
	ExternalServiceID   int    `json:"externalServiceId,omitempty"`
	ExternalServiceSlug string `json:"externalServiceSlug,omitempty"`
}

// RequestedByInfo 请求人信息
type RequestedByInfo struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// SeasonRequest 季请求
type SeasonRequest struct {
	ID           int              `json:"id"`
	SeasonNumber int              `json:"seasonNumber"`
	Status       int              `json:"status"`
	Episodes     []EpisodeRequest `json:"episodes,omitempty"` // 集请求（部分版本支持）
}

// EpisodeRequest 集请求
type EpisodeRequest struct {
	ID            int `json:"id"`
	EpisodeNumber int `json:"episodeNumber"`
	Status        int `json:"status"`
}

// GetTitle 获取标题（需要从详情 API 获取，此处为简化）
func (m *MediaRequestV2) GetTitle() string {
	// 实际场景可能需要再查询详情接口获取标题
	// 这里返回空字符串，由调用方补充
	return ""
}

// IsMovie 是否为电影
func (m *MediaRequestV2) IsMovie() bool {
	return m.Media.MediaType == "movie"
}

// IsTV 是否为剧集
func (m *MediaRequestV2) IsTV() bool {
	return m.Media.MediaType == "tv"
}
