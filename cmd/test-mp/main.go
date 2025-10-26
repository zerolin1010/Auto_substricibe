package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// 从环境变量读取配置
	mpURL := getEnv("MP_URL", "http://localhost:5000")
	username := getEnv("MP_USERNAME", "admin")
	password := getEnv("MP_PASSWORD", "password")

	fmt.Printf("=== MoviePilot API 测试工具 ===\n\n")
	fmt.Printf("MP URL: %s\n", mpURL)
	fmt.Printf("Username: %s\n\n", username)

	ctx := context.Background()

	// 1. 测试登录获取 Token
	fmt.Println("【步骤 1】测试登录...")
	token, err := login(ctx, mpURL, username, password)
	if err != nil {
		log.Fatalf("登录失败: %v", err)
	}
	fmt.Printf("✅ 登录成功！\n")
	fmt.Printf("Token: %s...\n\n", token[:50])

	// 2. 测试获取下载历史
	fmt.Println("【步骤 2】测试获取下载历史...")
	if err := testDownloadHistory(ctx, mpURL, token); err != nil {
		fmt.Printf("⚠️  获取下载历史失败: %v\n\n", err)
	}

	// 3. 测试获取入库历史
	fmt.Println("【步骤 3】测试获取入库历史...")
	if err := testTransferHistory(ctx, mpURL, token); err != nil {
		fmt.Printf("⚠️  获取入库历史失败: %v\n\n", err)
	}

	// 4. 测试 SSE 连接（带不同认证方式）
	fmt.Println("【步骤 4】测试 SSE 连接...")

	// 方式 1: Bearer Token
	fmt.Println("\n  尝试方式 1: Authorization: Bearer <token>")
	if err := testSSEWithBearer(ctx, mpURL, token); err != nil {
		fmt.Printf("  ❌ 失败: %v\n", err)
	}

	// 方式 2: Cookie
	fmt.Println("\n  尝试方式 2: Cookie: resource_token=<token>")
	if err := testSSEWithCookie(ctx, mpURL, token); err != nil {
		fmt.Printf("  ❌ 失败: %v\n", err)
	}

	// 方式 3: 查询参数
	fmt.Println("\n  尝试方式 3: Query Parameter: ?token=<token>")
	if err := testSSEWithQuery(ctx, mpURL, token); err != nil {
		fmt.Printf("  ❌ 失败: %v\n", err)
	}

	fmt.Println("\n=== 测试完成 ===")
}

// login 登录获取 Token
func login(ctx context.Context, baseURL, username, password string) (string, error) {
	url := baseURL + "/api/v1/login/access-token"

	formData := fmt.Sprintf("username=%s&password=%s", username, password)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(formData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	// 打印完整响应
	fmt.Printf("完整登录响应:\n")
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(body, &prettyJSON); err == nil {
		formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Printf("%s\n\n", string(formatted))
	}

	// 检查 Set-Cookie 响应头
	if cookies := resp.Cookies(); len(cookies) > 0 {
		fmt.Printf("收到的 Cookies:\n")
		for _, cookie := range cookies {
			fmt.Printf("  %s = %s\n", cookie.Name, cookie.Value)
		}
		fmt.Println()
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// testDownloadHistory 测试下载历史
func testDownloadHistory(ctx context.Context, baseURL, token string) error {
	url := baseURL + "/api/v1/history/download?page=1&page_size=5"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("✅ 成功！返回数据:\n%s\n\n", string(prettyJSON))
	return nil
}

// testTransferHistory 测试入库历史
func testTransferHistory(ctx context.Context, baseURL, token string) error {
	url := baseURL + "/api/v1/history/transfer?page=1&page_size=5"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("✅ 成功！返回数据:\n%s\n\n", string(prettyJSON))
	return nil
}

// testSSEWithBearer 使用 Bearer Token 测试 SSE
func testSSEWithBearer(ctx context.Context, baseURL, token string) error {
	url := baseURL + "/api/v1/system/message"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("  ✅ 连接成功！开始接收消息（5秒）...\n")
	reader := bufio.NewReader(resp.Body)

	timeout := time.After(5 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				done <- true
				return
			}
			fmt.Printf("  > %s", line)
		}
	}()

	select {
	case <-timeout:
		fmt.Printf("  ⏱️  超时，停止接收\n")
	case <-done:
		fmt.Printf("  🔚 连接关闭\n")
	}

	return nil
}

// testSSEWithCookie 使用 Cookie 测试 SSE
func testSSEWithCookie(ctx context.Context, baseURL, token string) error {
	url := baseURL + "/api/v1/system/message"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Cookie", fmt.Sprintf("resource_token=%s", token))
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("  ✅ 连接成功！开始接收消息（5秒）...\n")
	reader := bufio.NewReader(resp.Body)

	timeout := time.After(5 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				done <- true
				return
			}
			fmt.Printf("  > %s", line)
		}
	}()

	select {
	case <-timeout:
		fmt.Printf("  ⏱️  超时，停止接收\n")
	case <-done:
		fmt.Printf("  🔚 连接关闭\n")
	}

	return nil
}

// testSSEWithQuery 使用查询参数测试 SSE
func testSSEWithQuery(ctx context.Context, baseURL, token string) error {
	url := baseURL + "/api/v1/system/message?token=" + token

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("  ✅ 连接成功！开始接收消息（5秒）...\n")
	reader := bufio.NewReader(resp.Body)

	timeout := time.After(5 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				done <- true
				return
			}
			fmt.Printf("  > %s", line)
		}
	}()

	select {
	case <-timeout:
		fmt.Printf("  ⏱️  超时，停止接收\n")
	case <-done:
		fmt.Printf("  🔚 连接关闭\n")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
