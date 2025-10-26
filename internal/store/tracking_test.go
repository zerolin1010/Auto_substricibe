package store

import (
	"os"
	"testing"
	"time"
)

func TestTrackingCRUD(t *testing.T) {
	// 创建临时数据库
	dbPath := "/tmp/test_tracking.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// 测试保存跟踪记录
	tracking := &SubscriptionTracking{
		SourceRequestID: "test-123",
		TMDBID:          123,
		Title:           "Test Movie",
		MediaType:       MediaTypeMovie,
		SubscribeStatus: TrackingPending,
		RetryCount:      0,
	}

	if err := store.SaveTracking(tracking); err != nil {
		t.Fatalf("Failed to save tracking: %v", err)
	}

	// 测试获取跟踪记录
	got, err := store.GetTracking("test-123")
	if err != nil {
		t.Fatalf("Failed to get tracking: %v", err)
	}
	if got == nil {
		t.Fatal("Expected tracking record, got nil")
	}
	if got.Title != "Test Movie" {
		t.Errorf("Expected title 'Test Movie', got '%s'", got.Title)
	}

	// 测试更新跟踪记录
	now := time.Now()
	got.SubscribeStatus = TrackingSubscribed
	got.SubscribeTime = &now
	if err := store.UpdateTracking(got); err != nil {
		t.Fatalf("Failed to update tracking: %v", err)
	}

	// 验证更新
	updated, err := store.GetTracking("test-123")
	if err != nil {
		t.Fatalf("Failed to get updated tracking: %v", err)
	}
	if updated.SubscribeStatus != TrackingSubscribed {
		t.Errorf("Expected status 'subscribed', got '%s'", updated.SubscribeStatus)
	}

	// 测试按状态列出
	trackings, err := store.ListTrackingByStatus(TrackingSubscribed, 10)
	if err != nil {
		t.Fatalf("Failed to list trackings: %v", err)
	}
	if len(trackings) != 1 {
		t.Errorf("Expected 1 tracking, got %d", len(trackings))
	}

	t.Log("✅ Tracking CRUD test passed")
}

func TestEventCRUD(t *testing.T) {
	// 创建临时数据库
	dbPath := "/tmp/test_events.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// 测试保存事件
	event := &DownloadEvent{
		SourceRequestID: "test-456",
		EventType:       EventSubscribed,
		EventData:       `{"status": "success"}`,
	}

	if err := store.SaveEvent(event); err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}

	// 测试列出事件
	events, err := store.ListEvents("test-456", 10)
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
	if events[0].EventType != EventSubscribed {
		t.Errorf("Expected event type 'subscribed', got '%s'", events[0].EventType)
	}

	t.Log("✅ Event CRUD test passed")
}

func TestReportCRUD(t *testing.T) {
	// 创建临时数据库
	dbPath := "/tmp/test_reports.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// 测试保存报告
	report := &DailyReport{
		ReportDate:       "2025-01-20",
		TotalSubscribed:  10,
		TotalDownloaded:  8,
		TotalTransferred: 7,
		TotalFailed:      1,
		ReportContent:    `{"summary": "test"}`,
	}

	if err := store.SaveReport(report); err != nil {
		t.Fatalf("Failed to save report: %v", err)
	}

	// 测试获取报告
	got, err := store.GetReport("2025-01-20")
	if err != nil {
		t.Fatalf("Failed to get report: %v", err)
	}
	if got == nil {
		t.Fatal("Expected report, got nil")
	}
	if got.TotalSubscribed != 10 {
		t.Errorf("Expected 10 subscribed, got %d", got.TotalSubscribed)
	}

	// 测试列出最近报告
	reports, err := store.ListRecentReports(7)
	if err != nil {
		t.Fatalf("Failed to list reports: %v", err)
	}
	if len(reports) != 1 {
		t.Errorf("Expected 1 report, got %d", len(reports))
	}

	t.Log("✅ Report CRUD test passed")
}
