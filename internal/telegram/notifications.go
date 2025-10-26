package telegram

import (
	"fmt"
	"time"
)

// NotifySubscribed 订阅成功通知
func (b *Bot) NotifySubscribed(title, mediaType string, tmdbID int) {
	msg := fmt.Sprintf(
		"✅ <b>已自动订阅</b>\n\n"+
			"📺 %s\n"+
			"🏷️ 类型: %s\n"+
			"🆔 TMDB ID: %d\n"+
			"⏰ %s",
		escapeHTML(title),
		getMediaTypeEmoji(mediaType),
		tmdbID,
		time.Now().Format("2006-01-02 15:04:05"),
	)
	b.SendMessageAsync(msg)
}

// NotifyDownloadStarted 开始下载通知
func (b *Bot) NotifyDownloadStarted(title string) {
	msg := fmt.Sprintf(
		"⬇️ <b>开始下载</b>\n\n"+
			"📺 %s\n"+
			"⏰ %s",
		escapeHTML(title),
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
		escapeHTML(title),
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
		escapeHTML(title),
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
		escapeHTML(title),
		escapeHTML(reason),
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
		escapeHTML(title),
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
		escapeHTML(errorMsg),
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

func escapeHTML(s string) string {
	// Telegram HTML 转义
	s = replaceAll(s, "&", "&amp;")
	s = replaceAll(s, "<", "&lt;")
	s = replaceAll(s, ">", "&gt;")
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}
