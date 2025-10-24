#!/bin/bash

# MoviePilot 获取 Token 脚本

MP_URL="${MP_URL:-http://138.201.254.254:5000}"
USERNAME="${MP_USERNAME}"
PASSWORD="${MP_PASSWORD}"

if [ -z "$USERNAME" ] || [ -z "$PASSWORD" ]; then
    echo "错误：请设置 MP_USERNAME 和 MP_PASSWORD 环境变量"
    echo ""
    echo "使用方法："
    echo "  export MP_USERNAME='your_username'"
    echo "  export MP_PASSWORD='your_password'"
    echo "  ./get_mp_token.sh"
    exit 1
fi

echo "正在从 MoviePilot 获取 Token..."
echo "URL: $MP_URL"
echo "用户名: $USERNAME"
echo ""

# 调用登录 API
response=$(curl -s -X POST "$MP_URL/api/v1/login/access-token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=password&username=$USERNAME&password=$PASSWORD")

echo "API 响应:"
echo "$response" | jq . 2>/dev/null || echo "$response"
echo ""

# 提取 access_token
token=$(echo "$response" | jq -r '.access_token' 2>/dev/null)

if [ "$token" != "null" ] && [ -n "$token" ]; then
    echo "✅ Token 获取成功!"
    echo ""
    echo "请将以下内容添加到 .env 文件中："
    echo "MP_TOKEN=$token"
    echo ""
    echo "或者直接运行："
    echo "export MP_TOKEN='$token'"
else
    echo "❌ Token 获取失败"
    echo "请检查用户名和密码是否正确"
fi
