package store

import (
	"database/sql"
	"time"
)

// SaveTracking 保存订阅跟踪记录
func (s *SQLiteStore) SaveTracking(tracking *SubscriptionTracking) error {
	now := time.Now()
	tracking.CreatedAt = now
	tracking.UpdatedAt = now

	query := `
		INSERT INTO subscription_tracking (
			source_request_id, tmdb_id, title, media_type, subscribe_status,
			subscribe_time, download_start_time, download_finish_time, transfer_time,
			retry_count, last_retry_time, error_message, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_request_id) DO UPDATE SET
			subscribe_status = excluded.subscribe_status,
			subscribe_time = excluded.subscribe_time,
			download_start_time = excluded.download_start_time,
			download_finish_time = excluded.download_finish_time,
			transfer_time = excluded.transfer_time,
			retry_count = excluded.retry_count,
			last_retry_time = excluded.last_retry_time,
			error_message = excluded.error_message,
			updated_at = excluded.updated_at
	`

	_, err := s.db.Exec(query,
		tracking.SourceRequestID, tracking.TMDBID, tracking.Title, tracking.MediaType,
		tracking.SubscribeStatus, tracking.SubscribeTime, tracking.DownloadStartTime,
		tracking.DownloadFinishTime, tracking.TransferTime, tracking.RetryCount,
		tracking.LastRetryTime, tracking.ErrorMessage, tracking.CreatedAt, tracking.UpdatedAt,
	)
	return err
}

// GetTracking 获取订阅跟踪记录
func (s *SQLiteStore) GetTracking(sourceRequestID string) (*SubscriptionTracking, error) {
	query := `
		SELECT id, source_request_id, tmdb_id, title, media_type, subscribe_status,
			subscribe_time, download_start_time, download_finish_time, transfer_time,
			retry_count, last_retry_time, error_message, created_at, updated_at
		FROM subscription_tracking
		WHERE source_request_id = ?
	`

	tracking := &SubscriptionTracking{}
	err := s.db.QueryRow(query, sourceRequestID).Scan(
		&tracking.ID, &tracking.SourceRequestID, &tracking.TMDBID, &tracking.Title,
		&tracking.MediaType, &tracking.SubscribeStatus, &tracking.SubscribeTime,
		&tracking.DownloadStartTime, &tracking.DownloadFinishTime, &tracking.TransferTime,
		&tracking.RetryCount, &tracking.LastRetryTime, &tracking.ErrorMessage,
		&tracking.CreatedAt, &tracking.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tracking, nil
}

// UpdateTracking 更新订阅跟踪记录
func (s *SQLiteStore) UpdateTracking(tracking *SubscriptionTracking) error {
	tracking.UpdatedAt = time.Now()

	query := `
		UPDATE subscription_tracking SET
			subscribe_status = ?, subscribe_time = ?, download_start_time = ?,
			download_finish_time = ?, transfer_time = ?, retry_count = ?,
			last_retry_time = ?, error_message = ?, updated_at = ?
		WHERE source_request_id = ?
	`

	_, err := s.db.Exec(query,
		tracking.SubscribeStatus, tracking.SubscribeTime, tracking.DownloadStartTime,
		tracking.DownloadFinishTime, tracking.TransferTime, tracking.RetryCount,
		tracking.LastRetryTime, tracking.ErrorMessage, tracking.UpdatedAt,
		tracking.SourceRequestID,
	)
	return err
}

// ListTrackingByStatus 根据状态列出跟踪记录
func (s *SQLiteStore) ListTrackingByStatus(status TrackingStatus, limit int) ([]*SubscriptionTracking, error) {
	query := `
		SELECT id, source_request_id, tmdb_id, title, media_type, subscribe_status,
			subscribe_time, download_start_time, download_finish_time, transfer_time,
			retry_count, last_retry_time, error_message, created_at, updated_at
		FROM subscription_tracking
		WHERE subscribe_status = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trackings []*SubscriptionTracking
	for rows.Next() {
		tracking := &SubscriptionTracking{}
		err := rows.Scan(
			&tracking.ID, &tracking.SourceRequestID, &tracking.TMDBID, &tracking.Title,
			&tracking.MediaType, &tracking.SubscribeStatus, &tracking.SubscribeTime,
			&tracking.DownloadStartTime, &tracking.DownloadFinishTime, &tracking.TransferTime,
			&tracking.RetryCount, &tracking.LastRetryTime, &tracking.ErrorMessage,
			&tracking.CreatedAt, &tracking.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		trackings = append(trackings, tracking)
	}

	return trackings, rows.Err()
}

// SaveEvent 保存下载事件
func (s *SQLiteStore) SaveEvent(event *DownloadEvent) error {
	event.CreatedAt = time.Now()

	query := `
		INSERT INTO download_events (source_request_id, event_type, event_data, created_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, event.SourceRequestID, event.EventType, event.EventData, event.CreatedAt)
	return err
}

// ListEvents 列出指定请求的事件
func (s *SQLiteStore) ListEvents(sourceRequestID string, limit int) ([]*DownloadEvent, error) {
	query := `
		SELECT id, source_request_id, event_type, event_data, created_at
		FROM download_events
		WHERE source_request_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, sourceRequestID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*DownloadEvent
	for rows.Next() {
		event := &DownloadEvent{}
		err := rows.Scan(&event.ID, &event.SourceRequestID, &event.EventType, &event.EventData, &event.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// SaveReport 保存每日报告
func (s *SQLiteStore) SaveReport(report *DailyReport) error {
	report.CreatedAt = time.Now()

	query := `
		INSERT INTO daily_reports (report_date, total_subscribed, total_downloaded, total_transferred, total_failed, report_content, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(report_date) DO UPDATE SET
			total_subscribed = excluded.total_subscribed,
			total_downloaded = excluded.total_downloaded,
			total_transferred = excluded.total_transferred,
			total_failed = excluded.total_failed,
			report_content = excluded.report_content,
			created_at = excluded.created_at
	`

	_, err := s.db.Exec(query,
		report.ReportDate, report.TotalSubscribed, report.TotalDownloaded,
		report.TotalTransferred, report.TotalFailed, report.ReportContent, report.CreatedAt,
	)
	return err
}

// GetReport 获取指定日期的报告
func (s *SQLiteStore) GetReport(reportDate string) (*DailyReport, error) {
	query := `
		SELECT id, report_date, total_subscribed, total_downloaded, total_transferred, total_failed, report_content, created_at
		FROM daily_reports
		WHERE report_date = ?
	`

	report := &DailyReport{}
	err := s.db.QueryRow(query, reportDate).Scan(
		&report.ID, &report.ReportDate, &report.TotalSubscribed, &report.TotalDownloaded,
		&report.TotalTransferred, &report.TotalFailed, &report.ReportContent, &report.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return report, nil
}

// ListRecentReports 列出最近N天的报告
func (s *SQLiteStore) ListRecentReports(days int) ([]*DailyReport, error) {
	query := `
		SELECT id, report_date, total_subscribed, total_downloaded, total_transferred, total_failed, report_content, created_at
		FROM daily_reports
		ORDER BY report_date DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*DailyReport
	for rows.Next() {
		report := &DailyReport{}
		err := rows.Scan(
			&report.ID, &report.ReportDate, &report.TotalSubscribed, &report.TotalDownloaded,
			&report.TotalTransferred, &report.TotalFailed, &report.ReportContent, &report.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, rows.Err()
}
