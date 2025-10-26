# MoviePilot API 测试工具

这个工具用于测试和调试与 MoviePilot 的通信。

## 快速使用

### 方式 1：使用脚本（推荐）

```bash
# 确保 .env 文件配置正确
cat .env

# 运行测试脚本
./test-mp.sh
```

### 方式 2：手动运行

```bash
# 编译
go build -o build/test-mp ./cmd/test-mp

# 使用环境变量运行
export MP_URL="http://138.201.254.254:5000"
export MP_USERNAME="admin"
export MP_PASSWORD="your-password"

./build/test-mp
```

### 方式 3：从 .env 加载

```bash
# 加载环境变量
export $(cat .env | grep -v '^#' | xargs)

# 运行测试
./build/test-mp
```

## 测试内容

工具会按顺序执行以下测试：

### 1️⃣ 登录测试
- 测试用户名密码登录
- 获取 Access Token
- 验证 Token 格式

**预期输出：**
```
【步骤 1】测试登录...
✅ 登录成功！
Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### 2️⃣ 下载历史测试
- 测试 `/api/v1/history/download` 端点
- 使用 Bearer Token 认证
- 获取最近 5 条下载记录

**预期输出：**
```
【步骤 2】测试获取下载历史...
✅ 成功！返回数据:
{
  "total": 10,
  "items": [...]
}
```

### 3️⃣ 入库历史测试
- 测试 `/api/v1/history/transfer` 端点
- 使用 Bearer Token 认证
- 获取最近 5 条入库记录

**预期输出：**
```
【步骤 3】测试获取入库历史...
✅ 成功！返回数据:
{
  "total": 8,
  "items": [...]
}
```

### 4️⃣ SSE 连接测试

工具会尝试三种不同的认证方式连接 SSE：

#### 方式 1: Authorization Header (Bearer Token)
```http
GET /api/v1/system/message
Authorization: Bearer <token>
Accept: text/event-stream
```

#### 方式 2: Cookie
```http
GET /api/v1/system/message
Cookie: resource_token=<token>
Accept: text/event-stream
```

#### 方式 3: Query Parameter
```http
GET /api/v1/system/message?token=<token>
Accept: text/event-stream
```

**成功时的输出：**
```
【步骤 4】测试 SSE 连接...

  尝试方式 1: Authorization: Bearer <token>
  ✅ 连接成功！开始接收消息（5秒）...
  > :keepalive
  > data: {"message": {...}}
  >
  ⏱️  超时，停止接收
```

**失败时的输出：**
```
  尝试方式 1: Authorization: Bearer <token>
  ❌ 失败: status 403: {"detail":"resource token not found"}
```

## 常见问题排查

### 问题 1：登录失败

**错误：**
```
登录失败: status 401: Incorrect username or password
```

**解决：**
1. 检查 MP_URL 是否正确
2. 检查 MP_USERNAME 和 MP_PASSWORD 是否正确
3. 确认 MoviePilot 服务正常运行

### 问题 2：Token 认证失败

**错误：**
```
status 401: Not authenticated
```

**解决：**
1. Token 可能已过期，重新登录
2. 检查 Authorization header 格式

### 问题 3：SSE 连接 403

**错误：**
```
status 403: {"detail":"resource token not found"}
```

**解决：**
这是我们当前遇到的问题。工具会测试多种认证方式：
- Bearer Token（标准方式）
- Cookie（resource_token）
- Query Parameter

查看哪种方式能成功连接。

### 问题 4：网络连接失败

**错误：**
```
dial tcp: connect: connection refused
```

**解决：**
1. 检查 MP_URL 是否可访问
2. 检查防火墙设置
3. 确认 MoviePilot 服务运行中

## 查看实时 SSE 消息

如果 SSE 连接成功，工具会持续显示接收到的消息（5秒）：

```
  > :keepalive
  >
  > data: {"message": {"mtype": "订阅", "ctype": "subscribeComplete", ...}}
  >
  > data: {"message": {"mtype": "订阅", "ctype": "downloadStart", ...}}
  >
```

这些消息格式与您看到的 MP 日志一致：
- `subscribeAdded`: 订阅已添加
- `subscribeComplete`: 订阅完成（找到资源）
- `downloadStart`: 开始下载
- `downloadComplete`: 下载完成
- `transferComplete`: 入库完成

## 调试技巧

### 1. 查看完整响应

修改 `cmd/test-mp/main.go`，将超时时间改长：

```go
timeout := time.After(30 * time.Second)  // 改为 30 秒
```

### 2. 保存 Token 到文件

```bash
./build/test-mp 2>&1 | grep "Token:" | awk '{print $2}' > token.txt
```

### 3. 使用 curl 测试

```bash
# 获取 Token
TOKEN=$(./build/test-mp 2>&1 | grep "Token:" | awk '{print $2}')

# 测试 SSE（方式 1）
curl -N -H "Authorization: Bearer $TOKEN" \
     -H "Accept: text/event-stream" \
     "http://138.201.254.254:5000/api/v1/system/message"

# 测试 SSE（方式 2）
curl -N -H "Cookie: resource_token=$TOKEN" \
     -H "Accept: text/event-stream" \
     "http://138.201.254.254:5000/api/v1/system/message"

# 测试 SSE（方式 3）
curl -N -H "Accept: text/event-stream" \
     "http://138.201.254.254:5000/api/v1/system/message?token=$TOKEN"
```

### 4. 使用 websocat 测试（如果 MP 支持 WebSocket）

```bash
websocat "ws://138.201.254.254:5000/api/v1/system/message?token=$TOKEN"
```

## 下一步

1. 运行测试工具，确定哪种 SSE 认证方式有效
2. 查看成功方式的请求头和参数
3. 更新 `internal/tracker/sse.go` 使用正确的认证方式
4. 重新编译和测试

## 测试日志保存

```bash
# 保存完整测试日志
./test-mp.sh > test-results.log 2>&1

# 查看日志
cat test-results.log
```

## 帮助

如果所有方式都失败，请提供：
1. 完整的测试输出
2. MoviePilot 版本
3. MP 配置（隐藏敏感信息）
