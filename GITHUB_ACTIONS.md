# GitHub Actions 自动构建 Docker 镜像

## 📦 功能说明

本项目已配置 GitHub Actions，可以自动构建并发布 Docker 镜像到 Docker Hub。

## 🔧 配置步骤

### 1. 准备 Docker Hub 账号

如果还没有 Docker Hub 账号：
1. 访问 https://hub.docker.com/
2. 注册账号
3. （可选）创建访问令牌：Settings → Security → New Access Token

### 2. 在 GitHub 仓库中配置 Secrets

1. 打开你的 GitHub 仓库
2. 进入 **Settings** → **Secrets and variables** → **Actions**
3. 点击 **New repository secret**，添加以下两个 secrets：

   - **名称**: `DOCKER_USERNAME`
     **值**: 你的 Docker Hub 用户名

   - **名称**: `DOCKER_PASSWORD`
     **值**: 你的 Docker Hub 密码或访问令牌（推荐使用令牌）

### 3. 推送代码触发构建

配置完成后，有以下几种方式触发自动构建：

#### 方式 1: 推送到 main 分支
```bash
git push origin main
```
- 会自动构建并推送 `latest` 标签

#### 方式 2: 创建版本标签
```bash
# 创建并推送标签
git tag v1.0.0
git push origin v1.0.0
```
- 会自动构建并推送 `v1.0.0` 和 `1.0` 标签

#### 方式 3: 手动触发
1. 进入 GitHub 仓库的 **Actions** 页面
2. 选择 "Build and Push Docker Image" 工作流
3. 点击 **Run workflow** 按钮

## 📊 查看构建状态

1. 进入 GitHub 仓库的 **Actions** 页面
2. 查看工作流运行状态
3. 点击具体的运行记录可以查看详细日志

## 🐳 使用构建的镜像

构建完成后，可以从 Docker Hub 拉取镜像：

```bash
# 拉取最新版本
docker pull <你的用户名>/jellyseerr-moviepilot-syncer:latest

# 拉取特定版本
docker pull <你的用户名>/jellyseerr-moviepilot-syncer:v1.0.0
```

## 🔄 更新镜像

每次推送代码或创建新标签时，GitHub Actions 会自动：
1. 检出代码
2. 设置 Docker Buildx
3. 登录 Docker Hub
4. 构建 Docker 镜像
5. 推送到 Docker Hub

整个过程通常需要 3-5 分钟。

## 📝 镜像标签规则

工作流会根据不同的触发条件创建不同的标签：

| 触发方式 | 生成的标签 | 示例 |
|---------|-----------|------|
| 推送到 main 分支 | `main`, `latest` | `username/image:latest` |
| 推送标签 v1.2.3 | `v1.2.3`, `1.2`, `latest` | `username/image:v1.2.3` |
| 推送 PR | `pr-123` | `username/image:pr-123` |

## 🛠️ 故障排查

### 构建失败

1. 检查 Actions 页面的错误日志
2. 确认 Secrets 配置正确
3. 确认 Docker Hub 用户名和密码有效

### 推送失败

1. 检查 Docker Hub 账号是否有推送权限
2. 如果使用访问令牌，确认令牌权限包含 "Read & Write"

### 镜像未更新

1. 确认工作流已成功运行（绿色对勾）
2. 等待几分钟，Docker Hub 同步需要时间
3. 使用 `docker pull` 时添加 `--no-cache` 参数

## 💡 高级配置

### 自定义镜像名称

编辑 `.github/workflows/docker-publish.yml`：
```yaml
env:
  IMAGE_NAME: 你的镜像名称  # 修改这里
```

### 只在标签推送时构建

修改触发条件：
```yaml
on:
  push:
    tags:
      - 'v*'
```

### 使用 GitHub Container Registry (ghcr.io)

如果不想使用 Docker Hub，可以使用 GitHub 自带的容器注册表，无需配置额外的 Secrets。

---

完成配置后，你的 Docker 镜像将自动构建和发布！🎉
