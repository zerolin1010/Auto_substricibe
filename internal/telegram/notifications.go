package telegram

import (
	"fmt"
	"html"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// NotifySubscribed 订阅成功通知（带图片）
func (b *Bot) NotifySubscribed(title, mediaType string, tmdbID int, posterPath string) {
	if !b.enabled {
		return
	}

	caption := fmt.Sprintf(
		"✅ <b>已自动订阅</b>\n\n"+
			"📺 %s\n"+
			"🏷️ 类型: %s\n"+
			"🆔 TMDB ID: %d\n"+
			"⏰ %s",
		html.EscapeString(title),
		getMediaTypeEmoji(mediaType),
		tmdbID,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	// 如果有海报，发送图片消息
	if posterPath != "" {
		imageURL := fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", posterPath)
		b.logger.Info("Sending notification with poster",
			zap.String("title", title),
			zap.String("poster_path", posterPath),
			zap.String("image_url", imageURL),
		)
		b.sendPhotoAsync(imageURL, caption)
	} else {
		// 没有海报，发送纯文本
		b.logger.Info("Sending notification without poster (posterPath is empty)",
			zap.String("title", title),
			zap.Int("tmdb_id", tmdbID),
		)
		b.SendMessageAsync(caption)
	}
}

// NotifyAlreadyExists 媒体已存在通知（带图片）
func (b *Bot) NotifyAlreadyExists(title, mediaType string, tmdbID int, posterPath string) {
	if !b.enabled {
		return
	}

	caption := fmt.Sprintf(
		"ℹ️ <b>媒体已在库中</b>\n\n"+
			"📺 %s\n"+
			"🏷️ 类型: %s\n"+
			"🆔 TMDB ID: %d\n"+
			"💡 该影片已存在于媒体库，无需重复下载\n"+
			"⏰ %s",
		html.EscapeString(title),
		getMediaTypeEmoji(mediaType),
		tmdbID,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	// 如果有海报，发送图片消息
	if posterPath != "" {
		imageURL := fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", posterPath)
		b.logger.Info("Sending 'already exists' notification with poster",
			zap.String("title", title),
			zap.String("poster_path", posterPath),
		)
		b.sendPhotoAsync(imageURL, caption)
	} else {
		// 没有海报，发送纯文本
		b.logger.Info("Sending 'already exists' notification without poster",
			zap.String("title", title),
			zap.Int("tmdb_id", tmdbID),
		)
		b.SendMessageAsync(caption)
	}
}

// sendPhotoAsync 异步发送图片消息
func (b *Bot) sendPhotoAsync(photoURL, caption string) {
	go func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		for _, chatID := range b.chatIDs {
			// 创建 PhotoConfig
			photo := tgbotapi.PhotoConfig{
				BaseFile: tgbotapi.BaseFile{
					BaseChat: tgbotapi.BaseChat{},
					File:     tgbotapi.FileURL(photoURL),
				},
				Caption:   caption,
				ParseMode: "HTML",
			}

			// 尝试解析为数字 ID，否则当作 @username
			if chatIDNum, err := strconv.ParseInt(chatID, 10, 64); err == nil {
				photo.ChatID = chatIDNum
			} else {
				photo.ChannelUsername = chatID
			}

			if _, err := b.api.Send(photo); err != nil {
				b.logger.Error("Failed to send telegram photo",
					zap.String("chat_id", chatID),
					zap.Error(err),
				)
			} else {
				b.logger.Debug("Telegram photo sent",
					zap.String("chat_id", chatID),
				)
			}
		}
	}()
}

// NotifyDownloadStarted 开始下载通知
func (b *Bot) NotifyDownloadStarted(title string) {
	msg := fmt.Sprintf(
		"⬇️ <b>开始下载</b>\n\n"+
			"📺 %s\n"+
			"⏰ %s",
		html.EscapeString(title),
		time.Now().Format("2006-01-02 15:04:05"),
	)
	b.SendMessageAsync(msg)
}

// NotifyDownloadComplete 下载完成通知
func (b *Bot) NotifyDownloadComplete(title string) {
	msg := fmt.Sprintf(
		"✅ <b>下载完成</b>\n\n"+
			"📺 %s\n"+
			"⏰ %s",
		html.EscapeString(title),
		time.Now().Format("2006-01-02 15:04:05"),
	)
	b.SendMessageAsync(msg)
}

// NotifyTransferComplete 入库成功通知
func (b *Bot) NotifyTransferComplete(title string) {
	msg := fmt.Sprintf(
		"📦 <b>入库成功</b>\n\n"+
			"📺 %s\n"+
			"⏰ %s",
		html.EscapeString(title),
		time.Now().Format("2006-01-02 15:04:05"),
	)
	b.SendMessageAsync(msg)
}

// NotifyFailed 失败通知
func (b *Bot) NotifyFailed(title, reason string) {
	msg := fmt.Sprintf(
		"❌ <b>订阅失败</b>\n\n"+
			"📺 %s\n"+
			"💬 原因: %s\n"+
			"⏰ %s",
		html.EscapeString(title),
		html.EscapeString(reason),
		time.Now().Format("2006-01-02 15:04:05"),
	)
	b.SendMessageAsync(msg)
}

// NotifyRetrying 重试通知
func (b *Bot) NotifyRetrying(title string, attempt, maxAttempts int) {
	msg := fmt.Sprintf(
		"🔄 <b>智能重试</b>\n\n"+
			"📺 %s\n"+
			"🔢 尝试: %d/%d\n"+
			"⏰ %s",
		html.EscapeString(title),
		attempt,
		maxAttempts,
		time.Now().Format("2006-01-02 15:04:05"),
	)
	b.SendMessageAsync(msg)
}

// NotifyDailyReport 每日报告通知
func (b *Bot) NotifyDailyReport(report string) {
	msg := fmt.Sprintf(
		"📊 <b>每日订阅报告</b>\n\n%s",
		report,
	)
	b.SendMessageAsync(msg)
}

// NotifyError 错误通知
func (b *Bot) NotifyError(errorMsg string) {
	msg := fmt.Sprintf(
		"⚠️ <b>系统错误</b>\n\n"+
			"💬 %s\n"+
			"⏰ %s",
		html.EscapeString(errorMsg),
		time.Now().Format("2006-01-02 15:04:05"),
	)
	b.SendMessageAsync(msg)
}

// 辅助函数

func getMediaTypeEmoji(mediaType string) string {
	switch mediaType {
	case "movie":
		return "🎬 电影"
	case "tv":
		return "📺 剧集"
	default:
		return mediaType
	}
}

