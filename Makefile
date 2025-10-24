.PHONY: build run test clean docker-build docker-run help

# 版本信息
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go 相关
BINARY_NAME = syncer
MAIN_PATH = ./cmd/syncer
BUILD_DIR = ./build

# 构建标志
LDFLAGS = -ldflags "\
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)"

help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## 编译程序
	@echo "编译 $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "编译完成: $(BUILD_DIR)/$(BINARY_NAME)"

run: ## 运行程序 (单次同步)
	go run $(LDFLAGS) $(MAIN_PATH) -mode=once

run-daemon: ## 运行程序 (守护进程模式)
	go run $(LDFLAGS) $(MAIN_PATH) -mode=daemon

run-dry: ## 运行程序 (干跑模式)
	go run $(LDFLAGS) $(MAIN_PATH) -mode=once -dry-run

test: ## 运行测试
	go test -v -race ./...

tidy: ## 整理依赖
	go mod tidy
	go mod verify

clean: ## 清理构建文件
	@echo "清理构建文件..."
	rm -rf $(BUILD_DIR)
	rm -rf ./data
	@echo "清理完成"

docker-build: ## 构建 Docker 镜像
	docker build -t jellyseerr-moviepilot-syncer:$(VERSION) .

docker-run: ## 运行 Docker 容器
	docker run --rm \
		--env-file .env \
		-v $(PWD)/data:/app/data \
		jellyseerr-moviepilot-syncer:$(VERSION)

fmt: ## 格式化代码
	go fmt ./...

vet: ## 代码检查
	go vet ./...

lint: ## 运行 linter
	@which golangci-lint > /dev/null || (echo "请安装 golangci-lint" && exit 1)
	golangci-lint run ./...

deps: ## 安装依赖
	go mod download

install: build ## 安装到 GOPATH/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

version: ## 显示版本信息
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"
