# Jellyseerr → MoviePilot 自动订阅同步器

一个使用 Go 编写的可靠服务，自动将 Jellyseerr/Overseerr 中已批准的请求同步到 MoviePilot，触发其检索和下载流程。

## ✨ 特性

- 🔄 **自动同步**：自动从 Jellyseerr/Overseerr 拉取已批准的请求
- 💾 **本地队列**：使用 SQLite 存储请求，保证幂等性和可靠性
- 📺 **完整支持**：支持电影和剧集（含多季、按集订阅）
- 🔁 **智能重试**：自动重试失败的请求，支持指数退避
- 🚦 **速率限制**：内置速率限制，避免 API 过载
- 🔒 **安全**：日志中自动屏蔽敏感信息
- 🐳 **容器化**：提供 Docker 和 Docker Compose 支持
- 🔑 **自动登录**：使用用户名密码自动获取和刷新 Token

## 📋 前置要求

- Docker 和 Docker Compose (推荐)
- 或 Go 1.22+ (如果从源码构建)
- Jellyseerr/Overseerr 实例及 API Key
- MoviePilot 实例的用户名和密码

## 🚀 快速开始

### 使用 Docker Compose（推荐）

1. 克隆仓库：
```bash
git clone <repo-url>
cd jellyseerr-moviepilot-syncer
```

2. 创建环境配置文件：
```bash
cp .env.example .env
```

3. 编辑 `.env` 文件，填入你的配置：
```bash
# Jellyseerr 配置
JELLY_URL=https://your-jellyseerr.com
JELLY_API_KEY=your-api-key-here

# MoviePilot 配置
MP_URL=http://your-moviepilot.com:5000
MP_USERNAME=your-username
MP_PASSWORD=your-password
MP_TOKEN_REFRESH_HOURS=24
```

4. 启动服务：
```bash
docker-compose up -d
```

5. 查看日志：
```bash
docker-compose logs -f
```

### 使用 Docker 镜像

```bash
# 拉取镜像
docker pull your-dockerhub-username/jellyseerr-moviepilot-syncer:latest

# 运行
docker run -d \
  --name jellyseerr-moviepilot-syncer \
  -e JELLY_URL=https://your-jellyseerr.com \
  -e JELLY_API_KEY=your-api-key \
  -e MP_URL=http://your-moviepilot.com:5000 \
  -e MP_USERNAME=your-username \
  -e MP_PASSWORD=your-password \
  -v ./data:/app/data \
  your-dockerhub-username/jellyseerr-moviepilot-syncer:latest
```

### 使用源码构建

1. 克隆仓库并安装依赖：
```bash
git clone <repo-url>
cd jellyseerr-moviepilot-syncer
go mod download
```

2. 编译：
```bash
make build
```

3. 设置环境变量并运行：
```bash
cp .env.example .env
# 编辑 .env 文件
./build/syncer -mode=daemon
```

## ⚙️ 配置说明

### 环境变量

| 环境变量 | 说明 | 默认值 | 必需 |
|---------|------|--------|------|
| `JELLY_URL` | Jellyseerr/Overseerr 地址 | - | ✅ |
| `JELLY_API_KEY` | API 密钥 | - | ✅ |
| `JELLY_FILTER` | 请求过滤器 | `approved` | ❌ |
| `JELLY_PAGE_SIZE` | 分页大小 | `50` | ❌ |
| `MP_URL` | MoviePilot 地址 | - | ✅ |
| `MP_USERNAME` | MoviePilot 用户名 | - | ✅ |
| `MP_PASSWORD` | MoviePilot 密码 | - | ✅ |
| `MP_TOKEN_REFRESH_HOURS` | Token 刷新间隔（小时） | `24` | ❌ |
| `MP_AUTH_SCHEME` | 认证方案 | `bearer` | ❌ |
| `MP_RATE_LIMIT_PER_SEC` | 每秒请求限制 | `3` | ❌ |
| `MP_DRY_RUN` | 干跑模式 | `false` | ❌ |
| `MP_TV_EPISODE_MODE` | 剧集模式 | `season` | ❌ |
| `STORE_TYPE` | 存储类型 | `sqlite` | ❌ |
| `STORE_PATH` | 存储路径 | `./data/syncer.db` | ❌ |
| `SYNC_INTERVAL` | 同步间隔（分钟） | `5` | ❌ |
| `ENABLE_RETRY` | 启用重试 | `true` | ❌ |
| `MAX_RETRIES` | 最大重试次数 | `3` | ❌ |
| `LOG_LEVEL` | 日志级别 | `info` | ❌ |

## 🔧 工作原理

```
┌─────────────┐       ┌──────────────┐       ┌────────────┐
│  Jellyseerr │──────▶│ 本地 SQLite  │──────▶│ MoviePilot │
│   已批准队列 │       │   请求队列    │       │   订阅系统  │
└─────────────┘       └──────────────┘       └────────────┘
```

1. **拉取阶段**：从 Jellyseerr 获取已批准的请求
2. **存储阶段**：保存到本地 SQLite 数据库（幂等）
3. **登录阶段**：使用用户名密码自动获取 MoviePilot Token
4. **同步阶段**：逐条转换并推送到 MoviePilot
5. **刷新阶段**：定期刷新 Token（默认 24 小时）

## 📖 使用说明

### 运行模式

#### 守护进程模式（推荐）
持续运行，定时同步：
```bash
./syncer -mode=daemon
```

#### 单次同步模式
执行一次完整同步后退出：
```bash
./syncer -mode=once
```

#### 干跑模式
测试配置，不实际创建订阅：
```bash
./syncer -mode=once -dry-run
```

### 命令行参数

- `-mode`: 运行模式（`once` 或 `daemon`）
- `-dry-run`: 干跑模式
- `-version`: 显示版本信息

## 🛠️ 开发

### 项目结构

```
.
├── cmd/syncer/          # 主程序入口
├── internal/
│   ├── jelly/          # Jellyseerr 客户端
│   ├── mp/             # MoviePilot 客户端（含 Token 管理）
│   ├── store/          # 存储层
│   └── core/           # 核心同步逻辑
├── configs/            # 配置管理
├── Dockerfile          # Docker 镜像
├── docker-compose.yml  # Docker Compose 配置
└── Makefile           # 构建工具
```

### 构建命令

```bash
make build              # 编译
make run                # 运行（单次）
make run-daemon         # 运行（守护进程）
make test               # 测试
make docker-build       # 构建 Docker 镜像
```

### 构建并发布 Docker 镜像

#### 方式 1: 使用 GitHub Actions（推荐）

最简单的方式是使用 GitHub Actions 自动构建和发布：

1. 在 GitHub 仓库设置中配置 Docker Hub Secrets
2. 推送代码或创建标签即可自动构建

详细步骤请参考 [GITHUB_ACTIONS.md](GITHUB_ACTIONS.md)

#### 方式 2: 本地手动构建

```bash
# 设置 Docker Hub 用户名
export DOCKER_USERNAME=your-dockerhub-username

# 设置版本号（可选）
export VERSION=v1.0.0

# 运行构建和发布脚本
./docker-build-push.sh
```

## ❓ 常见问题

### Q: 如何获取 Jellyseerr API Key？
A: 登录 Jellyseerr → 设置 → API → 复制 API Key

### Q: MoviePilot 的用户名和密码在哪里？
A: 登录 MoviePilot 时使用的用户名和密码

### Q: Token 多久刷新一次？
A: 默认 24 小时自动刷新，可通过 `MP_TOKEN_REFRESH_HOURS` 配置

### Q: 请求没有同步？
A: 检查：
1. 日志中是否有错误信息
2. 请求在 Jellyseerr 中是否为 "已批准" 状态
3. MoviePilot 用户名密码是否正确
4. 使用 `-dry-run` 模式测试

### Q: 如何查看同步统计？
A: 查看日志，每次同步完成后会打印统计信息

## 📝 许可证

MIT License

## 🙏 致谢

- [Jellyseerr](https://github.com/juandjara/jellyseer)
- [Overseerr](https://github.com/sct/overseerr)
- [MoviePilot](https://github.com/jxxghp/MoviePilot)

## 📮 支持

如有问题或建议，请提交 Issue。
