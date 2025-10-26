# MoviePilot API 测试结果报告

**测试日期**: 2025-10-26
**MP 版本**: 最新版本
**MP URL**: http://138.201.254.254:5000

---

## 🧪 测试摘要

| 测试项 | 状态 | 详情 |
|--------|------|------|
| 登录 API | ✅ 成功 | Bearer Token 正常获取 |
| 入库历史 API | ✅ 成功 | 返回完整数据 |
| 下载历史 API | ⚠️  格式问题 | 返回数组而非对象 |
| SSE (Bearer) | ❌ 失败 | 403: resource token not found |
| SSE (Cookie) | ❌ 失败 | 403: resource token not found |
| SSE (Query) | ❌ 失败 | 403: resource token not found |

---

## 📋 详细测试结果

### 1️⃣ 登录测试

**端点**: `POST /api/v1/login/access-token`

**请求**:
```
Content-Type: application/x-www-form-urlencoded
username=admin&password=xxx
```

**响应** ✅:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer",
  "user_id": 1,
  "user_name": "admin",
  "super_user": true,
  "level": 2,
  "avatar": "http://...",
  "permissions": {},
  "widzard": false
}
```

**Cookie**: 无 Set-Cookie header
**结论**: ✅ 可以正常获取 access_token

---

### 2️⃣ 入库历史测试

**端点**: `GET /api/v1/history/transfer?page=1&page_size=5`

**认证**: `Authorization: Bearer <token>`

**响应** ✅:
```json
{
  "success": true,
  "message": null,
  "data": {
    "total": 663,
    "list": [
      {
        "id": 663,
        "title": "天地剑心",
        "tmdbid": 240442,
        "type": "电视剧",
        "year": "2025",
        "status": true,
        "date": "2025-10-26 19:48:36",
        ...
      }
    ]
  }
}
```

**结论**: ✅ API 正常工作，可用于轮询

---

### 3️⃣ 下载历史测试

**端点**: `GET /api/v1/history/download?page=1&page_size=5`

**认证**: `Authorization: Bearer <token>`

**响应** ⚠️:
```
json: cannot unmarshal array into Go value of type map[string]interface {}
```

**问题**: 返回的是数组，而不是像 transfer API 那样的对象格式

**结论**: ⚠️  需要调整数据结构定义

---

### 4️⃣ SSE 连接测试

**端点**: `GET /api/v1/system/message`

#### 方式 1: Authorization Header

**请求**:
```
Authorization: Bearer <access_token>
Accept: text/event-stream
Cache-Control: no-cache
```

**响应** ❌:
```json
{
  "detail": "resource token not found"
}
```
**状态码**: 403

---

#### 方式 2: Cookie

**请求**:
```
Cookie: resource_token=<access_token>
Accept: text/event-stream
```

**响应** ❌:
```json
{
  "detail": "resource token not found"
}
```
**状态码**: 403

---

#### 方式 3: Query Parameter

**请求**:
```
GET /api/v1/system/message?token=<access_token>
Accept: text/event-stream
```

**响应** ❌:
```json
{
  "detail": "resource token not found"
}
```
**状态码**: 403

---

## 🔍 根本原因分析

### SSE 认证问题

1. **API 文档说明**:
   - SSE 端点的 security 定义: `{"resource_token_cookie":[]}`
   - 需要名为 `resource_token` 的 Cookie

2. **登录响应**:
   - ✅ 返回 `access_token` (JWT)
   - ❌ 不返回 `resource_token`
   - ❌ 没有 Set-Cookie header

3. **API 文档缺失**:
   - ❌ 没有获取 `resource_token` 的端点
   - ❌ 没有说明如何生成 `resource_token`

### 推测

`resource_token` 可能是：
- Web UI 专用的会话 token
- 在浏览器登录时通过不同机制生成
- 不对外部 API 调用开放
- 或需要额外的配置/插件

---

## ✅ 解决方案

### 当前方案：使用轮询

**优点**:
- ✅ API 完全可用（入库历史已验证）
- ✅ 无需复杂的认证机制
- ✅ 5 分钟间隔对媒体下载足够实时
- ✅ 更稳定可靠

**配置**:
```bash
TRACKER_ENABLED=true
TRACKER_CHECK_INTERVAL=5  # 5 分钟
TRACKER_SSE_ENABLED=false  # 禁用 SSE
```

**轮询流程**:
```
每 5 分钟:
1. 获取入库历史
2. 匹配 TMDB ID
3. 更新订阅状态
4. 发送 Telegram 通知
```

---

## 🚀 部署建议

### 更新配置

编辑 `.env` 文件：
```bash
# 禁用 SSE
TRACKER_SSE_ENABLED=false

# 轮询间隔（分钟）
TRACKER_CHECK_INTERVAL=5
```

### 重新部署

```bash
cd /opt/jellyseerr-moviepilot-sync
git pull
docker-compose down
docker-compose build
docker-compose up -d
```

### 验证日志

```bash
docker-compose logs -f syncer | grep -E "(Tracker|SSE|Polling)"
```

**预期输出**:
```
INFO  Starting tracker  check_interval_minutes=5 sse_enabled=false
INFO  SSE disabled, using polling only
INFO  Polling checker started  interval_minutes=5
```

---

## 📊 性能对比

| 方式 | 延迟 | 资源消耗 | 状态 |
|------|------|----------|------|
| SSE | < 1 秒 | 长连接 | ❌ 不可用 |
| 轮询 (5 分钟) | 0-5 分钟 | 定期请求 | ✅ 可用 |

**结论**: 对于媒体下载场景，5 分钟延迟完全可接受。

---

## 🔮 未来改进

### 短期

- [x] 使用轮询代替 SSE
- [ ] 修复下载历史 API 数据结构
- [ ] 优化轮询逻辑避免重复处理

### 长期

- [ ] 联系 MP 开发者了解 `resource_token` 获取方法
- [ ] 请求 MP 支持 Bearer Token 认证 SSE
- [ ] 或者逆向工程 Web UI 的认证流程

---

## 📝 测试命令

### 重新运行测试

```bash
cd /mnt/d/Desktop/Go/Auto_substricibe
./test-mp.sh
```

### 手动 cURL 测试

```bash
# 登录
TOKEN=$(curl -s -X POST "http://138.201.254.254:5000/api/v1/login/access-token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin&password=xxx" | jq -r '.access_token')

# 测试入库历史
curl -H "Authorization: Bearer $TOKEN" \
  "http://138.201.254.254:5000/api/v1/history/transfer?page=1&page_size=5"

# 测试 SSE（会失败）
curl -N -H "Authorization: Bearer $TOKEN" \
  -H "Accept: text/event-stream" \
  "http://138.201.254.254:5000/api/v1/system/message"
```

---

## 📚 相关文档

- [TEST_MP.md](./TEST_MP.md) - 测试工具使用说明
- [MP API 文档](https://api.movie-pilot.org/)
- [SSE 标准](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)

---

## 📞 联系支持

如果发现获取 `resource_token` 的方法，请更新此文档并提交 PR。

**相关 Issue**: [#TODO - MP SSE 认证问题]
