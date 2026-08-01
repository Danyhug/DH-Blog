# AI 网关设计（一期：搜索网关）

## 1. 目标

在博客后端内托管 Brave Search、Tavily、Exa 三家搜索供应商，对外暴露一套统一的搜索 API。
外部 agent 只需持有网关签发的 API Key，即可发起搜索；**由博客自行决定路由到哪个供应商**，
除非请求中显式指定 `provider`。

一期只做搜索。路径与数据模型按「AI 网关」的容量预留，二期可扩展 extract / crawl / LLM 代理。

### 已确认的设计决策

| 决策点 | 结论 |
| --- | --- |
| 对外响应契约 | 裸 JSON + 正确的 HTTP 状态码，**不套用** `{code,msg,data}` |
| provider 原生透传端点 | 一期不做，留到二期 |
| `provider=auto` 选路策略 | 能力优先过滤 + 剩余配额比例平衡 |
| 上游 API Key 存储 | 明文存库，管理接口脱敏返回 |

---

## 2. 代码分层

沿用现有的 `platform`（对外集成，无业务逻辑）+ `modules`（垂直业务模块）分层，
`internal/platform/ai` 是既有先例。

```
internal/platform/search/
  provider.go      Provider 接口、统一 Request/Result 结构、错误类型
  brave.go         Brave Web Search 适配
  tavily.go        Tavily Search 适配
  exa.go           Exa Search 适配
  metadata.go      各家的官网 / 文档 / 控制台 / logo / 计费口径（静态元信息）
  ratelimit.go     出站令牌桶（每 provider 一个，进程内单例）
  breaker.go       熔断器

internal/modules/aigateway/
  module.go        New() + RegisterRoutes()
  model.go         4 张表的 GORM 模型
  repository.go    provider / apikey / log / usage 的持久化
  service.go       编排：鉴权 → 缓存 → 选路 → 调用 → 回退 → 记账
  policy.go        选路策略引擎
  auth.go          API Key 校验中间件
  handler.go       网关对外接口（裸 JSON 契约）
  handler_admin.go 后台管理接口（沿用 AjaxResult 契约）
```

### 接入点（改动既有文件）

1. `internal/app/registry.go`
   - `moduleRegistrations` 追加 `aigateway` 一项
   - `buildContext` 增加 `aigatewayModule` 字段与 `aigateway()` 懒构造方法
   - `SchemaModels()` 会自动带上迁移，无需额外改动
2. `internal/middleware/ip.go`
   - `getResourceType` 中让 `gateway` 走与 `heartbeat` 相同的跳过分支（见 §9）
3. `blog-front/src/router/index.ts` 与 `AdminSide.vue`
   - 后台新增独立菜单「AI 网关」（`/admin/gateway`），位于「系统配置」之上
4. `blog-front/src/api/` 新增 `gateway.ts`

---

## 3. 对外 API 契约

网关路径为 `/api/gateway/v1/*`，使用独立的 API Key 鉴权，**不经过 JWT 中间件**。
注册方式参照 share 模块，直接使用 `routes.Engine.Group()` 绕开 `AdminAPI`。

### 3.1 端点

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/gateway/v1/search` | 统一搜索主入口 |
| GET | `/api/gateway/v1/search?q=...` | 同上，便于 curl 与轻量 agent |
| GET | `/api/gateway/v1/providers` | 查询当前可用供应商与剩余配额 |

### 3.2 鉴权

```
Authorization: Bearer gw_live_xxxxxxxx...
```

亦接受 `X-API-Key: gw_live_...`。两者都缺失或无效时返回 401。

### 3.3 请求体

```json
{
  "query": "Go 1.25 新特性",
  "provider": "auto",
  "max_results": 5,
  "topic": "general",
  "freshness": "week",
  "country": "cn",
  "language": "zh",
  "include_domains": ["go.dev"],
  "exclude_domains": [],
  "include_answer": false,
  "include_raw_content": false,
  "allow_fallback": true,
  "no_cache": false
}
```

| 字段 | 类型 | 默认 | 约束 |
| --- | --- | --- | --- |
| `query` | string | — | 必填，1–400 字符 |
| `provider` | string | `auto` | `auto` / `brave` / `tavily` / `exa` |
| `max_results` | int | 5 | 1–20 |
| `topic` | string | `general` | `general` / `news` |
| `freshness` | string | 空 | `day`/`week`/`month`/`year` 或 `YYYY-MM-DDtoYYYY-MM-DD` |
| `country` | string | 空 | 2 位 ISO 码，网关内部转成各家格式 |
| `language` | string | 空 | ISO 639-1 |
| `include_domains` | []string | `[]` | ≤ 50（取各家下限的安全值） |
| `exclude_domains` | []string | `[]` | ≤ 50 |
| `include_answer` | bool | false | 触发能力硬过滤 → 仅 Tavily |
| `include_raw_content` | bool | false | 触发能力硬过滤 → Tavily / Exa |
| `allow_fallback` | bool | true | 显式指定 provider 时是否允许回退 |
| `no_cache` | bool | false | 跳过结果缓存 |

GET 形式用同名 query 参数，数组用逗号分隔（`include_domains=go.dev,pkg.go.dev`）。

### 3.4 成功响应（200）

```json
{
  "query": "Go 1.25 新特性",
  "provider": "tavily",
  "answer": null,
  "results": [
    {
      "title": "Go 1.25 Release Notes",
      "url": "https://go.dev/doc/go1.25",
      "content": "...",
      "score": 0.81,
      "published_at": "2026-06-12T00:00:00Z",
      "raw_content": null
    }
  ],
  "meta": {
    "request_id": "gw_01J8XK2P...",
    "cached": false,
    "latency_ms": 412,
    "credits": 1,
    "fallback_from": "brave",
    "results_truncated": false
  }
}
```

- `answer` 仅在 `include_answer=true` 且实际路由到 Tavily 时非空。
- `meta.cost_micro_usd` 仅按额计费的供应商（Exa）非零，详见 §13.3。
- `fallback_from` 为空表示首选供应商直接成功。
- `results_truncated` 表示上游返回条数多于 `max_results`，已由网关截断。

### 3.5 错误响应

统一结构，配合正确的 HTTP 状态码：

```json
{
  "error": {
    "type": "rate_limit_exceeded",
    "message": "key quota exhausted",
    "provider": "brave",
    "request_id": "gw_01J8XK2P..."
  }
}
```

| HTTP | `type` | 触发条件 |
| --- | --- | --- |
| 400 | `invalid_request` | 参数校验失败 |
| 401 | `invalid_api_key` | Key 缺失 / 无效 / 已吊销 / 已过期 |
| 403 | `provider_not_allowed` | Key 未被授权使用所请求的 provider |
| 404 | `provider_not_found` | 指定了未注册的 provider |
| 429 | `rate_limit_exceeded` | Key 的 QPM 或月配额耗尽 |
| 502 | `provider_error` | 上游返回错误且无可回退候选 |
| 503 | `no_provider_available` | 全部候选被熔断 / 配额耗尽 / 未启用 |
| 504 | `provider_timeout` | 上游超时且无可回退候选 |

---

## 4. Provider 适配

### 4.1 接口

```go
package search

type Provider interface {
    Name() string
    Capabilities() Capability
    Search(ctx context.Context, req Request) (Response, error)
}

type Capability struct {
    Answer          bool // 能返回 LLM 生成的答案
    RawContent      bool // 能返回正文全文
    DomainFilter    bool // 原生支持 include/exclude domains
    SearchOperators bool // 支持 site: / filetype: 等操作符
    Pagination      bool // 支持翻页
}
```

- Brave：`{Answer: false, RawContent: false, DomainFilter: false, SearchOperators: true, Pagination: true}`
- Tavily：`{Answer: true, RawContent: true, DomainFilter: true, SearchOperators: false, Pagination: false}`
- Exa：`{Answer: false, RawContent: true, DomainFilter: true, SearchOperators: false, Pagination: false}`（详见 §13）

### 4.2 上游端点与鉴权

| | Brave | Tavily |
| --- | --- | --- |
| 端点 | `GET https://api.search.brave.com/res/v1/web/search` | `POST https://api.tavily.com/search` |
| 鉴权头 | `X-Subscription-Token: <key>` | `Authorization: Bearer <key>` |
| 其他必需头 | `Accept: application/json`<br>`Accept-Encoding: gzip` | `Content-Type: application/json` |

### 4.3 请求字段映射

| 统一字段 | Brave | Tavily |
| --- | --- | --- |
| `query` | `q` | `query` |
| `max_results` | `count`（≤20） | `max_results`（≤20） |
| `topic=news` | `result_filter=news` | `topic=news` |
| `freshness=day/week/month/year` | `pd` / `pw` / `pm` / `py` | `time_range=day/week/month/year` |
| `freshness=A to B` | `freshness=AtoB` | `start_date` + `end_date` |
| `country` | 2 位码，如 `CN` | 英文全称，如 `china`（需内置映射表） |
| `language` | `search_lang` + `ui_lang` | 不支持，忽略 |
| `include_domains` | 不支持 → 注入 `site:` 到 `q` | `include_domains[]` |
| `exclude_domains` | 不支持 → 注入 `-site:` 到 `q` | `exclude_domains[]` |
| `include_answer` | 不支持（Summarizer 另计费） | `include_answer` |
| `include_raw_content` | 近似：`extra_snippets=true` | `include_raw_content` |
| 安全搜索 | `safesearch=moderate`（固定） | 企业版才有，忽略 |
| 搜索深度 | 无 | `search_depth`，默认取 provider 的 `Extra` 配置 |

Brave 的 `count` 上限 20、`offset` 上限 9（页码而非条数）。

### 4.4 响应字段映射

| 统一字段 | Brave | Tavily |
| --- | --- | --- |
| `results[].title` | `web.results[].title` | `results[].title` |
| `results[].url` | `web.results[].url` | `results[].url` |
| `results[].content` | `web.results[].description` | `results[].content` |
| `results[].raw_content` | `web.results[].extra_snippets` 拼接 | `results[].raw_content` |
| `results[].score` | 无 → 按名次生成 `1 - i/n` | `results[].score` |
| `results[].published_at` | `web.results[].page_age` | `topic=news` 下的 `published_date` |
| `answer` | 无 | `answer` |
| `meta.credits` | 恒为 1 | `usage.credits`（需 `include_usage=true`） |

### 4.5 上游错误归类

三家都要映射到统一的 `search.Error`，带 `Retryable` 标志决定是否触发回退：

| 上游状态 | 归类 | Retryable |
| --- | --- | --- |
| 401 / 403 | `provider_auth_failed` | 否（配置问题，回退无意义但仍换下一家） |
| 400 / 422 | `provider_bad_request` | 否 |
| 429 | `provider_rate_limited` | 是 |
| Tavily 432 / 433、Exa 402 | `provider_quota_exceeded` | 是（同时把该 provider 本月配额标记为耗尽） |
| 5xx | `provider_unavailable` | 是 |
| 网络超时 | `provider_timeout` | 是 |

---

## 5. 选路策略引擎（`provider=auto`）

规则链短路执行：

**1. 显式指定** — `provider != "auto"` 时直接选定。该 provider 不可用时：
`allow_fallback=true` 则降级为 auto 流程，否则直接返回错误。

**2. 能力过滤（硬约束）**
- `include_answer=true` 或 `include_raw_content=true` → 候选集仅保留 `Capability.Answer` / `.RawContent` 为真的（即 Tavily）
- `query` 含 `site:` / `filetype:` / 成对英文引号 → 优先保留 `SearchOperators` 为真的（Brave）；
  若因此候选集为空则放宽（Tavily 会把操作符当普通词，属可接受降级）

**3. 健康过滤** — 剔除熔断器处于 open 状态的候选。

**4. 配额过滤** — 剔除本月配额已耗尽的候选。

**5. 排序** — 按当前的**调度方式**给剩余候选排序，详见 §14。

**6. 执行与回退** — 调用失败且错误 `Retryable=true` 时，从剩余候选取下一个重试**一次**，
在 `meta.fallback_from` 中记录首选。非 Retryable 错误（参数错、鉴权错）不回退，直接返回。

### 熔断器

- 滑动窗口 60s；连续 5 次失败，或窗口内失败率 > 50% 且样本 ≥ 10 → **open**
- open 持续 30s → 转 **half-open**，放行 1 个探测请求
- 探测成功 → closed；失败 → 重新 open 30s

---

## 6. 数据模型

### 6.1 `ai_gateway_providers` — 上游配置

```go
type Provider struct {
    model.BaseModel `gorm:"embedded"`
    Name         string `gorm:"column:name;uniqueIndex;not null"` // brave / tavily
    DisplayName  string `gorm:"column:display_name"`
    Enabled      bool   `gorm:"column:enabled;default:false"`
    APIKey       string `gorm:"column:api_key"`                   // 明文，接口脱敏返回
    BaseURL      string `gorm:"column:base_url"`                  // 留空用内置默认，允许指向自建代理
    Priority     int    `gorm:"column:priority;default:100"`      // 越小越优先
    Weight       int    `gorm:"column:weight;default:1"`
    RPS          float64 `gorm:"column:rps;default:1"`            // 出站限速，Brave 免费版必须为 1
    MonthlyQuota int    `gorm:"column:monthly_quota;default:0"`   // 0 = 不限
    Extra        string `gorm:"column:extra"`                     // JSON，provider 私有默认参数
}
func (Provider) TableName() string { return "ai_gateway_providers" }
```

`Extra` 示例（Tavily）：`{"search_depth":"basic","chunks_per_source":3}`

### 6.2 `ai_gateway_api_keys` — 下游 agent 凭证

```go
type APIKey struct {
    model.BaseModel `gorm:"embedded"`
    Name             string     `gorm:"column:name;not null"`
    KeyPrefix        string     `gorm:"column:key_prefix;uniqueIndex;not null"` // gw_live_ + 前 8 位
    KeyHash          string     `gorm:"column:key_hash;not null"`               // sha256(完整明文)
    Enabled          bool       `gorm:"column:enabled;default:true"`
    AllowedProviders string     `gorm:"column:allowed_providers"`               // 空 = 全部；否则 "brave,tavily"
    RateLimitPerMin  int        `gorm:"column:rate_limit_per_min;default:60"`   // 0 = 不限
    MonthlyQuota     int        `gorm:"column:monthly_quota;default:0"`         // 0 = 不限
    ExpireAt         *time.Time `gorm:"column:expire_at"`
    LastUsedAt       *time.Time `gorm:"column:last_used_at"`
    Note             string     `gorm:"column:note"`
}
func (APIKey) TableName() string { return "ai_gateway_api_keys" }
```

**校验用 SHA-256，不用 bcrypt** —— 每个请求都要验，bcrypt 的成本无法接受。
明文格式 `gw_live_<32 位 base62 随机>`，**仅在创建时返回一次**，之后不可再取。
校验流程：截取前缀 → 唯一索引查行 → `subtle.ConstantTimeCompare(sha256(明文), KeyHash)`。
命中的 APIKey 行缓存进 `dhcache`，TTL 60s，吊销/改配额时主动失效。

### 6.3 `ai_gateway_request_logs` — 请求流水

```go
type RequestLog struct {
    ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
    CreatedAt    time.Time `gorm:"column:created_at;index"`
    APIKeyID     uint      `gorm:"column:api_key_id;index"` // 0 = 内部调用
    Provider     string    `gorm:"column:provider"`
    Endpoint     string    `gorm:"column:endpoint"`         // search
    Query        string    `gorm:"column:query"`            // 截断 200 字符
    ResultCount  int       `gorm:"column:result_count"`
    Cached       bool      `gorm:"column:cached"`
    FallbackFrom string    `gorm:"column:fallback_from"`
    Status       string    `gorm:"column:status"`           // ok / provider_error / rate_limited / ...
    HTTPStatus   int       `gorm:"column:http_status"`
    LatencyMS    int       `gorm:"column:latency_ms"`
    Credits      int       `gorm:"column:credits"`
    Error        string    `gorm:"column:error"`
    ClientIP     string    `gorm:"column:client_ip"`
}
func (RequestLog) TableName() string { return "ai_gateway_request_logs" }
```

写入走异步（复用 `task` 包的 dispatcher 或独立 buffered channel），不阻塞响应。
保留期默认 90 天，由定时任务清理。

### 6.4 `ai_gateway_usage` — 计数器

```go
type Usage struct {
    ID      uint   `gorm:"column:id;primaryKey;autoIncrement"`
    Subject string `gorm:"column:subject;uniqueIndex:idx_gw_usage;not null"` // provider:brave / key:12
    Period  string `gorm:"column:period;uniqueIndex:idx_gw_usage;not null"`  // 2026-07
    Count   int    `gorm:"column:count;default:0"`
    Credits int    `gorm:"column:credits;default:0"`
}
func (Usage) TableName() string { return "ai_gateway_usage" }
```

存在的意义是避免每次请求都对 `request_logs` 做 `count(*)`。
更新用 `ON CONFLICT DO UPDATE count = count + ?` 原子自增；读取路径带 `dhcache` 缓存（TTL 30s）。

---

## 7. 缓存与限流

### 结果缓存

- 复用 `internal/dhcache`
- key：`gw:search:` + `sha256(provider + 归一化 query + 影响结果的参数)`
  - 归一化 = trim + 转小写 + 折叠连续空白
  - 参与 key 的参数：`max_results`、`topic`、`freshness`、`country`、`language`、
    `include_domains`、`exclude_domains`、`include_answer`、`include_raw_content`
- 默认 TTL 15 分钟，后台可配（0 = 关闭）
- `no_cache=true` 时跳过读，但**仍然写**
- 缓存命中不消耗上游配额，但仍记一条 `cached=true` 的流水

> dhcache 是纯内存实现，进程重启即失效。一期可接受；若后续量级上来再考虑落盘或换 Redis。

### 入站限流（每 API Key）

- QPM：滑动窗口计数器，存 `dhcache`，key `gw:rl:<keyID>:<分钟>`
- 月配额：查 `ai_gateway_usage` 的 `key:<id>` 行

### 出站限流（每 Provider）

令牌桶，按 `Provider.RPS` 配置，**进程内全局单例**。

> ⚠️ Brave 免费版为 1 req/s、2000 次/月。请求必须在桶上**排队等待**（带 2s 上限的超时），
> 而不是直接失败——否则多个 agent 并发一来就撞 429。排队超时的候选在选路第 4 步被剔除。

---

## 8. 后台管理

管理接口挂在 `/api/admin/gateway/*`，走既有 JWT，**沿用 `{code,msg,data}` 契约**
（与前端 axios 拦截器保持一致）。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/admin/gateway/providers` | 列出供应商，`api_key` 脱敏为 `tvly-****a3f9` |
| PUT | `/api/admin/gateway/providers/:name` | 更新配置；`api_key` 传空串表示不修改 |
| POST | `/api/admin/gateway/providers/:name/test` | 连通性测试，发一次固定 query |
| GET | `/api/admin/gateway/keys` | 列出 API Key（永不返回明文） |
| POST | `/api/admin/gateway/keys` | 创建，**响应中一次性返回明文** |
| PUT | `/api/admin/gateway/keys/:id` | 改名 / 启停 / 调配额 |
| DELETE | `/api/admin/gateway/keys/:id` | 吊销 |
| GET | `/api/admin/gateway/logs` | 分页流水，支持按 provider / status / 时间筛选 |
| GET | `/api/admin/gateway/stats` | 看板聚合数据 |

后台侧是独立页面 `blog-front/src/views/backend/GatewayView.vue`（菜单「AI 网关」，路由 `/admin/gateway`），
按标签页拆开，各标签页组件放在 `blog-front/src/components/backend/gateway/`：

1. **概览** —— 今日/近 7 天/近 30 天的调用数、成功率、缓存命中率、消耗额度与花费，各 provider 明细，本月配额进度
2. **供应商** —— key、启用开关、接口地址、优先级、权重、RPS、月配额、附加参数、连通性测试
3. **调度策略** —— 负载均衡 / 按优先级 / 模型判断（选项与文案由后端下发，避免与选路引擎脱节）
4. **接入密钥** —— 基础地址与端点速查、API Key 创建/吊销/改配额，明文用对话框展示一次并提示复制
5. **请求日志** —— 分页流水，可按 provider / status 筛选

当前标签写进地址栏 `?tab=`，刷新后仍停在同一屏。

---

## 9. 与既有代码的两个冲突点

### 9.1 `IPMiddleware` 会污染访问统计

`internal/middleware/ip.go` 对每个 `/api/*` 请求都异步写一条 `access_logs`，
并且**每条都会做一次 GeoIP 查询**（`utils.GetIPLocation`）。

网关是高频机器流量，会同时造成：博客的访问统计被机器请求淹没、对外 GeoIP 请求被放大。

**处理**：在 `getResourceType` 返回 `gateway` 时，走与 `heartbeat` 相同的跳过分支。
网关自身的流水由 `ai_gateway_request_logs` 承担，信息更完整。

### 9.2 CORS 为 `AllowOrigins: ["*"]`

`internal/router/router.go` 允许任意来源，意味着网关可被任意浏览器页面直接调用。
API Key 一旦写进前端代码就等同公开。

**处理**：不改 CORS（会影响既有前端），但在管理页与文档中明确标注
「网关 Key 仅用于服务端 / agent 侧，切勿嵌入浏览器代码」。
另可给 API Key 增加可选的 IP 白名单字段作为补充防护（二期）。

---

## 10. 测试计划

按 `AGENTS.md` 的要求，测试与实现同目录，表驱动。

| 测试文件 | 覆盖内容 |
| --- | --- |
| `platform/search/brave_test.go` | 请求参数拼装（含 `site:` 注入）、响应解析、错误码归类；用 `httptest` 打桩 |
| `platform/search/tavily_test.go` | 同上，含 432/433 配额错误 |
| `platform/search/ratelimit_test.go` | 令牌桶速率、排队超时 |
| `platform/search/breaker_test.go` | open / half-open / closed 状态迁移 |
| `modules/aigateway/policy_test.go` | 选路规则链：能力硬约束、熔断剔除、配额剔除、打分排序、回退 |
| `modules/aigateway/auth_test.go` | Key 校验、过期、吊销、provider 白名单、QPM 与月配额 |
| `modules/aigateway/module_test.go` | 路由注册、端到端（打桩 provider）、错误响应结构与状态码 |

---

## 11. 分期

### 一期（已完成）

- [x] `platform/search` 的 Provider 接口 + Brave / Tavily 适配
- [x] 出站令牌桶 + 熔断器
- [x] 4 张表与迁移
- [x] API Key 签发与校验中间件
- [x] 选路策略引擎（能力优先 + 剩余配额平衡）+ 一次回退
- [x] 结果缓存
- [x] 异步流水记账 + 用量计数器
- [x] `/api/gateway/v1/search`（POST + GET）与 `/providers`
- [x] 后台管理接口与独立的「AI 网关」页面（按标签页拆分）
- [x] `getResourceType` 跳过 gateway

实现期间对设计做的两处调整：

1. **配额读取不加缓存。** 原设计写了 `dhcache` TTL 30s，但每次成功调用都要写计数器，
   缓存会被立刻失效，等于没有缓存；而配额判断读到陈旧值反而会放行超额请求。
   现在直接查 `ai_gateway_usage`——一条走唯一索引的 `IN` 查询，成本远低于上游往返。
2. **`provider_not_found` 有独立的日志状态。** 原设计把它并入 `no_provider_available`，
   会让「agent 指错供应商名」和「网关自身没有可用上游」在看板上混为一谈。

一期还刻意保留了两处限制：

- `dhcache` 是纯内存实现，进程重启后结果缓存清空（配额与流水在库里，不受影响）。
- ~~请求日志的 90 天清理逻辑已实现但尚未接定时任务~~ —— 二期已接上，见下。

### 二期（原生透传部分已完成）

- [x] provider 原生格式透传端点（`/tavily/search`、`/brave/web/search`）
- [x] 请求日志的定时清理（模块自持 ticker，与 share 的 `tokenManager` 同一写法）
- [ ] Tavily extract / crawl，Brave news / image
- [ ] 以 MCP Server 形式暴露，供支持 MCP 的 agent 直连
- [ ] 自描述的 OpenAI tool schema 端点
- [ ] API Key 的 IP 白名单
- [ ] 流水落盘归档与更长周期的统计

---

## 12. 原生透传（二期已实现）

### 12.1 端点

| 方法 | 路径 | 转发到 |
| --- | --- | --- |
| POST | `/api/gateway/v1/tavily/search` | `POST {tavilyBase}/search` |
| GET | `/api/gateway/v1/brave/web/search` | `GET {braveBase}/web/search` |

请求体与响应体都是上游的原生格式，现有 SDK 只需把 base URL 指向博客即可接入。
响应会附加两个信息头：`X-Gateway-Provider`、命中缓存时的 `X-Gateway-Cached: 1`。

### 12.2 与统一接口的关键差异

**没有选路，也没有回退。** 调用方选择这个端点，正是因为它要那家供应商的 schema；
用另一家的响应结构去顶替，只会让对面的 SDK 解析失败。所以透传路径上：

- 供应商未启用 → 404，不会改走另一家
- 上游返回 4xx/5xx → **原样返回上游的状态码与错误体**，交给 SDK 自己解析，不包装成网关错误
- 熔断打开 / 限速排队超时 / 该供应商配额耗尽 → 503

鉴权、每分钟限速、月配额、出站令牌桶、熔断、用量记账、请求流水**全部照常生效**。
流水的 `endpoint` 列记为 `tavily/search` / `brave/web/search`，以便和统一接口区分。

### 12.3 两处安全约束

1. **上游路径不接受调用方输入。** 每个网关路由硬编码对应一个上游路径。
   若把路径透传出去，网关就变成了指向供应商域名的开放代理。
2. **参数与凭证过滤。**
   - Brave：query 参数走**白名单**（`q`/`count`/`offset`/`freshness`/`goggles` 等），
     白名单外的一律丢弃，调用方无法夹带网关未记账的参数或试图覆盖凭证。
   - Tavily：请求体中的 `api_key` / `apiKey` 字段会被剥离，避免把调用方自己的密钥转发出去。
   - 请求体上限 64 KB，上游响应读取上限 8 MB。

### 12.4 缓存

与统一接口共用 `dhcache`，键为 `gw:pass:` + sha256(供应商 + 方法 + 路径 + query + 请求体)。
Tavily 的请求体在入口处会被反序列化再重新序列化一次——顺带剥离凭证，同时让
`{"query":"go","max_results":5}` 和 `{"max_results":5,"query":"go"}` 落到同一个缓存条目。
只有 2xx 响应进缓存，且响应体超过 512 KB 时跳过缓存以免撑爆内存。


---

## 13. Exa 适配（三期）

### 13.1 契约

| | Exa |
| --- | --- |
| 端点 | `POST https://api.exa.ai/search` |
| 鉴权头 | `x-api-key: <key>` |
| 原生透传 | `POST /api/gateway/v1/exa/search` |

能力：`{Answer: false, RawContent: true, DomainFilter: true, SearchOperators: false, Pagination: false}`

Exa 是**语义检索**而非关键词检索，这决定了两条路由规则：

- 它不认 `site:` / `filetype:` 等操作符（官方文档明确要求改用 `includeDomains`），
  所以带操作符的查询会被软性偏向 Brave。
- 它的普通搜索**不产出答案**（只有 deep 系列模式才综合，属于另一个延迟与成本量级），
  因此 `include_answer=true` 的请求会被能力硬过滤排除在外。

### 13.2 字段映射

| 统一字段 | Exa |
| --- | --- |
| `query` | `query` |
| `max_results` | `numResults`（≤100） |
| `topic=news` | `category: "news"` |
| `freshness=day/week/...` | `startPublishedDate = now - 跨度` |
| `freshness=A to B` | `startPublishedDate` + `endPublishedDate`（含当天 23:59:59.999） |
| `include_domains` | `includeDomains[]` |
| `exclude_domains` | `excludeDomains[]` |
| `include_raw_content` | `contents.text = true` |
| `country` / `language` | ✗，丢弃 |

响应侧：`results[].highlights` 拼接成统一的 `content`（无 highlights 时依次回落到
`summary`、截断后的 `text`）；Exa 不返回相关性分数，所以 `score` 按名次生成。

**Exa 始终请求 `contents.highlights`**。Exa 不带 `contents` 时只返回标题和链接，
没有任何摘要，这样的结果对 agent 毫无用处。

### 13.3 计费口径不同带来的改动

三家的计费单位并不可比：

| 供应商 | 计费方式 |
| --- | --- |
| Brave | 按请求，每次 1 次额度 |
| Tavily | 按 credit，basic/fast 1、advanced 2 |
| Exa | **按美元**，单次费用随搜索类型浮动 |

如果把美元金额塞进 `credits` 列，看板上的"消耗额度"就会变成把次数和金额相加的无意义数字。
因此新增 `cost_micro_usd` 列（微美元，1e-6 USD）：

- `credits` 仍是"消耗了几个计费单位"，三家都按次/按 credit 计，可以相加
- `cost_micro_usd` 只有按额计费的供应商非零，单独展示
- `ai_gateway_request_logs` 与 `ai_gateway_usage` 都有这一列，看板可看单次花费与本月累计

微美元而非毫美元：Exa 最便宜的模式单次约 $0.0025，毫美元精度会把它四舍五入没了。

### 13.4 供应商展示元信息

后台需要展示每家的 logo 与入口链接。这些是**上游的固定属性而非用户配置**，
所以放在 `internal/platform/search/metadata.go` 的代码常量里，而不是数据库：

```go
type Metadata struct {
    Name, DisplayName, HomeURL, DocsURL, ConsoleURL, LogoURL, Billing string
}
```

- `ConsoleURL` 直接指向各家的密钥页面，省去运维去翻文档
- `Billing` 是一句话的计费口径说明，避免看板数字被误读
- logo 用各家官网自己的 favicon；前端加载失败时降级为首字母色块，
  色相由名称哈希派生，保证同一供应商每次渲染颜色一致

`GET /api/gateway/v1/providers` 也带上 `home_url`/`docs_url`/`logo_url`，
让 agent 侧也能自描述地展示来源。

### 13.5 一个配置注意事项

选路打分里"剩余配额比例"占 70%，而 `MonthlyQuota = 0` 表示不限、比例恒为 1.0。
Exa 的种子配置默认不限额，于是在 `auto` 模式下它会稳定压过设了配额的 Tavily。
若希望两家均摊，给 Exa 也设一个月请求上限即可。


---

## 14. 调度方式（四期）

一期把「能力优先 + 剩余配额平衡」写死在选路引擎里。实际用起来会发现，
不同场景想要的顺序并不一样：有时希望几家免费额度均匀耗完，有时希望固定用主、
坏了才切备。所以把排序策略提取成后台可切换的配置。

### 14.1 边界：策略只管顺序，不管能不能用

前四步过滤——**Key 白名单、能力、健康、配额**——是正确性约束，任何策略下都照常执行：

- 要 `include_answer` 却路由到不会生成答案的供应商，返回的是一个空字段，不是"另一种排序"
- 熔断打开或配额耗尽的供应商，无论优先级多高都不能用

策略只决定**通过过滤后的候选按什么顺序尝试**。这条边界让新增策略永远不会引入正确性问题。

### 14.2 三种方式

| 值 | 名称 | 排序依据 | 状态 |
| --- | --- | --- | --- |
| `balanced` | 负载均衡 | `剩余配额比例 × 0.7 + 归一化权重 × 0.3`，**不看优先级** | 已实现（默认） |
| `priority` | 按优先级 | 优先级升序 → 权重降序 → 名称，**配额只用于过滤** | 已实现 |
| `model` | 模型判断 | 由小模型判断问题适合交给哪家 | **未接入**，回落到 `balanced` |

**为什么 `balanced` 不再把优先级当硬分层。** 一期的实现里，优先级是硬分层、
打分只在同层内生效。两种策略并存后这会造成语义重叠：设了优先级，"负载均衡"就被
固定顺序架空了。现在两个字段各归其主——`balanced` 只用权重和配额，`priority` 只用
优先级和权重。想表达主备就切到 `priority`，这也正是它存在的意义。

**`model` 的处理方式。** 尚未接入时既不能假装生效，也不该让选项消失：

- 后端如实持久化该选择，排序回落到 `balanced`，并打一条 warning 日志
- `GET /api/admin/gateway/settings` 在每个选项上返回 `implemented` 标记
- 后台对未接入的方式打「开发中」标签，选中后额外显示一条提示，
  明说"当前请求实际仍按负载均衡调度"

选项文案由后端下发而非前端硬编码，避免文案与引擎实际行为随时间脱节。

### 14.3 存储与接口

新增 `ai_gateway_settings` 表（`setting_key` / `setting_value`），沿用 system 模块的
键值对形态，由 aigateway 模块自己拥有，不与 system 耦合。首次启动补齐默认值 `balanced`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/admin/gateway/settings` | 当前调度方式 + 全部可选项及其文案、实现状态 |
| PUT | `/api/admin/gateway/settings` | 切换调度方式，非法值返回 400 |

切换**立即生效**，不需要重启：策略缓存在 Service 里，写库成功后同步更新。
供应商配置变更触发的 `Reload` 会重新读取该设置，不会把它重置回默认值。
