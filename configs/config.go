package configs

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config 应用配置
type Config struct {
	// Jellyseerr/Overseerr 配置
	JellyURL      string
	JellyAPIKey   string
	JellyFilter   string // approved, pending 等
	JellyPageSize int

	// MoviePilot 配置
	MPURL           string
	MPUsername      string // MoviePilot 用户名（必需）
	MPPassword      string // MoviePilot 密码（必需）
	MPAuthScheme    string // bearer, x-api-token, query-token
	MPRateLimitPS   int    // 每秒请求数限制
	MPDryRun        bool
	MPTVEpisodeMode string // season 或 episode
	MPTokenRefresh  int    // Token 刷新间隔（小时），默认 24 小时

	// 存储配置
	StoreType string // sqlite 或 json
	StorePath string // 存储路径

	// 同步配置
	SyncInterval int  // daemon 模式下同步间隔（分钟）
	EnableRetry  bool // 是否启用重试
	MaxRetries   int  // 最大重试次数

	// 日志配置
	LogLevel string // debug, info, warn, error

	// Telegram 配置
	TelegramEnabled bool
	TelegramToken   string
	TelegramChatIDs []string // 支持多个 chat ID

	// Tracker 配置
	TrackerEnabled       bool
	TrackerCheckInterval int  // 检查间隔（分钟）
	TrackerSSEEnabled    bool // 是否启用 SSE 监听

	// SmartRetry 配置
	SmartRetryEnabled      bool
	SmartRetryMaxAttempts  int // 最大重试次数
	SmartRetryInitialDelay int // 初始延迟（小时）
	SmartRetryCheckInterval int // 检查间隔（小时）

	// Reporter 配置
	ReportEnabled bool
	ReportTime    string // 每日报告时间，格式 HH:MM
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	cfg := &Config{
		// Jellyseerr 默认值
		JellyURL:      getEnv("JELLY_URL", ""),
		JellyAPIKey:   getEnv("JELLY_API_KEY", ""),
		JellyFilter:   getEnv("JELLY_FILTER", "approved"),
		JellyPageSize: getEnvAsInt("JELLY_PAGE_SIZE", 50),

		// MoviePilot 默认值
		MPURL:           getEnv("MP_URL", ""),
		MPUsername:      getEnv("MP_USERNAME", ""),
		MPPassword:      getEnv("MP_PASSWORD", ""),
		MPAuthScheme:    getEnv("MP_AUTH_SCHEME", "bearer"),
		MPRateLimitPS:   getEnvAsInt("MP_RATE_LIMIT_PER_SEC", 3),
		MPDryRun:        getEnvAsBool("MP_DRY_RUN", false),
		MPTVEpisodeMode: getEnv("MP_TV_EPISODE_MODE", "season"),
		MPTokenRefresh:  getEnvAsInt("MP_TOKEN_REFRESH_HOURS", 24),

		// 存储配置
		StoreType: getEnv("STORE_TYPE", "sqlite"),
		StorePath: getEnv("STORE_PATH", "./data/syncer.db"),

		// 同步配置
		SyncInterval: getEnvAsInt("SYNC_INTERVAL", 5),
		EnableRetry:  getEnvAsBool("ENABLE_RETRY", true),
		MaxRetries:   getEnvAsInt("MAX_RETRIES", 3),

		// 日志配置
		LogLevel: getEnv("LOG_LEVEL", "info"),

		// Telegram 配置
		TelegramEnabled: getEnvAsBool("TELEGRAM_ENABLED", false),
		TelegramToken:   getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatIDs: getEnvAsSlice("TELEGRAM_CHAT_IDS", ",", []string{}),

		// Tracker 配置
		TrackerEnabled:       getEnvAsBool("TRACKER_ENABLED", true),
		TrackerCheckInterval: getEnvAsInt("TRACKER_CHECK_INTERVAL", 5),
		TrackerSSEEnabled:    getEnvAsBool("TRACKER_SSE_ENABLED", true),

		// SmartRetry 配置
		SmartRetryEnabled:       getEnvAsBool("SMART_RETRY_ENABLED", true),
		SmartRetryMaxAttempts:   getEnvAsInt("SMART_RETRY_MAX_ATTEMPTS", 3),
		SmartRetryInitialDelay:  getEnvAsInt("SMART_RETRY_INITIAL_DELAY", 24),
		SmartRetryCheckInterval: getEnvAsInt("SMART_RETRY_CHECK_INTERVAL", 1),

		// Reporter 配置
		ReportEnabled: getEnvAsBool("REPORT_ENABLED", true),
		ReportTime:    getEnv("REPORT_TIME", "09:00"),
	}

	// 校验必需配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.JellyURL == "" {
		return fmt.Errorf("JELLY_URL is required")
	}
	if c.JellyAPIKey == "" {
		return fmt.Errorf("JELLY_API_KEY is required")
	}
	if c.MPURL == "" {
		return fmt.Errorf("MP_URL is required")
	}
	// 用户名密码必须提供
	if c.MPUsername == "" {
		return fmt.Errorf("MP_USERNAME is required")
	}
	if c.MPPassword == "" {
		return fmt.Errorf("MP_PASSWORD is required")
	}

	// 规范化 URL（确保以 / 结尾）
	c.JellyURL = strings.TrimRight(c.JellyURL, "/")
	c.MPURL = strings.TrimRight(c.MPURL, "/")

	// 验证认证方案
	validAuthSchemes := []string{"bearer", "x-api-token", "query-token"}
	if !contains(validAuthSchemes, c.MPAuthScheme) {
		return fmt.Errorf("MP_AUTH_SCHEME must be one of: %v", validAuthSchemes)
	}

	// 验证剧集模式
	validEpisodeModes := []string{"season", "episode"}
	if !contains(validEpisodeModes, c.MPTVEpisodeMode) {
		return fmt.Errorf("MP_TV_EPISODE_MODE must be one of: %v", validEpisodeModes)
	}

	// 验证存储类型
	validStoreTypes := []string{"sqlite", "json"}
	if !contains(validStoreTypes, c.StoreType) {
		return fmt.Errorf("STORE_TYPE must be one of: %v", validStoreTypes)
	}

	return nil
}

// MaskSensitive 返回一个屏蔽敏感信息的配置副本，用于日志输出
func (c *Config) MaskSensitive() map[string]interface{} {
	return map[string]interface{}{
		"jelly_url":          c.JellyURL,
		"jelly_api_key":      maskString(c.JellyAPIKey),
		"jelly_filter":       c.JellyFilter,
		"jelly_page_size":    c.JellyPageSize,
		"mp_url":                 c.MPURL,
		"mp_username":            maskString(c.MPUsername),
		"mp_password":            "****",
		"mp_auth_scheme":         c.MPAuthScheme,
		"mp_token_refresh_hours": c.MPTokenRefresh,
		"mp_rate_limit_ps":   c.MPRateLimitPS,
		"mp_dry_run":         c.MPDryRun,
		"mp_tv_episode_mode": c.MPTVEpisodeMode,
		"store_type":         c.StoreType,
		"store_path":         c.StorePath,
		"sync_interval":      c.SyncInterval,
		"enable_retry":       c.EnableRetry,
		"max_retries":        c.MaxRetries,
		"log_level":          c.LogLevel,
	}
}

// 工具函数

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

func getEnvAsSlice(key, separator string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	parts := strings.Split(valueStr, separator)
	// 去除空白项
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}
