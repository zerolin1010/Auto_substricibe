package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store 存储接口
type Store interface {
	// Request 相关
	SaveRequest(req *Request) error
	GetRequest(sourceRequestID string) (*Request, error)
	ListPendingRequests(limit int) ([]*Request, error)
	UpdateRequestStatus(sourceRequestID string, status SyncStatus) error

	// MPLink 相关
	SaveMPLink(link *MPLink) error
	GetMPLink(sourceRequestID string) (*MPLink, error)
	UpdateMPLink(link *MPLink) error
	ListFailedLinks(limit int) ([]*MPLink, error)

	// SubscriptionTracking 相关
	SaveTracking(tracking *SubscriptionTracking) error
	GetTracking(sourceRequestID string) (*SubscriptionTracking, error)
	UpdateTracking(tracking *SubscriptionTracking) error
	ListTrackingByStatus(status TrackingStatus, limit int) ([]*SubscriptionTracking, error)

	// DownloadEvent 相关
	SaveEvent(event *DownloadEvent) error
	ListEvents(sourceRequestID string, limit int) ([]*DownloadEvent, error)

	// DailyReport 相关
	SaveReport(report *DailyReport) error
	GetReport(reportDate string) (*DailyReport, error)
	ListRecentReports(days int) ([]*DailyReport, error)

	// 统计
	GetStats() (*Stats, error)

	Close() error
}

// Stats 统计信息
type Stats struct {
	TotalRequests   int
	PendingRequests int
	SyncedRequests  int
	FailedRequests  int
}

// SQLiteStore SQLite 存储实现
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore 创建 SQLite 存储
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(1) // SQLite 建议单连接
	db.SetMaxIdleConns(1)

	store := &SQLiteStore{db: db}

	// 初始化表
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return store, nil
}

// migrate 数据库迁移
func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_request_id TEXT NOT NULL UNIQUE,
		media_type TEXT NOT NULL,
		tmdb_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		poster_path TEXT,
		seasons_json TEXT,
		episodes_json TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		requested_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status);
	CREATE INDEX IF NOT EXISTS idx_requests_source_id ON requests(source_request_id);

	CREATE TABLE IF NOT EXISTS mp_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_request_id TEXT NOT NULL UNIQUE,
		mp_subscribe_id TEXT,
		state TEXT NOT NULL DEFAULT 'pending',
		last_error TEXT,
		retry_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (source_request_id) REFERENCES requests(source_request_id)
	);

	CREATE INDEX IF NOT EXISTS idx_mp_links_state ON mp_links(state);
	CREATE INDEX IF NOT EXISTS idx_mp_links_source_id ON mp_links(source_request_id);

	-- 订阅跟踪表
	CREATE TABLE IF NOT EXISTS subscription_tracking (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_request_id TEXT NOT NULL UNIQUE,
		tmdb_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		media_type TEXT NOT NULL,
		subscribe_status TEXT NOT NULL DEFAULT 'pending',
		subscribe_time DATETIME,
		download_start_time DATETIME,
		download_finish_time DATETIME,
		transfer_time DATETIME,
		retry_count INTEGER NOT NULL DEFAULT 0,
		last_retry_time DATETIME,
		error_message TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_tracking_status ON subscription_tracking(subscribe_status);
	CREATE INDEX IF NOT EXISTS idx_tracking_source_id ON subscription_tracking(source_request_id);
	CREATE INDEX IF NOT EXISTS idx_tracking_created_at ON subscription_tracking(created_at);

	-- 下载事件表
	CREATE TABLE IF NOT EXISTS download_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_request_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		event_data TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_events_source_id ON download_events(source_request_id);
	CREATE INDEX IF NOT EXISTS idx_events_type ON download_events(event_type);
	CREATE INDEX IF NOT EXISTS idx_events_created_at ON download_events(created_at);

	-- 每日报告表
	CREATE TABLE IF NOT EXISTS daily_reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		report_date TEXT NOT NULL UNIQUE,
		total_subscribed INTEGER NOT NULL DEFAULT 0,
		total_downloaded INTEGER NOT NULL DEFAULT 0,
		total_transferred INTEGER NOT NULL DEFAULT 0,
		total_failed INTEGER NOT NULL DEFAULT 0,
		report_content TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_reports_date ON daily_reports(report_date);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// 迁移：为已存在的 requests 表添加 poster_path 列
	// 使用 PRAGMA 检查列是否存在
	var hasPosterPath bool
	rows, err := s.db.Query("PRAGMA table_info(requests)")
	if err != nil {
		return fmt.Errorf("check table schema: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table info: %w", err)
		}
		if name == "poster_path" {
			hasPosterPath = true
			break
		}
	}

	// 如果不存在 poster_path 列，添加它
	if !hasPosterPath {
		if _, err := s.db.Exec("ALTER TABLE requests ADD COLUMN poster_path TEXT"); err != nil {
			return fmt.Errorf("add poster_path column: %w", err)
		}
	}

	return nil
}

// SaveRequest 保存请求
func (s *SQLiteStore) SaveRequest(req *Request) error {
	now := time.Now()
	req.CreatedAt = now
	req.UpdatedAt = now

	query := `
		INSERT INTO requests (source_request_id, media_type, tmdb_id, title, poster_path, seasons_json, episodes_json, status, requested_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_request_id) DO UPDATE SET
			media_type = excluded.media_type,
			tmdb_id = excluded.tmdb_id,
			title = excluded.title,
			poster_path = excluded.poster_path,
			seasons_json = excluded.seasons_json,
			episodes_json = excluded.episodes_json,
			status = excluded.status,
			requested_at = excluded.requested_at,
			updated_at = excluded.updated_at
	`

	result, err := s.db.Exec(query,
		req.SourceRequestID, req.MediaType, req.TMDBID, req.Title, req.PosterPath,
		req.SeasonsJSON, req.EpisodesJSON, req.Status, req.RequestedAt,
		req.CreatedAt, req.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if req.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			req.ID = id
		}
	}

	return nil
}

// GetRequest 获取请求
func (s *SQLiteStore) GetRequest(sourceRequestID string) (*Request, error) {
	query := `
		SELECT id, source_request_id, media_type, tmdb_id, title, poster_path, seasons_json, episodes_json, status, requested_at, created_at, updated_at
		FROM requests
		WHERE source_request_id = ?
	`

	req := &Request{}
	var posterPath sql.NullString
	err := s.db.QueryRow(query, sourceRequestID).Scan(
		&req.ID, &req.SourceRequestID, &req.MediaType, &req.TMDBID, &req.Title, &posterPath,
		&req.SeasonsJSON, &req.EpisodesJSON, &req.Status, &req.RequestedAt,
		&req.CreatedAt, &req.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// 处理 NULL 值
	if posterPath.Valid {
		req.PosterPath = posterPath.String
	}

	return req, nil
}

// ListPendingRequests 列出待处理请求
func (s *SQLiteStore) ListPendingRequests(limit int) ([]*Request, error) {
	query := `
		SELECT id, source_request_id, media_type, tmdb_id, title, poster_path, seasons_json, episodes_json, status, requested_at, created_at, updated_at
		FROM requests
		WHERE status = 'pending' OR status = 'retrying'
		ORDER BY requested_at ASC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*Request
	for rows.Next() {
		req := &Request{}
		var posterPath sql.NullString
		if err := rows.Scan(
			&req.ID, &req.SourceRequestID, &req.MediaType, &req.TMDBID, &req.Title, &posterPath,
			&req.SeasonsJSON, &req.EpisodesJSON, &req.Status, &req.RequestedAt,
			&req.CreatedAt, &req.UpdatedAt,
		); err != nil {
			return nil, err
		}
		// 处理 NULL 值
		if posterPath.Valid {
			req.PosterPath = posterPath.String
		}
		requests = append(requests, req)
	}

	return requests, rows.Err()
}

// UpdateRequestStatus 更新请求状态
func (s *SQLiteStore) UpdateRequestStatus(sourceRequestID string, status SyncStatus) error {
	query := `
		UPDATE requests
		SET status = ?, updated_at = ?
		WHERE source_request_id = ?
	`

	_, err := s.db.Exec(query, status, time.Now(), sourceRequestID)
	return err
}

// SaveMPLink 保存 MoviePilot 链接
func (s *SQLiteStore) SaveMPLink(link *MPLink) error {
	now := time.Now()
	link.CreatedAt = now
	link.UpdatedAt = now

	query := `
		INSERT INTO mp_links (source_request_id, mp_subscribe_id, state, last_error, retry_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_request_id) DO UPDATE SET
			mp_subscribe_id = excluded.mp_subscribe_id,
			state = excluded.state,
			last_error = excluded.last_error,
			retry_count = excluded.retry_count,
			updated_at = excluded.updated_at
	`

	result, err := s.db.Exec(query,
		link.SourceRequestID, link.MPSubscribeID, link.State, link.LastError,
		link.RetryCount, link.CreatedAt, link.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if link.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			link.ID = id
		}
	}

	return nil
}

// GetMPLink 获取 MoviePilot 链接
func (s *SQLiteStore) GetMPLink(sourceRequestID string) (*MPLink, error) {
	query := `
		SELECT id, source_request_id, mp_subscribe_id, state, last_error, retry_count, created_at, updated_at
		FROM mp_links
		WHERE source_request_id = ?
	`

	link := &MPLink{}
	err := s.db.QueryRow(query, sourceRequestID).Scan(
		&link.ID, &link.SourceRequestID, &link.MPSubscribeID, &link.State,
		&link.LastError, &link.RetryCount, &link.CreatedAt, &link.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return link, nil
}

// UpdateMPLink 更新 MoviePilot 链接
func (s *SQLiteStore) UpdateMPLink(link *MPLink) error {
	link.UpdatedAt = time.Now()

	query := `
		UPDATE mp_links
		SET mp_subscribe_id = ?, state = ?, last_error = ?, retry_count = ?, updated_at = ?
		WHERE source_request_id = ?
	`

	_, err := s.db.Exec(query,
		link.MPSubscribeID, link.State, link.LastError, link.RetryCount,
		link.UpdatedAt, link.SourceRequestID,
	)
	return err
}

// ListFailedLinks 列出失败的链接
func (s *SQLiteStore) ListFailedLinks(limit int) ([]*MPLink, error) {
	query := `
		SELECT id, source_request_id, mp_subscribe_id, state, last_error, retry_count, created_at, updated_at
		FROM mp_links
		WHERE state = 'failed' OR state = 'retrying'
		ORDER BY updated_at ASC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*MPLink
	for rows.Next() {
		link := &MPLink{}
		if err := rows.Scan(
			&link.ID, &link.SourceRequestID, &link.MPSubscribeID, &link.State,
			&link.LastError, &link.RetryCount, &link.CreatedAt, &link.UpdatedAt,
		); err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// GetStats 获取统计信息
func (s *SQLiteStore) GetStats() (*Stats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' OR status = 'retrying' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'synced' THEN 1 ELSE 0 END) as synced,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
		FROM requests
	`

	stats := &Stats{}
	err := s.db.QueryRow(query).Scan(
		&stats.TotalRequests, &stats.PendingRequests,
		&stats.SyncedRequests, &stats.FailedRequests,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// Close 关闭数据库连接
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
