#!/bin/bash

# MoviePilot API 测试脚本

# 加载环境变量
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# 编译测试工具
echo "编译测试工具..."
go build -o build/test-mp ./cmd/test-mp

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"
echo ""

# 运行测试
./build/test-mp
