#!/bin/sh
set -e

# 确保数据目录存在且有正确权限
if [ -d "/app/data" ]; then
    echo "检查数据目录权限..."

    # 尝试创建测试文件来验证写入权限
    if ! touch /app/data/.write_test 2>/dev/null; then
        echo "警告: /app/data 目录没有写入权限"
        echo "请确保挂载的卷有正确的权限，或使用 root 用户运行容器"
        echo ""
        echo "解决方案："
        echo "1. 修改宿主机目录权限: sudo chown -R 1000:1000 ./data"
        echo "2. 或在 docker-compose.yml 中添加: user: \"0:0\""
        exit 1
    fi
    rm -f /app/data/.write_test
    echo "数据目录权限正常"
else
    echo "错误: /app/data 目录不存在"
    exit 1
fi

# 执行传入的命令
exec "$@"
