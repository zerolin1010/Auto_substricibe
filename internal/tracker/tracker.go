package tracker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/jellyseerr-moviepilot-syncer/configs"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/mp"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/store"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/telegram"
	"go.uber.org/zap"
)

// Tracker 订阅跟踪器
type Tracker struct {
	cfg       *configs.Config
	mpClient  *mp.Client
	store     store.Store
	telegram  *telegram.Bot
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewTracker 创建跟踪器
func NewTracker(cfg *configs.Config, mpClient *mp.Client, st store.Store, tg *telegram.Bot, logger *zap.Logger) *Tracker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Tracker{
		cfg:      cfg,
		mpClient: mpClient,
		store:    st,
		telegram: tg,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动跟踪器
func (t *Tracker) Start() error {
	if !t.cfg.TrackerEnabled {
		t.logger.Info("Tracker disabled in config")
		return nil
	}

	t.logger.Info("Starting tracker",
		zap.Int("check_interval_minutes", t.cfg.TrackerCheckInterval),
		zap.Bool("sse_enabled", t.cfg.TrackerSSEEnabled),
	)

	// 启动 SSE 监听器（如果启用）
	if t.cfg.TrackerSSEEnabled {
		t.wg.Add(1)
		go t.runSSEListener()
	}

	// 启动轮询检查器
	t.wg.Add(1)
	go t.runPollingChecker()

	return nil
}

// Stop 停止跟踪器
func (t *Tracker) Stop() error {
	t.logger.Info("Stopping tracker")
	t.cancel()
	t.wg.Wait()
	t.logger.Info("Tracker stopped")
	return nil
}

// runSSEListener 运行 SSE 监听器
func (t *Tracker) runSSEListener() {
	defer t.wg.Done()

	t.logger.Info("SSE listener started")

	// TODO: 实现 SSE 连接和消息处理
	// MoviePilot SSE endpoint: /api/v1/system/message
	// 需要处理重连、错误恢复等

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			t.logger.Info("SSE listener stopped")
			return
		case <-ticker.C:
			// 保持连接活跃
			// TODO: 实际的 SSE 实现应该是阻塞式的
			t.logger.Debug("SSE listener heartbeat")
		}
	}
}

// runPollingChecker 运行轮询检查器
func (t *Tracker) runPollingChecker() {
	defer t.wg.Done()

	t.logger.Info("Polling checker started",
		zap.Int("interval_minutes", t.cfg.TrackerCheckInterval),
	)

	ticker := time.NewTicker(time.Duration(t.cfg.TrackerCheckInterval) * time.Minute)
	defer ticker.Stop()

	// 立即执行一次
	if err := t.checkDownloadStatus(); err != nil {
		t.logger.Error("Failed to check download status", zap.Error(err))
	}

	for {
		select {
		case <-t.ctx.Done():
			t.logger.Info("Polling checker stopped")
			return
		case <-ticker.C:
			if err := t.checkDownloadStatus(); err != nil {
				t.logger.Error("Failed to check download status", zap.Error(err))
			}
		}
	}
}

// checkDownloadStatus 检查下载状态
func (t *Tracker) checkDownloadStatus() error {
	t.logger.Debug("Checking download status")

	// 获取所有已订阅但未完成的跟踪记录
	subscribed, err := t.store.ListTrackingByStatus(store.TrackingSubscribed, 100)
	if err != nil {
		return fmt.Errorf("list subscribed tracking: %w", err)
	}

	downloading, err := t.store.ListTrackingByStatus(store.TrackingDownloading, 100)
	if err != nil {
		return fmt.Errorf("list downloading tracking: %w", err)
	}

	downloaded, err := t.store.ListTrackingByStatus(store.TrackingDownloaded, 100)
	if err != nil {
		return fmt.Errorf("list downloaded tracking: %w", err)
	}

	allTracking := append(subscribed, downloading...)
	allTracking = append(allTracking, downloaded...)

	if len(allTracking) == 0 {
		t.logger.Debug("No tracking records to check")
		return nil
	}

	t.logger.Info("Checking tracking records",
		zap.Int("count", len(allTracking)),
	)

	// 获取下载历史
	downloadHistory, err := t.mpClient.GetDownloadHistory(t.ctx, 1, 100)
	if err != nil {
		t.logger.Warn("Failed to get download history", zap.Error(err))
		// 不返回错误，继续检查入库历史
	} else {
		t.processDownloadHistory(allTracking, downloadHistory)
	}

	// 获取入库历史
	transferHistory, err := t.mpClient.GetTransferHistory(t.ctx, 1, 100)
	if err != nil {
		t.logger.Warn("Failed to get transfer history", zap.Error(err))
	} else {
		t.processTransferHistory(allTracking, transferHistory)
	}

	return nil
}

// processDownloadHistory 处理下载历史
func (t *Tracker) processDownloadHistory(tracking []*store.SubscriptionTracking, history *mp.DownloadHistoryResponse) {
	for _, record := range tracking {
		// 在下载历史中查找匹配的记录
		for _, item := range history.Items {
			if item.TMDBID == record.TMDBID && item.Type == string(record.MediaType) {
				// 找到匹配的下载记录
				// 只有从 subscribed 状态才发送"开始下载"通知（避免重复）
				if record.SubscribeStatus == store.TrackingSubscribed {
					// 从已订阅变为下载中
					t.logger.Info("Download started",
						zap.String("title", record.Title),
						zap.Int("tmdb_id", record.TMDBID),
					)

					now := time.Now()
					record.SubscribeStatus = store.TrackingDownloading
					record.DownloadStartTime = &now
					if err := t.store.UpdateTracking(record); err != nil {
						t.logger.Error("Failed to update tracking", zap.Error(err))
						continue
					}

					// 发送 Telegram 通知（只发送一次）
					if t.telegram != nil && t.telegram.IsEnabled() {
						t.telegram.NotifyDownloadStarted(record.Title)
					}

					// 保存事件
					event := &store.DownloadEvent{
						SourceRequestID: record.SourceRequestID,
						EventType:       store.EventDownloadStarted,
						EventData:       fmt.Sprintf("{\"tmdb_id\": %d, \"title\": \"%s\"}", record.TMDBID, record.Title),
					}
					if err := t.store.SaveEvent(event); err != nil {
						t.logger.Error("Failed to save event", zap.Error(err))
					}
				}

				// 检查下载是否完成
				// 只有从 downloading 状态才发送"下载完成"通知（避免重复）
				if item.Status == "completed" && record.SubscribeStatus == store.TrackingDownloading {
					t.logger.Info("Download completed",
						zap.String("title", record.Title),
						zap.Int("tmdb_id", record.TMDBID),
					)

					now := time.Now()
					record.SubscribeStatus = store.TrackingDownloaded
					record.DownloadFinishTime = &now
					if err := t.store.UpdateTracking(record); err != nil {
						t.logger.Error("Failed to update tracking", zap.Error(err))
						continue
					}

					// 发送 Telegram 通知（只发送一次）
					if t.telegram != nil && t.telegram.IsEnabled() {
						t.telegram.NotifyDownloadComplete(record.Title)
					}

					// 保存事件
					event := &store.DownloadEvent{
						SourceRequestID: record.SourceRequestID,
						EventType:       store.EventDownloadComplete,
						EventData:       fmt.Sprintf("{\"tmdb_id\": %d, \"title\": \"%s\"}", record.TMDBID, record.Title),
					}
					if err := t.store.SaveEvent(event); err != nil {
						t.logger.Error("Failed to save event", zap.Error(err))
					}
				}

				break
			}
		}
	}
}

// processTransferHistory 处理入库历史
func (t *Tracker) processTransferHistory(tracking []*store.SubscriptionTracking, history *mp.TransferHistoryResponse) {
	for _, record := range tracking {
		// 在入库历史中查找匹配的记录
		for _, item := range history.Items {
			if item.TMDBID == record.TMDBID && item.Type == string(record.MediaType) {
				// 找到匹配的入库记录
				// 只有从 downloaded 或 downloading 状态才发送"入库完成"通知（避免重复）
				// 同时排除已经是 transferred 状态的记录
				if (record.SubscribeStatus == store.TrackingDownloaded || record.SubscribeStatus == store.TrackingDownloading) &&
					record.SubscribeStatus != store.TrackingTransferred {
					t.logger.Info("Transfer completed",
						zap.String("title", record.Title),
						zap.Int("tmdb_id", record.TMDBID),
					)

					now := time.Now()
					record.SubscribeStatus = store.TrackingTransferred
					record.TransferTime = &now
					if err := t.store.UpdateTracking(record); err != nil {
						t.logger.Error("Failed to update tracking", zap.Error(err))
						continue
					}

					// 发送 Telegram 通知（只发送一次）
					if t.telegram != nil && t.telegram.IsEnabled() {
						t.telegram.NotifyTransferComplete(record.Title)
					}

					// 保存事件
					event := &store.DownloadEvent{
						SourceRequestID: record.SourceRequestID,
						EventType:       store.EventTransferComplete,
						EventData:       fmt.Sprintf("{\"tmdb_id\": %d, \"title\": \"%s\"}", record.TMDBID, record.Title),
					}
					if err := t.store.SaveEvent(event); err != nil {
						t.logger.Error("Failed to save event", zap.Error(err))
					}

					break
				}
			}
		}
	}
}
