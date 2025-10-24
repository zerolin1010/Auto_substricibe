package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/jellyseerr-moviepilot-syncer/configs"
	"github.com/yourusername/jellyseerr-moviepilot-syncer/internal/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// 命令行参数
	var (
		showVersion = flag.Bool("version", false, "显示版本信息")
		mode        = flag.String("mode", "once", "运行模式: once (单次同步) 或 daemon (守护进程)")
		dryRun      = flag.Bool("dry-run", false, "干跑模式（仅打印，不实际创建订阅）")
	)
	flag.Parse()

	// 显示版本
	if *showVersion {
		fmt.Printf("Jellyseerr-MoviePilot-Syncer\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Commit:  %s\n", commit)
		fmt.Printf("Date:    %s\n", date)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := configs.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 命令行参数覆盖配置
	if *dryRun {
		cfg.MPDryRun = true
	}

	// 初始化日志
	logger, err := initLogger(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync() // 忽略 sync 错误
	}()

	// 创建上下文（需要在创建同步器之前）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 打印配置（屏蔽敏感信息）
	logger.Info("starting syncer",
		zap.String("version", version),
		zap.String("mode", *mode),
		zap.Any("config", cfg.MaskSensitive()),
	)

	// 创建同步器
	syncer, err := core.NewSyncer(cfg, logger, ctx)
	if err != nil {
		logger.Fatal("创建同步器失败", zap.Error(err))
	}
	defer syncer.Close()

	// 监听信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 启动同步
	errChan := make(chan error, 1)
	go func() {
		switch *mode {
		case "once":
			errChan <- syncer.SyncOnce(ctx)
		case "daemon":
			errChan <- syncer.RunDaemon(ctx)
		default:
			errChan <- fmt.Errorf("未知的运行模式: %s", *mode)
		}
	}()

	// 等待完成或信号
	select {
	case err := <-errChan:
		if err != nil {
			logger.Error("同步失败", zap.Error(err))
			os.Exit(1)
		}
		logger.Info("同步完成")
	case sig := <-sigChan:
		logger.Info("接收到信号，正在退出...", zap.String("signal", sig.String()))
		cancel()
		// 等待清理
		<-errChan
	}
}

// initLogger 初始化日志
func initLogger(level string) (*zap.Logger, error) {
	// 解析日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 配置日志
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// 自定义时间格式
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	return config.Build()
}
