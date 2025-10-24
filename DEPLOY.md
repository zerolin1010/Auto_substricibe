# 部署指南

## 📦 Docker 镜像

### 使用预构建镜像（推荐）

官方镜像已发布到 Docker Hub：`zerolin1010/jellyseerr-moviepilot-syncer`

直接使用 Docker Compose 或 Docker 命令拉取镜像即可：

```bash
# 使用 Docker Compose
docker-compose up -d

# 或直接拉取镜像
docker pull zerolin1010/jellyseerr-moviepilot-syncer:latest
```

## 🔨 自定义构建（开发者）

### 前置准备

1. 安装 Docker
2. 注册 Docker Hub 账号
3. 登录 Docker Hub：
   ```bash
   docker login
   ```

### 方式 1: 使用 GitHub Actions（推荐）

GitHub Actions 会自动构建和发布 Docker 镜像，详见 [GITHUB_ACTIONS.md](GITHUB_ACTIONS.md)

### 方式 2: 使用自动化脚本

```bash
# 1. 设置你的 Docker Hub 用户名
export DOCKER_USERNAME=your-dockerhub-username

# 2. 设置版本号（可选）
export VERSION=v1.0.0

# 3. 运行脚本
./docker-build-push.sh
```

脚本会：
- 自动构建 Docker 镜像
- 询问是否推送到 Docker Hub
- 打标签为 `latest` 和指定版本

### 方式 3: 手动构建和发布

```bash
# 1. 构建镜像
docker build -t zerolin1010/jellyseerr-moviepilot-syncer:latest .

# 2. 打标签
docker tag zerolin1010/jellyseerr-moviepilot-syncer:latest \
           zerolin1010/jellyseerr-moviepilot-syncer:v1.0.0

# 3. 推送到 Docker Hub
docker push zerolin1010/jellyseerr-moviepilot-syncer:latest
docker push zerolin1010/jellyseerr-moviepilot-syncer:v1.0.0
```

### 验证镜像

```bash
# 拉取镜像
docker pull zerolin1010/jellyseerr-moviepilot-syncer:latest

# 查看镜像信息
docker images | grep jellyseerr-moviepilot-syncer

# 测试运行
docker run --rm \
  -e JELLY_URL=https://test.com \
  -e JELLY_API_KEY=test \
  -e MP_URL=http://test.com \
  -e MP_USERNAME=test \
  -e MP_PASSWORD=test \
  your-username/jellyseerr-moviepilot-syncer:latest \
  -version
```

## 🚀 生产环境部署

### 使用 Docker Compose

1. 创建 `.env` 文件：
   ```bash
   cp .env.example .env
   # 编辑 .env 文件填入实际配置
   ```

2. 修改 `docker-compose.yml` 中的镜像名称：
   ```yaml
   image: your-username/jellyseerr-moviepilot-syncer:latest
   ```

3. 启动服务：
   ```bash
   docker-compose up -d
   ```

4. 查看日志：
   ```bash
   docker-compose logs -f
   ```

### 使用 Docker Run

```bash
docker run -d \
  --name jellyseerr-moviepilot-syncer \
  --restart unless-stopped \
  -e JELLY_URL=https://your-jellyseerr.com \
  -e JELLY_API_KEY=your-api-key \
  -e MP_URL=http://your-moviepilot.com:5000 \
  -e MP_USERNAME=your-username \
  -e MP_PASSWORD=your-password \
  -e MP_TOKEN_REFRESH_HOURS=24 \
  -e SYNC_INTERVAL=5 \
  -e LOG_LEVEL=info \
  -v /path/to/data:/app/data \
  your-username/jellyseerr-moviepilot-syncer:latest
```

## 🔧 配置说明

### 必需配置

```env
# Jellyseerr
JELLY_URL=https://your-jellyseerr.com
JELLY_API_KEY=your-api-key

# MoviePilot
MP_URL=http://your-moviepilot.com:5000
MP_USERNAME=your-username
MP_PASSWORD=your-password
```

### 可选配置

```env
# Token 刷新间隔（小时），默认 24
MP_TOKEN_REFRESH_HOURS=24

# 同步间隔（分钟），默认 5
SYNC_INTERVAL=5

# 日志级别: debug/info/warn/error
LOG_LEVEL=info

# 干跑模式（测试用）
MP_DRY_RUN=false
```

## 📊 监控和维护

### 查看日志

```bash
# Docker Compose
docker-compose logs -f

# Docker Run
docker logs -f jellyseerr-moviepilot-syncer
```

### 查看统计

日志中会定期输出同步统计：
```
sync completed {"total": 10, "pending": 0, "synced": 10, "failed": 0}
```

### 重启服务

```bash
# Docker Compose
docker-compose restart

# Docker Run
docker restart jellyseerr-moviepilot-syncer
```

### 更新镜像

```bash
# Docker Compose
docker-compose pull
docker-compose up -d

# Docker Run
docker pull your-username/jellyseerr-moviepilot-syncer:latest
docker stop jellyseerr-moviepilot-syncer
docker rm jellyseerr-moviepilot-syncer
# 重新运行 docker run 命令
```

## 🔒 安全建议

1. **不要将 `.env` 文件提交到 Git**
   - 已添加到 `.gitignore`
   - 包含敏感的用户名和密码

2. **使用强密码**
   - MoviePilot 密码应足够复杂

3. **限制网络访问**
   - 如果可能，使用内网部署
   - 配置防火墙规则

4. **定期更新**
   - 及时更新到最新版本
   - 关注安全公告

## 📝 故障排查

### 问题 1: 容器启动失败

检查日志：
```bash
docker logs jellyseerr-moviepilot-syncer
```

常见原因：
- 配置错误（检查 `.env` 文件）
- 端口冲突
- 权限问题

### 问题 2: 无法连接 MoviePilot

1. 检查 MP_URL 是否正确
2. 检查用户名密码是否正确
3. 尝试手动登录 MoviePilot 验证

### 问题 3: Token 刷新失败

查看日志中的详细错误信息：
```bash
docker logs jellyseerr-moviepilot-syncer 2>&1 | grep -i "token\|login"
```

### 问题 4: 查看数据库

如果需要检查数据库状态：
```bash
# 进入容器
docker exec -it jellyseerr-moviepilot-syncer sh

# 查看数据库（需要安装 sqlite3）
# 或者直接查看挂载的数据文件
```

## 🎯 性能优化

### 调整同步间隔

对于大量请求，可以减少同步间隔：
```env
SYNC_INTERVAL=1  # 每分钟同步一次
```

### 调整速率限制

如果 MoviePilot 性能足够，可以提高速率：
```env
MP_RATE_LIMIT_PER_SEC=5  # 每秒 5 个请求
```

### 资源限制

在 `docker-compose.yml` 中添加资源限制：
```yaml
services:
  syncer:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 256M
        reservations:
          memory: 128M
```

## 📈 扩展部署

### 多实例部署

**不推荐**同时运行多个实例，因为：
- SQLite 不支持并发写入
- 可能导致重复订阅

如果确实需要，请：
1. 使用不同的数据库文件
2. 配置不同的 `JELLY_FILTER`
3. 确保不处理相同的请求

---

完成部署后，建议：
1. 观察日志几分钟确保正常运行
2. 在 Jellyseerr 中添加测试请求验证
3. 检查 MoviePilot 中是否成功创建订阅
