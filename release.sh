#!/bin/bash
set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Jellyseerr-MoviePilot-Syncer 版本发布工具 ===${NC}"
echo ""

# 检查是否有未提交的更改
if [[ -n $(git status -s) ]]; then
    echo -e "${RED}错误: 存在未提交的更改${NC}"
    echo "请先提交所有更改后再发布版本"
    exit 1
fi

# 获取当前分支
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    echo -e "${YELLOW}警告: 当前不在 main 分支 (当前: $CURRENT_BRANCH)${NC}"
    read -p "是否继续? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 获取最新的标签
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo -e "当前最新版本: ${GREEN}$LATEST_TAG${NC}"
echo ""

# 解析版本号
VERSION=${LATEST_TAG#v}
IFS='.' read -r -a VERSION_PARTS <<< "$VERSION"
MAJOR=${VERSION_PARTS[0]:-0}
MINOR=${VERSION_PARTS[1]:-0}
PATCH=${VERSION_PARTS[2]:-0}

# 显示版本选项
echo "请选择版本更新类型:"
echo "  1) Patch (修复): v${MAJOR}.${MINOR}.$((PATCH + 1))"
echo "  2) Minor (功能): v${MAJOR}.$((MINOR + 1)).0"
echo "  3) Major (重大): v$((MAJOR + 1)).0.0"
echo "  4) 自定义版本号"
echo ""

read -p "选择 (1-4): " -n 1 -r CHOICE
echo ""

case $CHOICE in
    1)
        NEW_VERSION="v${MAJOR}.${MINOR}.$((PATCH + 1))"
        ;;
    2)
        NEW_VERSION="v${MAJOR}.$((MINOR + 1)).0"
        ;;
    3)
        NEW_VERSION="v$((MAJOR + 1)).0.0"
        ;;
    4)
        read -p "请输入版本号 (格式: v1.0.0): " NEW_VERSION
        if [[ ! $NEW_VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            echo -e "${RED}错误: 版本号格式不正确${NC}"
            exit 1
        fi
        ;;
    *)
        echo -e "${RED}无效选择${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "即将发布新版本: ${GREEN}${NEW_VERSION}${NC}"
echo ""

# 询问更新说明
read -p "请输入版本更新说明: " RELEASE_NOTE

# 确认发布
echo ""
echo -e "${YELLOW}发布摘要:${NC}"
echo "  版本号: $NEW_VERSION"
echo "  说明: $RELEASE_NOTE"
echo ""
read -p "确认发布? (y/N): " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "已取消发布"
    exit 1
fi

# 创建标签
echo -e "${GREEN}创建 Git 标签...${NC}"
git tag -a "$NEW_VERSION" -m "$RELEASE_NOTE"

# 推送标签
echo -e "${GREEN}推送标签到 GitHub...${NC}"
git push origin "$NEW_VERSION"

echo ""
echo -e "${GREEN}✓ 版本 $NEW_VERSION 发布成功!${NC}"
echo ""
echo "GitHub Actions 将自动构建并推送以下 Docker 镜像:"
echo "  - zerolin1010/jellyseerr-moviepilot-syncer:$NEW_VERSION"
echo "  - zerolin1010/jellyseerr-moviepilot-syncer:${NEW_VERSION%.*}"
echo "  - zerolin1010/jellyseerr-moviepilot-syncer:latest"
echo ""
echo "查看构建状态: https://github.com/你的用户名/jellyseerr-moviepilot-syncer/actions"
echo ""
