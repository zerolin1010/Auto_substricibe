package tracker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// MPNotification MP 通知消息结构
type MPNotification struct {
	Channel           *string `json:"channel"`
	Source            *string `json:"source"`
	MType             string  `json:"mtype"`              // 通知类型，如 "订阅"
	CType             string  `json:"ctype"`              // 内容类型，如 "subscribeAdded", "subscribeComplete"
	Title             string  `json:"title"`              // 标题
	Text              string  `json:"text"`               // 详细文本
	Image             string  `json:"image"`              // 海报图片
	Link              *string `json:"link"`
	UserID            *string `json:"userid"`
	Username          string  `json:"username"`
	Date              *string `json:"date"`
	Action            int     `json:"action"`
	Targets           *string `json:"targets"`
	Buttons           *string `json:"buttons"`
	OriginalMessageID *string `json:"original_message_id"`
	OriginalChatID    *string `json:"original_chat_id"`
}

// SSEMessage SSE 消息
type SSEMessage struct {
	Message MPNotification `json:"message"`
}

// SSEClient SSE 客户端
type SSEClient struct {
	baseURL  string
	token    string
	logger   *zap.Logger
	ctx      context.Context
	onMessage func(*MPNotification)
}

// NewSSEClient 创建 SSE 客户端
func NewSSEClient(baseURL, token string, logger *zap.Logger, ctx context.Context) *SSEClient {
	return &SSEClient{
		baseURL: baseURL,
		token:   token,
		logger:  logger,
		ctx:     ctx,
	}
}

// SetMessageHandler 设置消息处理器
func (c *SSEClient) SetMessageHandler(handler func(*MPNotification)) {
	c.onMessage = handler
}

// Connect 连接到 SSE 端点
func (c *SSEClient) Connect() error {
	url := fmt.Sprintf("%s/api/v1/system/message", c.baseURL)

	c.logger.Info("Connecting to MP SSE endpoint", zap.String("url", url))

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("SSE client stopped")
			return nil
		default:
			if err := c.connectOnce(url); err != nil {
				c.logger.Error("SSE connection failed, retrying in 30s",
					zap.Error(err),
				)
				time.Sleep(30 * time.Second)
			}
		}
	}
}

// connectOnce 单次连接尝试
func (c *SSEClient) connectOnce(url string) error {
	req, err := http.NewRequestWithContext(c.ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// 设置认证头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{
		Timeout: 0, // SSE 不应该有超时
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("SSE connection established")

	// 读取 SSE 流
	reader := bufio.NewReader(resp.Body)
	var eventData strings.Builder

	for {
		select {
		case <-c.ctx.Done():
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					c.logger.Info("SSE connection closed by server")
					return fmt.Errorf("connection closed")
				}
				return fmt.Errorf("read line: %w", err)
			}

			line = strings.TrimRight(line, "\n\r")

			// SSE 格式：data: {json}
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				eventData.WriteString(data)
			} else if line == "" {
				// 空行表示事件结束
				if eventData.Len() > 0 {
					c.handleEvent(eventData.String())
					eventData.Reset()
				}
			} else if strings.HasPrefix(line, ":") {
				// 注释行，忽略（用于保持连接）
				continue
			}
		}
	}
}

// handleEvent 处理事件
func (c *SSEClient) handleEvent(data string) {
	c.logger.Debug("Received SSE event", zap.String("data", data))

	// 解析 JSON
	var msg SSEMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		c.logger.Error("Failed to unmarshal SSE message",
			zap.Error(err),
			zap.String("data", data),
		)
		return
	}

	// 调用消息处理器
	if c.onMessage != nil {
		c.onMessage(&msg.Message)
	}
}
