package core

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/yourusername/jellyseerr-moviepilot-syncer/configs"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/jelly"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/mp"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/store"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/telegram"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/tmdb"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/tracker"
	"go.uber.org/zap"
)

// Syncer 同步器
type Syncer struct {
	cfg         *configs.Config
	jellyClient *jelly.Client
	mpClient    *mp.Client
	tmdbClient  *tmdb.Client
	store       store.Store
	telegram    *telegram.Bot
	tracker     *tracker.Tracker
	logger      *zap.Logger
}

// NewSyncer 创建同步器
func NewSyncer(cfg *configs.Config, logger *zap.Logger, ctx context.Context) (*Syncer, error) {
	// 创建 Jellyseerr 客户端
	jellyClient := jelly.NewClient(cfg.JellyURL, cfg.JellyAPIKey)

	// 创建 TMDB 客户端（可选）
	tmdbClient := tmdb.NewClient(cfg.TMDPAPIKey)
	if tmdbClient != nil {
		logger.Info("TMDB client enabled for poster fetching")
	} else {
		logger.Info("TMDB client disabled (no API key configured)")
	}

	// 创建 MoviePilot 客户端
	mpClient, err := mp.NewClient(mp.ClientConfig{
		BaseURL:      cfg.MPURL,
		Username:     cfg.MPUsername,
		Password:     cfg.MPPassword,
		AuthScheme:   cfg.MPAuthScheme,
		RateLimitPS:  cfg.MPRateLimitPS,
		MaxRetries:   cfg.MaxRetries,
		DryRun:       cfg.MPDryRun,
		TokenRefresh: cfg.MPTokenRefresh,
	}, ctx)
	if err != nil {
		return nil, fmt.Errorf("create mp client: %w", err)
	}

	// 创建存储
	var st store.Store
	if cfg.StoreType == "sqlite" {
		st, err = store.NewSQLiteStore(cfg.StorePath)
		if err != nil {
			return nil, fmt.Errorf("create store: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported store type: %s", cfg.StoreType)
	}

	// 创建 Telegram Bot（如果启用）
	var tgBot *telegram.Bot
	if cfg.TelegramEnabled {
		logger.Info("Telegram bot enabled, initializing...",
			zap.Int("chat_count", len(cfg.TelegramChatIDs)),
		)
		tgBot, err = telegram.NewBot(cfg.TelegramToken, cfg.TelegramChatIDs, logger)
		if err != nil {
			logger.Error("Failed to create telegram bot", zap.Error(err))
			// 即使失败也继续，只是没有通知功能
		}
	} else {
		logger.Info("Telegram bot disabled in config")
	}

	// 创建 Tracker（如果启用）
	var trk *tracker.Tracker
	if cfg.TrackerEnabled {
		logger.Info("Tracker enabled, initializing...")
		trk = tracker.NewTracker(cfg, mpClient, st, tgBot, logger)
	} else {
		logger.Info("Tracker disabled in config")
	}

	return &Syncer{
		cfg:         cfg,
		jellyClient: jellyClient,
		mpClient:    mpClient,
		tmdbClient:  tmdbClient,
		store:       st,
		telegram:    tgBot,
		tracker:     trk,
		logger:      logger,
	}, nil
}

// SyncOnce 执行一次同步
func (s *Syncer) SyncOnce(ctx context.Context) error {
	s.logger.Info("starting sync")

	// 1. 从 Jellyseerr 获取已批准的请求
	requests, err := s.jellyClient.FetchAllApprovedRequests(ctx, s.cfg.JellyPageSize)
	if err != nil {
		return fmt.Errorf("fetch approved requests: %w", err)
	}

	s.logger.Info("fetched requests from Jellyseerr", zap.Int("count", len(requests)))

	// 2. 转换并保存到本地存储
	for _, req := range requests {
		if err := s.processRequest(ctx, req); err != nil {
			s.logger.Error("process request failed",
				zap.Int("request_id", req.ID),
				zap.Error(err),
			)
			continue
		}
	}

	// 3. 处理待同步的请求
	if err := s.processPendingRequests(ctx); err != nil {
		return fmt.Errorf("process pending requests: %w", err)
	}

	// 4. 打印统计信息
	stats, err := s.store.GetStats()
	if err != nil {
		s.logger.Warn("get stats failed", zap.Error(err))
	} else {
		s.logger.Info("sync completed",
			zap.Int("total", stats.TotalRequests),
			zap.Int("pending", stats.PendingRequests),
			zap.Int("synced", stats.SyncedRequests),
			zap.Int("failed", stats.FailedRequests),
		)
	}

	return nil
}

// processRequest 处理单个请求
func (s *Syncer) processRequest(ctx context.Context, jellyReq *jelly.MediaRequestV2) error {
	sourceRequestID := strconv.Itoa(jellyReq.ID)

	// 检查是否已存在
	existing, err := s.store.GetRequest(sourceRequestID)
	if err != nil {
		return fmt.Errorf("get request: %w", err)
	}

	if existing != nil && existing.Status == store.StatusSynced {
		// 已同步，跳过
		return nil
	}

	// 获取媒体详情（获取标题和海报）
	var title string
	var posterPath string
	mediaType := "movie"
	if jellyReq.IsMovie() {
		mediaType = "movie"
	} else if jellyReq.IsTV() {
		mediaType = "tv"
	}

	details, err := s.jellyClient.GetMediaDetails(ctx, mediaType, jellyReq.Media.TMDBID)
	if err != nil {
		s.logger.Warn("get media details failed, using fallback",
			zap.Int("tmdb_id", jellyReq.Media.TMDBID),
			zap.Error(err),
		)
		title = fmt.Sprintf("TMDB-%d", jellyReq.Media.TMDBID)
	} else {
		title = details.GetTitle()
		posterPath = details.PosterPath
		s.logger.Debug("fetched media details from Jellyseerr",
			zap.String("title", title),
			zap.String("poster_path", posterPath),
			zap.Int("tmdb_id", jellyReq.Media.TMDBID),
		)
	}

	// 如果 Jellyseerr 没有返回 posterPath，尝试从 TMDB 获取
	if posterPath == "" && s.tmdbClient != nil {
		s.logger.Debug("posterPath empty from Jellyseerr, trying TMDB API",
			zap.Int("tmdb_id", jellyReq.Media.TMDBID),
		)
		tmdbPoster, err := s.tmdbClient.GetPosterPath(ctx, mediaType, jellyReq.Media.TMDBID)
		if err != nil {
			s.logger.Warn("failed to fetch poster from TMDB",
				zap.Int("tmdb_id", jellyReq.Media.TMDBID),
				zap.Error(err),
			)
		} else if tmdbPoster != "" {
			posterPath = tmdbPoster
			s.logger.Info("fetched poster from TMDB API",
				zap.String("title", title),
				zap.String("poster_path", posterPath),
			)
		}
	}

	// 转换为本地请求
	localReq := &store.Request{
		SourceRequestID: sourceRequestID,
		MediaType:       store.MediaType(mediaType),
		TMDBID:          jellyReq.Media.TMDBID,
		Title:           title,
		PosterPath:      posterPath,
		Status:          store.StatusPending,
		RequestedAt:     jellyReq.CreatedAt,
	}

	// 处理剧集季和集
	if jellyReq.IsTV() && len(jellyReq.Seasons) > 0 {
		seasons := []int{}
		episodes := make(map[int][]int)

		for _, season := range jellyReq.Seasons {
			// 根据配置，可能排除特别季 S00
			if season.SeasonNumber == 0 {
				s.logger.Debug("skipping special season S00",
					zap.String("title", title),
				)
				continue
			}

			seasons = append(seasons, season.SeasonNumber)

			// 处理集
			if len(season.Episodes) > 0 {
				eps := []int{}
				for _, ep := range season.Episodes {
					eps = append(eps, ep.EpisodeNumber)
				}
				episodes[season.SeasonNumber] = eps
			}
		}

		if err := localReq.SetSeasons(seasons); err != nil {
			return fmt.Errorf("set seasons: %w", err)
		}
		if len(episodes) > 0 {
			if err := localReq.SetEpisodes(episodes); err != nil {
				return fmt.Errorf("set episodes: %w", err)
			}
		}
	}

	// 保存到存储
	if err := s.store.SaveRequest(localReq); err != nil {
		return fmt.Errorf("save request: %w", err)
	}

	s.logger.Debug("saved request",
		zap.String("source_request_id", sourceRequestID),
		zap.String("title", title),
		zap.String("type", string(localReq.MediaType)),
	)

	return nil
}

// processPendingRequests 处理待同步的请求
func (s *Syncer) processPendingRequests(ctx context.Context) error {
	// 获取待处理请求
	requests, err := s.store.ListPendingRequests(100)
	if err != nil {
		return fmt.Errorf("list pending requests: %w", err)
	}

	s.logger.Info("processing pending requests", zap.Int("count", len(requests)))

	for _, req := range requests {
		if err := s.syncToMoviePilot(ctx, req); err != nil {
			s.logger.Error("sync to MoviePilot failed",
				zap.String("source_request_id", req.SourceRequestID),
				zap.String("title", req.Title),
				zap.Error(err),
			)

			// 保存错误信息
			link := &store.MPLink{
				SourceRequestID: req.SourceRequestID,
				State:           store.StatusFailed,
				LastError:       sanitizeError(err),
			}
			if err := s.store.SaveMPLink(link); err != nil {
				s.logger.Error("save mp link failed", zap.Error(err))
			}

			// 更新请求状态
			if err := s.store.UpdateRequestStatus(req.SourceRequestID, store.StatusFailed); err != nil {
				s.logger.Error("update request status failed", zap.Error(err))
			}

			continue
		}

		// 更新请求状态为已同步
		if err := s.store.UpdateRequestStatus(req.SourceRequestID, store.StatusSynced); err != nil {
			s.logger.Error("update request status failed", zap.Error(err))
		}
	}

	return nil
}

// syncToMoviePilot 同步到 MoviePilot
func (s *Syncer) syncToMoviePilot(ctx context.Context, req *store.Request) error {
	// 检查是否已有 MP 链接
	existing, err := s.store.GetMPLink(req.SourceRequestID)
	if err != nil {
		return fmt.Errorf("get mp link: %w", err)
	}

	if existing != nil && existing.State == store.StatusSynced {
		// 已同步
		return nil
	}

	// 电影：直接创建订阅
	if req.MediaType == store.MediaTypeMovie {
		return s.subscribeMovie(ctx, req)
	}

	// 剧集：按季或按集创建订阅
	if req.MediaType == store.MediaTypeTV {
		return s.subscribeTV(ctx, req)
	}

	return fmt.Errorf("unknown media type: %s", req.MediaType)
}

// subscribeMovie 订阅电影
func (s *Syncer) subscribeMovie(ctx context.Context, req *store.Request) error {
	mpReq := &mp.SubscribeRequest{
		Name:   req.Title,
		Type:   "电影", // MoviePilot 需要中文类型
		TMDBID: req.TMDBID,
	}

	resp, err := s.mpClient.Subscribe(ctx, mpReq)
	if err != nil {
		return fmt.Errorf("subscribe movie: %w", err)
	}

	// 检查是否为"已存在"响应
	alreadyExists := resp.IsAlreadyExists()
	if alreadyExists {
		s.logger.Info("movie already exists in library",
			zap.String("title", req.Title),
			zap.Int("tmdb_id", req.TMDBID),
			zap.String("message", resp.Message),
		)
	}

	// 保存链接
	subscribeID := ""
	if resp.Data != nil {
		if resp.Data.ID > 0 {
			subscribeID = strconv.Itoa(resp.Data.ID)
		} else if resp.Data.SubscribeID > 0 {
			subscribeID = strconv.Itoa(resp.Data.SubscribeID)
		}
	}

	link := &store.MPLink{
		SourceRequestID: req.SourceRequestID,
		MPSubscribeID:   subscribeID,
		State:           store.StatusSynced,
	}

	if err := s.store.SaveMPLink(link); err != nil {
		return fmt.Errorf("save mp link: %w", err)
	}

	if !alreadyExists {
		s.logger.Info("subscribed movie to MoviePilot",
			zap.String("title", req.Title),
			zap.Int("tmdb_id", req.TMDBID),
			zap.String("subscribe_id", subscribeID),
		)
	}

	// 保存到跟踪表
	now := time.Now()
	var trackingStatus store.TrackingStatus
	if alreadyExists {
		// 已存在的影片直接标记为已入库
		trackingStatus = store.TrackingTransferred
	} else {
		trackingStatus = store.TrackingSubscribed
	}

	tracking := &store.SubscriptionTracking{
		SourceRequestID: req.SourceRequestID,
		TMDBID:          req.TMDBID,
		Title:           req.Title,
		MediaType:       req.MediaType,
		SubscribeStatus: trackingStatus,
		SubscribeTime:   &now,
	}

	// 如果已存在，也设置 TransferTime
	if alreadyExists {
		tracking.TransferTime = &now
	}

	if err := s.store.SaveTracking(tracking); err != nil {
		s.logger.Warn("Failed to save tracking", zap.Error(err))
	}

	// 发送 Telegram 通知
	if s.telegram != nil && s.telegram.IsEnabled() {
		if alreadyExists {
			// 发送"已在媒体库"通知
			s.logger.Debug("Sending 'already exists' notification", zap.String("title", req.Title))
			s.telegram.NotifyAlreadyExists(req.Title, string(req.MediaType), req.TMDBID, req.PosterPath)
		} else {
			// 发送普通订阅通知
			s.logger.Debug("Sending telegram notification for movie", zap.String("title", req.Title))
			s.telegram.NotifySubscribed(req.Title, string(req.MediaType), req.TMDBID, req.PosterPath)
		}
	} else {
		if s.telegram == nil {
			s.logger.Debug("Telegram bot not initialized, skipping notification")
		} else {
			s.logger.Debug("Telegram bot disabled, skipping notification")
		}
	}

	return nil
}

// subscribeTV 订阅剧集
func (s *Syncer) subscribeTV(ctx context.Context, req *store.Request) error {
	seasons, err := req.GetSeasons()
	if err != nil {
		return fmt.Errorf("get seasons: %w", err)
	}

	episodes, err := req.GetEpisodes()
	if err != nil {
		return fmt.Errorf("get episodes: %w", err)
	}

	// 跟踪是否有季已存在
	var alreadyExists bool

	// 根据配置决定按季还是按集
	if s.cfg.MPTVEpisodeMode == "season" {
		// 按季订阅
		for _, season := range seasons {
			mpReq := &mp.SubscribeRequest{
				Name:   req.Title,
				Type:   "电视剧", // MoviePilot 需要中文类型
				TMDBID: req.TMDBID,
				Season: season,
			}

			resp, err := s.mpClient.Subscribe(ctx, mpReq)
			if err != nil {
				return fmt.Errorf("subscribe season %d: %w", season, err)
			}

			// 检查是否已存在（检查第一季）
			if season == seasons[0] && resp.IsAlreadyExists() {
				alreadyExists = true
				s.logger.Info("TV show already exists in library",
					zap.String("title", req.Title),
					zap.Int("tmdb_id", req.TMDBID),
					zap.String("message", resp.Message),
				)
			}

			if !alreadyExists {
				s.logger.Info("subscribed TV season to MoviePilot",
					zap.String("title", req.Title),
					zap.Int("tmdb_id", req.TMDBID),
					zap.Int("season", season),
				)
			}

			// 保存链接（仅保存第一个）
			if season == seasons[0] {
				subscribeID := ""
				if resp.Data != nil {
					if resp.Data.ID > 0 {
						subscribeID = strconv.Itoa(resp.Data.ID)
					} else if resp.Data.SubscribeID > 0 {
						subscribeID = strconv.Itoa(resp.Data.SubscribeID)
					}
				}

				link := &store.MPLink{
					SourceRequestID: req.SourceRequestID,
					MPSubscribeID:   subscribeID,
					State:           store.StatusSynced,
				}

				if err := s.store.SaveMPLink(link); err != nil {
					return fmt.Errorf("save mp link: %w", err)
				}
			}
		}
	} else if s.cfg.MPTVEpisodeMode == "episode" {
		// 按集订阅
		for season, eps := range episodes {
			if len(eps) > 0 {
				mpReq := &mp.SubscribeRequest{
					Name:     req.Title,
					Type:     "电视剧", // MoviePilot 需要中文类型
					TMDBID:   req.TMDBID,
					Season:   season,
					Episodes: eps,
				}

				resp, err := s.mpClient.Subscribe(ctx, mpReq)
				if err != nil {
					return fmt.Errorf("subscribe season %d episodes: %w", season, err)
				}

				// 检查是否已存在（检查第一季）
				if season == seasons[0] && resp.IsAlreadyExists() {
					alreadyExists = true
					s.logger.Info("TV show already exists in library",
						zap.String("title", req.Title),
						zap.Int("tmdb_id", req.TMDBID),
						zap.String("message", resp.Message),
					)
				}

				if !alreadyExists {
					s.logger.Info("subscribed TV episodes to MoviePilot",
						zap.String("title", req.Title),
						zap.Int("tmdb_id", req.TMDBID),
						zap.Int("season", season),
						zap.Ints("episodes", eps),
					)
				}

				// 保存链接（仅保存第一个）
				if season == seasons[0] {
					subscribeID := ""
					if resp.Data != nil {
						if resp.Data.ID > 0 {
							subscribeID = strconv.Itoa(resp.Data.ID)
						} else if resp.Data.SubscribeID > 0 {
							subscribeID = strconv.Itoa(resp.Data.SubscribeID)
						}
					}

					link := &store.MPLink{
						SourceRequestID: req.SourceRequestID,
						MPSubscribeID:   subscribeID,
						State:           store.StatusSynced,
					}

					if err := s.store.SaveMPLink(link); err != nil {
						return fmt.Errorf("save mp link: %w", err)
					}
				}
			}
		}
	}

	// 保存到跟踪表
	now := time.Now()
	var trackingStatus store.TrackingStatus
	if alreadyExists {
		// 已存在的剧集直接标记为已入库
		trackingStatus = store.TrackingTransferred
	} else {
		trackingStatus = store.TrackingSubscribed
	}

	tracking := &store.SubscriptionTracking{
		SourceRequestID: req.SourceRequestID,
		TMDBID:          req.TMDBID,
		Title:           req.Title,
		MediaType:       req.MediaType,
		SubscribeStatus: trackingStatus,
		SubscribeTime:   &now,
	}

	// 如果已存在，也设置 TransferTime
	if alreadyExists {
		tracking.TransferTime = &now
	}

	if err := s.store.SaveTracking(tracking); err != nil {
		s.logger.Warn("Failed to save tracking", zap.Error(err))
	}

	// 发送 Telegram 通知
	if s.telegram != nil && s.telegram.IsEnabled() {
		if alreadyExists {
			// 发送"已在媒体库"通知
			s.logger.Debug("Sending 'already exists' notification for TV", zap.String("title", req.Title))
			s.telegram.NotifyAlreadyExists(req.Title, string(req.MediaType), req.TMDBID, req.PosterPath)
		} else {
			// 发送普通订阅通知
			s.logger.Debug("Sending telegram notification for TV", zap.String("title", req.Title))
			s.telegram.NotifySubscribed(req.Title, string(req.MediaType), req.TMDBID, req.PosterPath)
		}
	} else {
		if s.telegram == nil {
			s.logger.Debug("Telegram bot not initialized, skipping notification")
		} else {
			s.logger.Debug("Telegram bot disabled, skipping notification")
		}
	}

	return nil
}

// Close 关闭同步器
func (s *Syncer) Close() error {
	// 停止 tracker
	if s.tracker != nil {
		if err := s.tracker.Stop(); err != nil {
			s.logger.Error("Failed to stop tracker", zap.Error(err))
		}
	}
	return s.store.Close()
}

// RunDaemon 以守护进程模式运行
func (s *Syncer) RunDaemon(ctx context.Context) error {
	s.logger.Info("starting daemon mode",
		zap.Int("interval_minutes", s.cfg.SyncInterval),
	)

	// 启动 tracker
	if s.tracker != nil {
		if err := s.tracker.Start(); err != nil {
			s.logger.Error("Failed to start tracker", zap.Error(err))
		}
	}

	ticker := time.NewTicker(time.Duration(s.cfg.SyncInterval) * time.Minute)
	defer ticker.Stop()

	// 立即执行一次
	if err := s.SyncOnce(ctx); err != nil {
		s.logger.Error("sync failed", zap.Error(err))
	}

	// 定时执行
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("daemon stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := s.SyncOnce(ctx); err != nil {
				s.logger.Error("sync failed", zap.Error(err))
			}
		}
	}
}

// sanitizeError 清理错误信息（移除敏感信息）
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	// 这里可以添加更多的清理逻辑
	msg := err.Error()
	if len(msg) > 500 {
		msg = msg[:500] + "..."
	}
	return msg
}
