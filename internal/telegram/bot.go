package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Bot Telegram 机器人
type Bot struct {
	api     *tgbotapi.BotAPI
	chatIDs []string
	logger  *zap.Logger
	mu      sync.Mutex
	enabled bool
}

// NewBot 创建 Telegram Bot
func NewBot(token string, chatIDs []string, logger *zap.Logger) (*Bot, error) {
	if token == "" || len(chatIDs) == 0 {
		logger.Info("Telegram bot disabled: missing token or chat IDs")
		return &Bot{
			enabled: false,
			logger:  logger,
		}, nil
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create bot API: %w", err)
	}

	bot := &Bot{
		api:     api,
		chatIDs: chatIDs,
		logger:  logger,
		enabled: true,
	}

	logger.Info("Telegram bot initialized",
		zap.String("bot_username", api.Self.UserName),
		zap.Int("chat_count", len(chatIDs)),
	)

	return bot, nil
}

// SendMessage 发送消息到所有配置的 chat
func (b *Bot) SendMessage(text string) error {
	if !b.enabled {
		b.logger.Debug("Telegram bot disabled, message not sent")
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	var errors []string
	for _, chatID := range b.chatIDs {
		msg := tgbotapi.NewMessageToChannel(chatID, text)
		msg.ParseMode = "HTML"
		msg.DisableWebPagePreview = true

		if _, err := b.api.Send(msg); err != nil {
			b.logger.Error("Failed to send telegram message",
				zap.String("chat_id", chatID),
				zap.Error(err),
			)
			errors = append(errors, fmt.Sprintf("chat %s: %v", chatID, err))
		} else {
			b.logger.Debug("Telegram message sent",
				zap.String("chat_id", chatID),
			)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send to some chats: %s", strings.Join(errors, "; "))
	}

	return nil
}

// SendMessageAsync 异步发送消息（不阻塞）
func (b *Bot) SendMessageAsync(text string) {
	go func() {
		if err := b.SendMessage(text); err != nil {
			b.logger.Error("Async message send failed", zap.Error(err))
		}
	}()
}

// IsEnabled 检查 Bot 是否启用
func (b *Bot) IsEnabled() bool {
	return b.enabled
}

// Start 启动 Bot（接收命令 - 未来扩展）
func (b *Bot) Start(ctx context.Context) error {
	if !b.enabled {
		return nil
	}

	b.logger.Info("Telegram bot command listener started (currently not implemented)")

	// 未来可以在这里添加命令处理逻辑
	// 例如：/refresh, /stats, /help 等

	<-ctx.Done()
	b.logger.Info("Telegram bot stopped")
	return nil
}

// Close 关闭 Bot
func (b *Bot) Close() error {
	if b.api != nil {
		b.api.StopReceivingUpdates()
	}
	return nil
}
