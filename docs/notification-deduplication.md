# Telegram 通知去重机制

## 问题背景

MoviePilot 的自动订阅流程会触发多条消息：

```
用户在 Jellyseerr 请求影片
  ↓
系统自动订阅到 MoviePilot
  ↓
MoviePilot 自动流程：
  1. 🔍 自动搜索 → "正在搜索资源..."
  2. 📤 发送订阅 → "已发送订阅"
  3. ⬇️ 开始下载 → "开始下载"
  4. ✅ 下载完成 → "下载完成"
  5. 📦 入库完成 → "入库成功"
  6. ✅ 订阅完成 → "已完成订阅"
```

**潜在问题：**
- Tracker 轮询可能多次检测到同一状态
- 用户可能收到重复的通知
- 造成通知轰炸

## 解决方案

### 1. 状态机保护

使用严格的状态转换规则，确保每个状态只转换一次：

```
pending → subscribed → downloading → downloaded → transferred
   ↓          ↓            ↓            ↓            ↓
 （不通知）  （通知1次）  （通知1次）  （通知1次）  （通知1次）
```

### 2. 状态转换规则

#### 订阅成功 → 开始下载
```go
// 只有从 subscribed 状态才能转换为 downloading
if record.SubscribeStatus == store.TrackingSubscribed {
    record.SubscribeStatus = store.TrackingDownloading
    // 发送"开始下载"通知（只发送一次）
}
```

#### 开始下载 → 下载完成
```go
// 只有从 downloading 状态才能转换为 downloaded
if item.Status == "completed" && record.SubscribeStatus == store.TrackingDownloading {
    record.SubscribeStatus = store.TrackingDownloaded
    // 发送"下载完成"通知（只发送一次）
}
```

#### 下载完成 → 入库完成
```go
// 只有从 downloaded 或 downloading 状态才能转换为 transferred
// 同时排除已经是 transferred 状态的记录
if (record.SubscribeStatus == store.TrackingDownloaded ||
    record.SubscribeStatus == store.TrackingDownloading) &&
    record.SubscribeStatus != store.TrackingTransferred {
    record.SubscribeStatus = store.TrackingTransferred
    // 发送"入库完成"通知（只发送一次）
}
```

### 3. 数据库事务保护

每次状态更新后立即保存到数据库：

```go
if err := t.store.UpdateTracking(record); err != nil {
    t.logger.Error("Failed to update tracking", zap.Error(err))
    continue  // 失败则跳过，不发送通知
}
// 只有更新成功才发送通知
if t.telegram != nil && t.telegram.IsEnabled() {
    t.telegram.NotifyDownloadStarted(record.Title)
}
```

### 4. 幂等性保证

即使 Tracker 多次检查同一记录，也不会重复通知：

- 第1次检查：`subscribed` → `downloading` ✅ 发送通知
- 第2次检查：`downloading` → `downloading` ❌ 不发送通知
- 第3次检查：`downloading` → `downloading` ❌ 不发送通知

## 通知流程示例

### 正常流程（新影片）

```
18:00 - Jellyseerr 请求
18:05 - 系统同步，订阅到 MP
        状态: pending → subscribed
        通知: ✅ 已自动订阅（带海报）

18:10 - MP 开始下载
        Tracker 检测到下载历史
        状态: subscribed → downloading
        通知: ⬇️ 开始下载

20:30 - MP 下载完成
        Tracker 检测到 status=completed
        状态: downloading → downloaded
        通知: ✅ 下载完成

20:35 - MP 入库完成
        Tracker 检测到入库历史
        状态: downloaded → transferred
        通知: 📦 入库成功

20:40 - Tracker 再次检查
        状态: transferred（保持不变）
        通知: ❌ 不发送（已完成）
```

### 已存在影片流程

```
18:00 - Jellyseerr 请求
18:05 - 系统同步，订阅到 MP
        MP 返回"已完成订阅"
        状态: pending → transferred（直接跳转）
        通知: ℹ️ 媒体已在库中

18:10 - Tracker 检查
        状态: transferred（保持不变）
        通知: ❌ 不发送（已完成）
```

## 关键代码位置

| 文件 | 功能 |
|------|------|
| `internal/tracker/tracker.go` | 状态机逻辑和去重保护 |
| `internal/core/sync.go` | 初始订阅状态设置 |
| `internal/store/models.go` | 状态定义 |

## 测试验证

### 验证无重复通知

1. 请求一部新影片
2. 观察 Telegram 通知数量
3. 预期：4条通知（订阅、下载开始、下载完成、入库）

```bash
# 查看事件记录
docker-compose exec syncer sqlite3 /app/data/syncer.db << EOF
SELECT
    event_type,
    datetime(created_at, 'localtime') as time
FROM download_events
WHERE source_request_id = 'REQUEST_ID'
ORDER BY created_at;
EOF
```

### 验证状态转换

```bash
# 查看状态变化日志
docker-compose logs syncer | grep -E "(subscribed|downloading|downloaded|transferred)"
```

预期日志：
```
INFO subscribed movie to MoviePilot
INFO Download started
INFO Download completed
INFO Transfer completed
```

## 常见问题

### Q: 为什么有时候跳过"下载完成"直接到"入库完成"？

A: 如果 MP 下载和入库速度很快，Tracker 第一次检查时可能已经完成入库。这种情况下：
- 状态从 `downloading` 直接跳到 `transferred`
- 只发送"入库完成"通知
- 这是正常的快速流程

### Q: 如何确认没有重复通知？

A: 检查数据库事件表：

```sql
SELECT
    event_type,
    COUNT(*) as count
FROM download_events
WHERE source_request_id = 'xxx'
GROUP BY event_type;
```

每个 `event_type` 的 count 应该都是 1。

### Q: 如果 Tracker 检查间隔很短会怎样？

A: 不会有问题。状态机保护确保：
- 即使每分钟检查一次
- 也只会在状态真正变化时发送通知
- 幂等性保证不会重复

## 总结

通过**状态机 + 数据库事务 + 条件检查**三重保护，系统能够：

✅ 避免重复通知
✅ 确保每个关键状态只通知一次
✅ 正确处理快速流程和慢速流程
✅ 支持高频轮询而不产生副作用
