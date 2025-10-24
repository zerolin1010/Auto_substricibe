package store

import (
	"os"
	"testing"
	"time"
)

func TestNewSQLiteStore(t *testing.T) {
	// 使用临时数据库
	dbPath := "./test.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("store should not be nil")
	}
}

func TestSaveAndGetRequest(t *testing.T) {
	dbPath := "./test_save.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	req := &Request{
		SourceRequestID: "test-123",
		MediaType:       MediaTypeMovie,
		TMDBID:          550,
		Title:           "Fight Club",
		Status:          StatusPending,
		RequestedAt:     time.Now(),
	}

	// 保存
	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}

	// 获取
	got, err := store.GetRequest("test-123")
	if err != nil {
		t.Fatalf("GetRequest() error = %v", err)
	}

	if got == nil {
		t.Fatal("GetRequest() returned nil")
	}

	if got.SourceRequestID != req.SourceRequestID {
		t.Errorf("SourceRequestID = %v, want %v", got.SourceRequestID, req.SourceRequestID)
	}

	if got.TMDBID != req.TMDBID {
		t.Errorf("TMDBID = %v, want %v", got.TMDBID, req.TMDBID)
	}
}

func TestUpdateRequestStatus(t *testing.T) {
	dbPath := "./test_update.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	req := &Request{
		SourceRequestID: "test-456",
		MediaType:       MediaTypeTV,
		TMDBID:          1234,
		Title:           "Test Show",
		Status:          StatusPending,
		RequestedAt:     time.Now(),
	}

	// 保存
	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}

	// 更新状态
	if err := store.UpdateRequestStatus("test-456", StatusSynced); err != nil {
		t.Fatalf("UpdateRequestStatus() error = %v", err)
	}

	// 验证
	got, err := store.GetRequest("test-456")
	if err != nil {
		t.Fatalf("GetRequest() error = %v", err)
	}

	if got.Status != StatusSynced {
		t.Errorf("Status = %v, want %v", got.Status, StatusSynced)
	}
}

func TestListPendingRequests(t *testing.T) {
	dbPath := "./test_list.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	// 保存多个请求
	for i := 1; i <= 5; i++ {
		req := &Request{
			SourceRequestID: string(rune('0' + i)),
			MediaType:       MediaTypeMovie,
			TMDBID:          i,
			Title:           "Test Movie",
			Status:          StatusPending,
			RequestedAt:     time.Now(),
		}
		if err := store.SaveRequest(req); err != nil {
			t.Fatalf("SaveRequest() error = %v", err)
		}
	}

	// 列出待处理请求
	requests, err := store.ListPendingRequests(10)
	if err != nil {
		t.Fatalf("ListPendingRequests() error = %v", err)
	}

	if len(requests) != 5 {
		t.Errorf("ListPendingRequests() returned %d requests, want 5", len(requests))
	}
}

func TestGetStats(t *testing.T) {
	dbPath := "./test_stats.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	// 保存不同状态的请求
	statuses := []SyncStatus{StatusPending, StatusSynced, StatusFailed}
	for i, status := range statuses {
		req := &Request{
			SourceRequestID: string(rune('a' + i)),
			MediaType:       MediaTypeMovie,
			TMDBID:          i,
			Title:           "Test",
			Status:          status,
			RequestedAt:     time.Now(),
		}
		if err := store.SaveRequest(req); err != nil {
			t.Fatalf("SaveRequest() error = %v", err)
		}
	}

	// 获取统计
	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", stats.TotalRequests)
	}

	if stats.PendingRequests != 1 {
		t.Errorf("PendingRequests = %d, want 1", stats.PendingRequests)
	}

	if stats.SyncedRequests != 1 {
		t.Errorf("SyncedRequests = %d, want 1", stats.SyncedRequests)
	}

	if stats.FailedRequests != 1 {
		t.Errorf("FailedRequests = %d, want 1", stats.FailedRequests)
	}
}
