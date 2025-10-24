# 快速开始指南

## 前置准备

在开始之前，请确保你已经：

1. 安装了 Go 1.22 或更高版本（或使用 Docker）
2. 拥有 Jellyseerr/Overseerr 实例的 API Key
3. 拥有 MoviePilot 实例的 API Token

## 方式一：使用 Docker Compose（推荐）

### 1. 准备配置文件

复制示例配置文件：
```bash
cp .env.example .env
```

编辑 `.env` 文件，填入你的实际配置：
```bash
# Jellyseerr 配置
JELLY_URL=https://your-jellyseerr.com
JELLY_API_KEY=your-api-key-here

# MoviePilot 配置
MP_URL=http://your-moviepilot.com:5000
MP_TOKEN=your-token-here
```

### 2. 启动服务

```bash
# 启动（后台运行）
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 3. 查看同步状态

```bash
# 查看数据库统计
docker-compose exec syncer sqlite3 /app/data/syncer.db "SELECT status, COUNT(*) FROM requests GROUP BY status;"
```

## 方式二：从源码运行

### 1. 克隆并构建

```bash
git clone <repo-url>
cd jellyseerr-moviepilot-syncer
go mod download
make build
```

### 2. 配置环境变量

创建 `.env` 文件（参考 .env.example），然后：

```bash
# Linux/macOS
source .env

# 或者直接导出
export JELLY_URL=https://your-jellyseerr.com
export JELLY_API_KEY=your-api-key
export MP_URL=http://your-moviepilot.com:5000
export MP_TOKEN=your-token
```

### 3. 运行程序

```bash
# 单次同步（测试用）
./build/syncer -mode=once

# 守护进程模式
./build/syncer -mode=daemon

# 干跑模式（不实际创建订阅）
./build/syncer -mode=once -dry-run
```

或使用 Makefile：
```bash
make run           # 单次同步
make run-daemon    # 守护进程
make run-dry       # 干跑模式
```

## 测试连接

### 测试 Jellyseerr 连接

```bash
curl -H "X-Api-Key: YOUR_API_KEY" \
     "YOUR_JELLYSEERR_URL/api/v1/request?take=1&skip=0&filter=approved"
```

### 测试 MoviePilot 连接

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
     "YOUR_MOVIEPILOT_URL/api/v1/subscribe" \
     -X POST \
     -H "Content-Type: application/json" \
     -d '{}'
```

## 常见配置场景

### 场景 1：首次使用（测试配置）

```bash
# 设置为干跑模式
export MP_DRY_RUN=true

# 运行一次同步，检查输出
./build/syncer -mode=once -dry-run
```

### 场景 2：生产环境（守护进程）

```bash
# 关闭干跑模式
export MP_DRY_RUN=false

# 设置同步间隔为 10 分钟
export SYNC_INTERVAL=10

# 启动守护进程
./build/syncer -mode=daemon
```

### 场景 3：处理剧集按集订阅

```bash
# 设置为按集模式
export MP_TV_EPISODE_MODE=episode

# 运行同步
./build/syncer -mode=once
```

### 场景 4：调试模式

```bash
# 设置日志级别为 debug
export LOG_LEVEL=debug

# 运行同步
./build/syncer -mode=once
```

## 查看同步结果

### 方式 1：查看日志

```bash
# Docker
docker-compose logs -f

# 直接运行
# 日志会输出到控制台
```

### 方式 2：查询数据库

```bash
# 进入数据库
sqlite3 ./data/syncer.db

# 查看所有请求
SELECT * FROM requests;

# 查看待处理请求
SELECT * FROM requests WHERE status = 'pending';

# 查看失败请求
SELECT * FROM requests WHERE status = 'failed';

# 查看同步链接
SELECT * FROM mp_links;

# 统计信息
SELECT status, COUNT(*) as count FROM requests GROUP BY status;
```

## 故障排查

### 问题：配置加载失败

检查环境变量是否正确设置：
```bash
echo $JELLY_URL
echo $MP_URL
```

### 问题：无法连接到 Jellyseerr

1. 检查 URL 是否正确（包括 https:// 前缀）
2. 检查 API Key 是否有效
3. 测试网络连接：`curl -v YOUR_JELLYSEERR_URL`

### 问题：无法连接到 MoviePilot

1. 检查 URL 是否可访问
2. 检查 Token 是否有效
3. 检查认证方案（bearer/x-api-token/query-token）

### 问题：速率限制错误

降低请求频率：
```bash
export MP_RATE_LIMIT_PER_SEC=1
```

### 问题：查看详细错误信息

启用调试日志：
```bash
export LOG_LEVEL=debug
./build/syncer -mode=once
```

## 停止服务

### Docker
```bash
docker-compose down
```

### 直接运行
按 `Ctrl+C` 优雅退出

## 下一步

- 阅读完整文档：[README.md](README.md)
- 查看配置选项：[.env.example](.env.example)
- 提交问题：GitHub Issues
