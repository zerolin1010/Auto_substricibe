package tracker

import (
	"context"
	"fmt"
	"strings"
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
		t.logger.Warn("SSE is enabled but MP API requires resource_token which is not available via login API. SSE will likely fail with 403 errors. Consider disabling SSE (TRACKER_SSE_ENABLED=false) and use polling only.")
		t.wg.Add(1)
		go t.runSSEListener()
	} else {
		t.logger.Info("SSE disabled, using polling only")
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

	// 获取 MP Token
	token, err := t.mpClient.GetToken(t.ctx)
	if err != nil {
		t.logger.Error("Failed to get MP token for SSE", zap.Error(err))
		return
	}

	// 创建 SSE 客户端
	sseClient := NewSSEClient(t.cfg.MPURL, token, t.logger, t.ctx)

	// 设置消息处理器
	sseClient.SetMessageHandler(func(notification *MPNotification) {
		t.handleNotification(notification)
	})

	// 连接到 SSE（阻塞式）
	if err := sseClient.Connect(); err != nil {
		t.logger.Error("SSE connection failed", zap.Error(err))
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
	} else if len(downloadHistory) > 0 {
		t.logger.Debug("Got download history", zap.Int("count", len(downloadHistory)))
		t.processDownloadHistory(allTracking, downloadHistory)
	}

	// 获取入库历史
	transferHistory, err := t.mpClient.GetTransferHistory(t.ctx, 1, 100)
	if err != nil {
		t.logger.Warn("Failed to get transfer history", zap.Error(err))
	} else if len(transferHistory) > 0 {
		t.logger.Debug("Got transfer history", zap.Int("count", len(transferHistory)))
		t.processTransferHistory(allTracking, transferHistory)
	}

	return nil
}

// processDownloadHistory 处理下载历史
func (t *Tracker) processDownloadHistory(tracking []*store.SubscriptionTracking, history []mp.DownloadHistoryItem) {
	for _, record := range tracking {
		// 在下载历史中查找匹配的记录
		for _, item := range history {
			// 匹配 TMDB ID
			if item.TMDBID != record.TMDBID {
				continue
			}

			// 匹配媒体类型（MP 使用中文，需要转换）
			matchType := false
			if record.MediaType == store.MediaTypeMovie && item.Type == "电影" {
				matchType = true
			} else if record.MediaType == store.MediaTypeTV && item.Type == "电视剧" {
				matchType = true
			}

			if matchType {
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

				// 下载完成的判断放在入库历史中处理
				// 因为 MP API 的下载历史不提供明确的完成状态

				break
			}
		}
	}
}

// processTransferHistory 处理入库历史
func (t *Tracker) processTransferHistory(tracking []*store.SubscriptionTracking, history []mp.TransferHistoryItem) {
	for _, record := range tracking {
		// 在入库历史中查找匹配的记录
		for _, item := range history {
			// 匹配 TMDB ID
			if item.TMDBID != record.TMDBID {
				continue
			}

			// 匹配媒体类型（MP 使用中文，需要转换）
			matchType := false
			if record.MediaType == store.MediaTypeMovie && item.Type == "电影" {
				matchType = true
			} else if record.MediaType == store.MediaTypeTV && item.Type == "电视剧" {
				matchType = true
			}

			if matchType {
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

// handleNotification 处理 MP 通知
func (t *Tracker) handleNotification(notification *MPNotification) {
	t.logger.Info("Received MP notification",
		zap.String("type", notification.MType),
		zap.String("content_type", notification.CType),
		zap.String("title", notification.Title),
		zap.String("username", notification.Username),
	)

	// 提取标题中的影片名称（去掉年份和后缀）
	// 例如：辛德勒的名单 (1993) 已添加订阅 → 辛德勒的名单
	title := extractMediaTitle(notification.Title)

	// 根据内容类型处理
	switch notification.CType {
	case "subscribeAdded":
		t.handleSubscribeAdded(title, notification)
	case "subscribeComplete":
		t.handleSubscribeComplete(title, notification)
	case "downloadStart":
		t.handleDownloadStart(title, notification)
	case "downloadComplete":
		t.handleDownloadComplete(title, notification)
	case "transferComplete":
		t.handleTransferComplete(title, notification)
	default:
		t.logger.Debug("Unhandled notification type",
			zap.String("content_type", notification.CType),
		)
	}
}

// handleSubscribeAdded 处理订阅已添加
func (t *Tracker) handleSubscribeAdded(title string, notification *MPNotification) {
	// 这个通知通常在我们自己订阅后触发，已经处理过了
	t.logger.Debug("Subscribe added notification received",
		zap.String("title", title),
	)
}

// handleSubscribeComplete 处理订阅完成（找到资源）
func (t *Tracker) handleSubscribeComplete(title string, notification *MPNotification) {
	t.logger.Info("Subscribe complete notification received",
		zap.String("title", title),
	)

	// 这可能意味着 MP 已经找到资源并准备开始下载
	// 我们可以发送一个通知
	if t.telegram != nil && t.telegram.IsEnabled() {
		msg := fmt.Sprintf(
			"🎯 <b>已找到资源</b>\n\n"+
				"📺 %s\n"+
				"👤 用户: %s\n"+
				"⏰ %s",
			title,
			notification.Username,
			time.Now().Format("2006-01-02 15:04:05"),
		)
		t.telegram.SendMessageAsync(msg)
	}
}

// handleDownloadStart 处理开始下载
func (t *Tracker) handleDownloadStart(title string, notification *MPNotification) {
	t.logger.Info("Download start notification received",
		zap.String("title", title),
	)

	// 发送 Telegram 通知
	if t.telegram != nil && t.telegram.IsEnabled() {
		t.telegram.NotifyDownloadStarted(title)
	}
}

// handleDownloadComplete 处理下载完成
func (t *Tracker) handleDownloadComplete(title string, notification *MPNotification) {
	t.logger.Info("Download complete notification received",
		zap.String("title", title),
	)

	// 发送 Telegram 通知
	if t.telegram != nil && t.telegram.IsEnabled() {
		t.telegram.NotifyDownloadComplete(title)
	}
}

// handleTransferComplete 处理入库完成
func (t *Tracker) handleTransferComplete(title string, notification *MPNotification) {
	t.logger.Info("Transfer complete notification received",
		zap.String("title", title),
	)

	// 发送 Telegram 通知
	if t.telegram != nil && t.telegram.IsEnabled() {
		t.telegram.NotifyTransferComplete(title)
	}
}

// extractMediaTitle 从通知标题中提取媒体标题
// 例如：辛德勒的名单 (1993) 已添加订阅 → 辛德勒的名单
func extractMediaTitle(fullTitle string) string {
	// 移除年份
	if idx := strings.Index(fullTitle, "("); idx > 0 {
		fullTitle = strings.TrimSpace(fullTitle[:idx])
	}
	// 移除后缀
	suffixes := []string{"已添加订阅", "已完成订阅", "开始下载", "下载完成", "入库完成"}
	for _, suffix := range suffixes {
		fullTitle = strings.TrimSuffix(fullTitle, suffix)
	}
	return strings.TrimSpace(fullTitle)
}
