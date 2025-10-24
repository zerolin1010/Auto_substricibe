package mp

// SubscribeRequest 订阅请求
type SubscribeRequest struct {
	Name        string `json:"name,omitempty"`         // 媒体名称
	Year        int    `json:"year,omitempty"`         // 年份
	Type        string `json:"type"`                   // 类型：电影 movie 或 电视剧 tv
	TMDBID      int    `json:"tmdbid"`                 // TMDB ID
	Season      int    `json:"season,omitempty"`       // 季号（剧集）
	Episodes    []int  `json:"episodes,omitempty"`     // 集号列表（剧集）
	Username    string `json:"username,omitempty"`     // 用户名
	BestVersion bool   `json:"best_version,omitempty"` // 洗版
	ExistOk     bool   `json:"exist_ok,omitempty"`     // 已存在时是否继续
}

// SubscribeResponse 订阅响应
type SubscribeResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message,omitempty"`
	Data    *SubscribeData `json:"data,omitempty"`
	Code    int            `json:"code,omitempty"`
}

// SubscribeData 订阅数据
type SubscribeData struct {
	ID          int `json:"id,omitempty"` // MoviePilot 返回的是数字
	SubscribeID int `json:"subscribe_id,omitempty"`
}

// MediaSearchRequest 媒体搜索请求
type MediaSearchRequest struct {
	Title  string `json:"title"`
	Type   string `json:"type,omitempty"` // movie 或 tv
	Year   int    `json:"year,omitempty"`
	TMDBID int    `json:"tmdbid,omitempty"`
}

// MediaSearchResponse 媒体搜索响应
type MediaSearchResponse struct {
	Success bool          `json:"success"`
	Data    []MediaResult `json:"data,omitempty"`
	Message string        `json:"message,omitempty"`
}

// MediaResult 媒体搜索结果
type MediaResult struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	Year          int    `json:"year"`
	Type          string `json:"type"` // movie 或 tv
	TMDBID        int    `json:"tmdbid"`
	Overview      string `json:"overview"`
	PosterPath    string `json:"poster_path"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}
