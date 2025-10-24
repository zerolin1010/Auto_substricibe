#!/bin/bash

# Docker 构建和发布脚本

set -e

# 配置
DOCKER_USERNAME="${DOCKER_USERNAME:-your-dockerhub-username}"
IMAGE_NAME="jellyseerr-moviepilot-syncer"
VERSION="${VERSION:-latest}"

# 完整镜像名
FULL_IMAGE_NAME="${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}"

echo "========================================"
echo "Docker 构建和发布脚本"
echo "========================================"
echo "镜像名称: ${FULL_IMAGE_NAME}"
echo ""

# 构建镜像
echo "📦 正在构建 Docker 镜像..."
docker build \
  --build-arg VERSION="${VERSION}" \
  --build-arg COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
  --build-arg DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
  -t "${FULL_IMAGE_NAME}" \
  -t "${DOCKER_USERNAME}/${IMAGE_NAME}:latest" \
  .

echo "✅ 镜像构建成功！"
echo ""

# 询问是否推送
read -p "是否推送到 Docker Hub? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]
then
    echo "📤 正在推送到 Docker Hub..."
    docker push "${FULL_IMAGE_NAME}"
    docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:latest"
    echo "✅ 推送成功！"
    echo ""
    echo "镜像地址:"
    echo "  docker pull ${FULL_IMAGE_NAME}"
    echo "  docker pull ${DOCKER_USERNAME}/${IMAGE_NAME}:latest"
else
    echo "⏭️  跳过推送"
    echo ""
    echo "本地镜像:"
    echo "  ${FULL_IMAGE_NAME}"
fi

echo ""
echo "========================================"
echo "完成！"
echo "========================================"
