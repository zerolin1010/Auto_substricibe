package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	// 打开数据库
	db, err := sql.Open("sqlite", "./data/syncer.db")
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	// 查询统计
	var total, pending, synced, failed int
	err = db.QueryRow(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' OR status = 'retrying' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'synced' THEN 1 ELSE 0 END) as synced,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
		FROM requests
	`).Scan(&total, &pending, &synced, &failed)
	if err != nil {
		log.Fatalf("查询统计失败: %v", err)
	}

	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║         数据库同步状态                     ║")
	fmt.Println("╚═══════════════════════════════════════════╝")
	fmt.Printf("  总请求数: %d\n", total)
	fmt.Printf("  待处理: %d\n", pending)
	fmt.Printf("  已同步: %d\n", synced)
	fmt.Printf("  失败: %d\n", failed)
	fmt.Println()

	// 查询所有请求
	rows, err := db.Query(`
		SELECT id, source_request_id, media_type, tmdb_id, title, status, requested_at
		FROM requests
		ORDER BY id DESC
	`)
	if err != nil {
		log.Fatalf("查询请求失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║         所有请求列表                       ║")
	fmt.Println("╚═══════════════════════════════════════════╝")
	for rows.Next() {
		var id int64
		var sourceID, mediaType, title, status, requestedAt string
		var tmdbID int
		if err := rows.Scan(&id, &sourceID, &mediaType, &tmdbID, &title, &status, &requestedAt); err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}
		fmt.Printf("\n[请求 #%s]\n", sourceID)
		fmt.Printf("  标题: %s\n", title)
		fmt.Printf("  类型: %s\n", mediaType)
		fmt.Printf("  TMDB ID: %d\n", tmdbID)
		fmt.Printf("  状态: %s\n", status)
		fmt.Printf("  请求时间: %s\n", requestedAt)
	}
	fmt.Println()

	// 查询 MP 链接和错误
	linkRows, err := db.Query(`
		SELECT source_request_id, mp_subscribe_id, state, last_error, retry_count
		FROM mp_links
	`)
	if err != nil {
		log.Printf("查询 MP 链接失败: %v", err)
		return
	}
	defer linkRows.Close()

	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║      MoviePilot 订阅链接详情               ║")
	fmt.Println("╚═══════════════════════════════════════════╝")
	hasLinks := false
	for linkRows.Next() {
		hasLinks = true
		var sourceID, mpID, state, lastError string
		var retryCount int
		var mpIDNullable sql.NullString
		if err := linkRows.Scan(&sourceID, &mpIDNullable, &state, &lastError, &retryCount); err != nil {
			log.Printf("扫描链接失败: %v", err)
			continue
		}
		if mpIDNullable.Valid {
			mpID = mpIDNullable.String
		} else {
			mpID = "(空)"
		}

		fmt.Printf("\n[链接 #%s]\n", sourceID)
		fmt.Printf("  MP 订阅 ID: %s\n", mpID)
		fmt.Printf("  状态: %s\n", state)
		fmt.Printf("  重试次数: %d\n", retryCount)
		if lastError != "" {
			fmt.Printf("  错误信息: %s\n", lastError)
		}
	}
	if !hasLinks {
		fmt.Println("  (无订阅链接)")
	}
	fmt.Println()
}
