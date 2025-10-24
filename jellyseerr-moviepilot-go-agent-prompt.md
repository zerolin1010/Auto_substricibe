# Jellyseerr → 本地队列 → MoviePilot 自动订阅 —— Go 实现（AI 编程提示词）

> 目标：使用 **Go** 开发一个可靠的单进程服务，自动将 **Jellyseerr/Overseerr** 中“已批准的请求队列”拉取到**本地队列**，并**自动订阅**到 **MoviePilot**（触发其检索/下载流程）。本文件可**直接喂给 AI 编程代理（Agent）**作为系统提示词 / 指令集。

- Jellyseerr（项目：<https://github.com/juandjara/jellyseer>，API 文档：<https://api-docs.overseerr.dev/>）
- MoviePilot（项目：<https://github.com/jxxghp/MoviePilot>，Wiki：<https://wiki.movie-pilot.org/>，API 文档：<https://api.movie-pilot.org/>）

---

## 一、需求与接口要点（浓缩版）

- **总体流程**：Jellyseerr/Overseerr 获取“已批准的请求队列” → 落**本地队列**（SQLite/JSON）→ 逐条转为 **MoviePilot 订阅请求**。
- **认证**：
  - Jellyseerr/Overseerr：`X-Api-Key`（或登录 Cookie），以官方 OpenAPI 为准。
  - MoviePilot：支持 API Token / JWT（以官方 OpenAPI 为准）。
- **Jellyseerr/Overseerr 侧（读取队列）**  
  - 分页参数常见为 `take/skip`；可带 `filter`（如 `approved`、`pending` 等）。不同版本可能存在筛选差异，需在代码端**二次过滤**（以 `status == approved` 为准）。
  - **剧集**请求包含 **季（seasons）**，部分版本/PR 支持**按集（episodes）**；模型需兼容季/集两层。
- **MoviePilot 侧（创建订阅）**
  - 常见能力：根据 **TMDB ID** 创建电影/剧集订阅；剧集可按季（必要时按集）订阅。
  - 入口端点通常为 `POST /subscribe`（以公开 OpenAPI 为准）；必要时先 `GET /media/search` 将 TMDB → 内部 ID。
- **关键对齐**
  - **主键对齐**：以 **TMDB ID** 为唯一桥接键（电影直接用；剧集需带 seasons/episodes）。
  - **去重/幂等**：以 `overseerr_request_id` 作为幂等键，避免重复下发。
  - **分页**：循环 `skip += take` 直到返回条目 `< take`。

---

## 二、AI 编程提示词（喂给你的 Go 代码 Agent）

> **将本节整体复制**给你的 AI 编程代理（如 “AI Coding / Copilot Agents / Code Assistant”），填好方括号内容并执行。

### 系统角色（一次性全局设定）

你是一名严谨的 **Go 架构与集成工程师**。目标是实现一个**可靠的单进程服务**，将 Jellyseerr/Overseerr 的“已批准请求队列”拉取到本地并**自动订阅**到 MoviePilot。你必须：

- 以两端 **OpenAPI/Swagger** 为准，对不确定字段以**可配置**与**健壮解析**兜底；
- 严格实现**分页、重试、速率限制、日志、幂等**；
- 正确处理**电影 / 剧集（含多季、可能的按集）**映射；
- 产出**可运行**的 Go 项目（含 Makefile、README、自测），并**不要在日志中泄露密钥**。

### 运行环境与密钥（读取环境变量）

```bash
JELLY_URL=[https://qp.rochub.xyz/]
JELLY_API_KEY=[MTc2MTMwNDg4NDk3MjQ5YzVhNzM0LTY2NDktNGRkZi1iZmNhLTJjYTIzYjQ3MGMzYw==]
JELLY_FILTER=approved             # 可选：approved/pending/... 以实际版本为准
JELLY_PAGE_SIZE=50

MP_URL=[http://138.201.254.254:5000/]
MP_TOKEN=[ajdioasdia90d0asfu08fad8a0sdu]
MP_AUTH_SCHEME=bearer             # 可切换：bearer | x-api-token | query-token
MP_RATE_LIMIT_PER_SEC=3
MP_DRY_RUN=false                  # true 时仅打印将要创建的订阅
```

### 项目结构

```
.
├─ cmd/syncer/main.go           # 入口：一次性同步 + 守护定时
├─ internal/jelly/client.go     # Jellyseerr API 客户端
├─ internal/mp/client.go        # MoviePilot API 客户端
├─ internal/store/sqlite.go     # SQLite 存储（或 JSON 模式）
├─ internal/core/sync.go        # 对齐/转换/幂等/重试核心逻辑
├─ configs/config.go            # 环境配置加载与校验
├─ Makefile
├─ go.mod
└─ README.md
```

### 实现步骤（必须逐项完成）

1) **依赖与基础设施**
   - Go 1.22；`net/http` + `context`；建议 `hashicorp/go-retryablehttp` 或自写重试；`uber-go/zap` 日志；`modernc.org/sqlite`（或 `mattn/go-sqlite3`）。
   - **可选**：引入 OpenAPI 代码生成（如 `openapi-generator`），若文档不可直接拉取 JSON，则**手写最小必要模型**。

2) **Jellyseerr 客户端**
   - 认证：`X-Api-Key: ${JELLY_API_KEY}`。
   - 列表：实现 `ListRequests(ctx, filter string, take, skip int)`，直到返回数 `< take` 停止。
   - 防御性策略：即使远端 `filter` 异常，也在本地以 `status == approved` 再筛一次。
   - 数据模型至少包含：`requestId`, `mediaType`(movie/tv), `tmdbId`, `title`, `seasons[]`（兼容 `episodes[]`）, `requestedAt`, `status`。

3) **本地存储（SQLite / JSON）**
   - 表 `requests`：`id PK`, `source_request_id UNIQUE`, `media_type`, `tmdb_id`, `title`, `seasons_json`, `episodes_json`, `status`, `created_at`, `updated_at`。
   - 表 `mp_links`：`id PK`, `source_request_id UNIQUE`, `mp_subscribe_id`, `state`(created/synced/failed), `last_error`。
   - 保证**幂等**：同一 `source_request_id` 不重复推送；失败项入重试队列。

4) **MoviePilot 客户端**
   - 认证：支持 `Authorization: Bearer ${MP_TOKEN}` 或 `X-API-Token` / `?token=`（由 `MP_AUTH_SCHEME` 切换）。
   - 订阅接口：
     - **电影**：以 TMDB id 直接创建订阅。
     - **剧集**：将 `seasons[]`（必要时包含 `episodes[]`）映射为 MoviePilot 的订阅结构；若 API 仅支持季级订阅，则按配置将集级请求**聚合到季**。
   - 去重：可先查询是否存在同源订阅（或由本地幂等保证）。
   - **速率限制**与**指数退避重试**（对 429/5xx）。

5) **对齐/转换规则（核心）**
   - **电影**：`tmdbId` → 直接订阅。
   - **剧集**：
     - 优先按 Jellyseerr 的 `seasons[]` 映射；若出现“按集请求”，按 `MP_TV_EPISODE_MODE=season|episode` 决定聚合或逐集创建。
     - 注意“**特别季 S00**”可能导致“订阅永不完成”，可通过配置排除或拆分。
   - 所有下发请求携带**来源幂等键**（如 `overseerr_request_id`）以便本地与远端关联。

6) **命令行与调度**
   - `sync once`：全量同步一次（带 `--since` 可选增量）。
   - `sync daemon`：每 N 分钟拉取增量（以 `requestedAt` 或最大 `source_request_id` 为界），失败重试。
   - `--dry-run`：仅打印 MoviePilot 请求 payload，不实际下发。

7) **错误处理与重试**
   - 区分**可重试**（429/5xx/网络）与**不可重试**（4xx 业务/参数）。
   - 为每条失败记录存储 `last_error`（响应摘要，**不含密钥**），并按指数退避计划重试。

8) **日志与观测**
   - 使用结构化日志（成功/失败/重试计数、每次同步窗口、每条记录的 `source_request_id`）。
   - 可选暴露 `/metrics`（prometheus）。

9) **测试与验收**
   - 覆盖：分页、状态过滤、电影/多季/含特别季、按集聚合、幂等、429/5xx 重试、速率限制。
   - 提供 `curl` 示例（以实际 API 文档为准）：
     - Jellyseerr：`curl -H "X-Api-Key: $JELLY_API_KEY" "$JELLY_URL/api/v1/request?take=10&skip=0&filter=approved"`
     - MoviePilot：`curl -H "Authorization: Bearer $MP_TOKEN" -X POST "$MP_URL/api/v1/subscribe" -d '{...}'`

### 非功能要求（安全 / 可维护）

- 严禁在日志、panic、错误输出中打印任何**密钥/令牌**。
- 所有对外请求都带 **`context.Context`**，支持超时与取消。
- 模块化，接口清晰，便于替换存储与 HTTP 客户端实现。

---

## 三、边界与已知坑（务必处理）

- 某些版本的 Jellyseerr/Overseerr 端 `filter` 行为与 UI 不一致：需在本地**二次筛选**为 `approved`。
- 剧集的“**特别季 S00**”会影响完成条件：建议默认**排除 S00**或将其单独作为一条订阅。
- 可能存在“**按集请求**”与“**按季接口**”不匹配：提供 `MP_TV_EPISODE_MODE=season|episode` 配置实现聚合或逐集。
- 与 Emby 的联动不影响本同步器；**不要**做名称模糊匹配，始终以 **TMDB ID** 对齐。

---

## 四、交付物

- 可运行的 **Go 服务**（含 Dockerfile 与示例 compose）。
- 默认 **SQLite 本地落地**；支持 `--store json` 的轻量模式。
- 基础单测与（可选）GitHub Actions CI。
- `README.md`：环境变量说明、运行方式、常见问题（过滤异常、特别季策略、速率限制与重试策略等）。

---

## 五、你需要准备/提供给 Agent 的参数

- Jellyseerr/Overseerr：`JELLY_URL` 与 `JELLY_API_KEY`（后台设置页获取）。
- MoviePilot：`MP_URL` 与 `MP_TOKEN`（强口令；或使用登录后的 JWT）。
- 若你的 MoviePilot 版本/部署采用专用认证头或查询参数，请在 `MP_AUTH_SCHEME` 中配置。

---

## 参考链接

- Jellyseerr（Overseerr 派生版本）：<https://github.com/juandjara/jellyseer>  
- Overseerr API 文档：<https://api-docs.overseerr.dev/>  
- MoviePilot：<https://github.com/jxxghp/MoviePilot>  
- MoviePilot Wiki：<https://wiki.movie-pilot.org/>  
- MoviePilot API 文档：<https://api.movie-pilot.org/>
