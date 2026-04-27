# conventions（蒸馏自 `docs/开发规范.md`）

> 写代码前必读。背景与解释看源文档；本文件只列**强制**项与即拿即用的模板。

## 通则

- Go `1.24+`（`go.mod`），无 cgo（`Dockerfile` 中 `CGO_ENABLED=0`）。
- 提交前：`gofmt`/`gofumpt` → `goimports` → `go vet ./...` → `go build ./...` 全过。
- PR ≤ 400 行；≥ 800 行必拆。
- import 三段：标准库 / 第三方 / 本仓库；**禁止** dot import 与匿名 import。

## 包与目录

| 层级 | 包名 | 目录 |
|------|------|------|
| 控制器 | `controller` | `internal/application/<domain>/controller` |
| 应用服务 | `service` | `internal/application/<domain>/service` |
| 仓储端口 | `repository` | `internal/application/<domain>/repository` |
| 入参/出参 | `model`（含 `model/results`） | `internal/application/<domain>/model` |
| 领域 | `<domain>` | `internal/domain/<domain>` |
| 适配器 | `mapper` / `stream` / `adapter` / `pdf` 等 | `internal/infrastructure/<group>/<adapter>` |
| 站点壳 | `httpserver` / `binding` / `middleware` | `internal/interfaces/...` |
| 跨域共享 | `<topic>` | `shared/<topic>` |

文件粒度：

- 应用服务：**一个用例一文件**
- 控制器：一个域一文件 `controller.go` + 路径常量 `api.go`
- Mapper：一聚合根一文件，文件头加 `var _ repository.X = (*XMapper)(nil)`
- 单元测试：与被测同目录 `xxx_test.go`，包名 `<pkg>_test`（黑盒优先）

## HTTP 接口

**全域硬规则**：**application/service 不校验**前端/HTTP 原始入参（含 `Trim`、非空、MIME/大小、multipart 可解析性等）；**一律在 controller** 完成，再封装 `model.Validated*`。知识库上传另含正文抽取，仍在 controller 侧完成，见 `docs/开发规范.md` §3.2.1。

### 路由

```go
// internal/application/<domain>/controller/api.go
const (
    APIMountPath          = "/interview"
    PathSessionAnswers    = "/sessions/{sessionId}/answers"
    // 路径片段全部以 const 集中
)

// controller.go
func (c *XController) Register(r chi.Router) {
    r.Route(APIMountPath, func(sr chi.Router) {
        sr.Post(PathSessionAnswers, binding.Handle(c.submitAnswer))
        sr.Put(PathSessionAnswers, binding.Handle(c.saveAnswer))
    })
}
```

强制：

- 业务接口统一挂 `/api`，**没有** `/api/v1`
- 路径风格 `/<resource>/{id}/<sub-resource>`，小写连字符；动作只在 `/.../complete` 这类显式语义场景出现，且要在 `api.go` 注释说明

### 入参

```go
type SubmitAnswerReq struct {
    SessionID     string `path:"sessionId" validate:"required"`
    QuestionIndex int    `json:"questionIndex"` // 0 是合法值，不能加 required
    Answer        string `json:"answer" validate:"required"`
}
```

字段标签：

- `json:"xxx"` → application/json body
- `path:"xxx"` → chi path param（与 `/{xxx}` 同名）
- `query:"xxx"` → URL query
- `form:"xxx"` → multipart/form-data；`[]byte` 字段视为文件，结构体内同名 `Filename`/`ContentType` 自动填充
- `validate:"required"` → 反射校验，零值即 400

**整型 0 与未传无法区分** → 必填 0 字段改 `*int` / `*int64`（参考 `CreateInterviewSessionReq.ResumeID *int64`）。

### 控制器职责

控制器完成全部「HTTP/前端入参规则」（`binding.Validate`、`Trim`、非空、范围、domain 校验、multipart 正文抽取等），通过后封装为 `model.Validated*`；**service 不再重复**同类校验：

```go
return c.SubmitAnswerService.SubmitAnswer(ctx, model.ValidatedSubmitAnswer{
    SessionID:     sid,
    QuestionIndex: in.QuestionIndex,
    Answer:        answer,
})
```

service 不再重复同样的 HTTP 规则；只做**领域规则**（如题号上界依赖题库长度）。

### 出参与错误

- 成功：自动包成 `Result{200, "success", resp}`
- 错误三类：
  - `response.Err(http.StatusXxx, msg)` → HTTP 状态码 == `Result.code`
  - `response.BizErr(code, msg)` → HTTP=200，业务码（弱通道展示）
  - 其他 `error` → 500 + `errmsg.InternalServerError`
- **禁止** `panic` / `os.Exit`；启动期致命用 `log.Fatalf`（仅 `cmd/server`、`internal/config`）
- 错误**消息文案**全部走 `shared/errmsg/*.go` 常量

状态码选择：

| 场景 | 码 |
|------|---|
| 入参错 | 400 |
| 未鉴权/鉴权失败 | 401 / 403 |
| 资源不存在 | 404 / 410 |
| 业务冲突（如锁） | 409 |
| 注入项 nil（服务未就绪） | 503 |
| 上游 LLM 失败 | 502 |
| 内部异常 | 500 |

### PDF / 二进制

不走 `binding.Handle`，直接 `http.Handler`：

```go
sr.Get(PathExport, c.handleExportInterviewPDF)

func (c *X) handleExportInterviewPDF(w http.ResponseWriter, r *http.Request) {
    sid := strings.TrimSpace(chi.URLParam(r, "sessionId"))
    if c.ReportService == nil {
        response.WriteErr(w, response.Err(http.StatusServiceUnavailable, "..."))
        return
    }
    out, disp, err := c.ReportService.ExportInterviewPDF(r.Context(), sid)
    if err != nil { response.WriteErr(w, err); return }
    w.Header().Set("Content-Type", "application/pdf")
    w.Header().Set("Content-Disposition", disp)  // pdfexport.ContentDispositionRFC5987
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write(out)
}
```

## 应用服务

### 命名

`<动作><聚合>Service`：`CreateInterviewService` / `SubmitAnswerService` / `ListInterviewSessionsService`。一个 service 一个用例；超 200 行或承担两个用例必须拆。

### 构造

```go
func NewSubmitAnswerService(
    sessions repository.InterviewSessionWriter,
    cache    repository.InterviewSessionCache,
    enqueue  repository.InterviewEvaluateEnqueuer,
) *SubmitAnswerService {
    return &SubmitAnswerService{sessions: sessions, cache: cache, enqueue: enqueue}
}
```

强制：

- 名为 `New<X>Service(...) *XService`
- 参数顺序与 `wire.NewSet` 中 provider 顺序一致
- **只通过端口接口**依赖（`repository.*Port`），**禁止** import `infrastructure/*`
- 单测用 fake 实现端口，不需要起 Redis/PG

### 入参契约

- 公开方法**只接收** `model.Validated*` 或基础类型（`(ctx, page, size int)`）
- service 不做 HTTP 状态码以外的「重复格式校验」
- 领域规则校验（依赖业务上下文）必须在 service 内做

### Context

- 默认沿用 `r.Context()`
- **长跑路径**（出题、评估、PDF）：

```go
workCtx, workCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Minute)
defer workCancel()
```

- 释放锁/收尾的 cleanup 子 ctx：

```go
defer func() {
    relCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
    defer cancel()
    _ = cache.ReleaseCreatingLock(relCtx, resumeID)
}()
```

### 错误传播

- 直接 `return response.Err(...)` 让 controller 透传
- 边界翻译（gorm → 业务错）必须显式 `errors.Is(err, gorm.ErrRecordNotFound)` 分支

## 仓储端口

### 命名

| 类型 | 命名 |
|------|------|
| 写库 | `<聚合>Writer`（`InterviewSessionWriter`） |
| 只读 | `<能力>Reader`（`InterviewerRoleReader`）或 `<X>Source`（`ResumeTextSource`） |
| 缓存 | `<聚合>Cache`（`InterviewSessionCache`） |
| 入队 | `<动作>Enqueuer`（`InterviewEvaluateEnqueuer`） |
| 生成器 | `<能力>Generator`（`InterviewQuestionGenerator`） |

### 方法签名

- 第一参数恒 `ctx context.Context`
- 返回值最后一项恒 `error`
- 入参/出参类型恒为 application 层模型（`model.*` / `model/results.*`）
- **禁止**出现 `gorm/grom` 模型或 `redis.Cmd` 等基础设施类型
- 多返回用具名结构或元组（如 `([]Item, int64, error)` 列表 + total）

### 实现绑定

- Mapper 文件末尾加编译期断言：
  ```go
  var _ repository.InterviewSessionWriter = (*InterviewMapper)(nil)
  ```
- 一适配器多端口（如 `*ResumeMapper` 同时实现 `InterviewerRoleReader` + `ResumeTextSource`）→ 用 `wire.Bind` 绑定，**不要**新建包装类

## 基础设施适配器

### GORM 模型

- 放 `internal/infrastructure/postgres/grom/`（沿用 `grom` 拼写，**不要**改成 `gorm`）
- 一表一文件；列名 `column:xxx`；时间字段 `BeforeCreate` 内补默认值
- 状态枚举用 `shared/interview` 常量比较，**不**写 `"COMPLETED"` 字面量

### Mapper

- 包名 `mapper`
- 任何方法首行 nil 检查：
  ```go
  if m.gdb == nil { return errors.New("xxx: nil db") }
  ```
- 复杂查询用 `Raw(...).Scan(...)`
- 事务：`m.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })`

### Redis 适配器

- 缓存 key 用前缀常量在适配器自身包内（`session:<id>`、`interview:generate:lock:<resumeId>`）
- Stream key/group/字段名**必须**走 `shared/streamkey`
- SETNX 锁：`SetNX(ctx, key, "1", ttl)` + `Del` 释放；TTL 与业务总超时对齐，**不**用 `EXPIRE` 续命

### 消费者

- 文件 `<scenario>_consumer.go`；启动函数 `Start<Scenario>Consumer(ctx, rdb, ...deps)`
- 必有 `ensure<X>Group` 处理 `BUSYGROUP` / `NOGROUP`
- 单条处理函数 `process<Scenario>Message`，内部局部 `ack := func() { _ = rdb.XAck(...).Err() }`
- 处理失败：`MarkXxxFailed(...)` → `XAck`（**不**让消息无限循环）

## Wire（依赖注入）

强制流程：所有控制器与服务**必须**通过 `cmd/server/wire.go` ProviderSet 装配；不要手写 `New*Service(New*Mapper(db), ...)` 链。

新增 service / repository 步骤：

1. 在 `application/<domain>/repository/*.go` 写端口
2. 在 `application/<domain>/service/*.go` 写实现
3. 在 `cmd/server/wire.go` 对应 `<domain>ModuleSet` 追加 `service.New<X>Service` 与必要的 `wire.Bind`
4. 在 controller 结构体加字段
5. `cd cmd/server && go run github.com/google/wire/cmd/wire@v0.7.0 .`
6. `go build ./cmd/server` 验证

约束：

- 控制器用 `wire.Struct(new(controller.X), "*")`；其内部字段必须**全部**为已 provided 的接口/类型
- 接口同名实现冲突 → `wire.Bind(new(IFace), new(*Impl))`，**不**在 set 内重复 `provideX` 函数
- nil 兜底（如 OpenAI 不可用退回 Stub）写专门 `provideXxx`：

```go
func provideInterviewQuestionGenerator(oa *ai.OpenAIService, cfg *config.Config, lg *zap.Logger) ivrepo.InterviewQuestionGenerator {
    if oa == nil {
        return ai.NewStubInterviewQuestionGenerator()
    }
    return aiq.NewOpenAIInterviewQuestionGenerator(oa, cfg, lg)
}
```

## 错误与日志

### 错误文案

```go
// shared/errmsg/<domain>.go
const SubmitAnswerSessionNotFound = "面试会话不存在"

// service 内引用
return nil, response.Err(http.StatusNotFound, errmsg.SubmitAnswerSessionNotFound)
```

中文为前后端共用文案；纯内部日志的英文短语在 `shared/logmsg` 加 `Msg*` 常量。错误**不带敏感信息**（不拼 SQL、不拼请求体）。

### 日志

```go
lg.Info(logmsg.MsgInterviewEvaluateAIBegin,
    zap.String(logmsg.FieldSessionID, sid),
    zap.Int("questionCount", len(qs)),
)
```

强制：

- message 走 `logmsg.Msg*`，zap.Field key 走 `logmsg.Field*`
- 不输出 PII / 简历正文；调用模型时只记录 model、耗时、token 数
- 启动/关闭 INFO；预期失败 WARN；不可恢复内部错 ERROR（自动带堆栈）

### 访问日志抑制

频繁轮询 / 无业务价值的 GET → 加规则到 `config.parseHTTPAccessLogSuppress`。

## 配置

新增配置流程：

1. `internal/config/<topic>_config.go` 加结构体字段 + `validateXxxConfig()`
2. `LoadEnvironmentVariables()` 内调用并写入 `Config`
3. `LogStartup` 追加脱敏日志（`shared/logmsg.FieldXxx`）
4. 同步 `docs/项目架构.md` §10 / `docs/项目管理.md` §7 ENV 清单

强制：

- **必填**配置缺失 → `log.Fatalf`（参考 `parseMaxResumeUploadBytes`），**不**默认值掩盖
- 业务包内**禁止** `os.Getenv`

## 数据校验与边界

- 所有外部输入 `strings.TrimSpace` 后再校验
- 题号、分页：controller 检查下界（≥0/≥1），service 检查上界（依赖业务上下文）
- 分页用 `internal/domain.NormalizeListPaging(page, size)`：默认 size=20，max=100；新写分页 service **必须**调用一次
- UUID 用 `shared/uuid.NewUUID16()`（16 字符短 ID），**不**直接 `uuid.New().String()`

## 状态机

唯一真理：`shared/interview/session_status.go` + `internal/domain/interview/`。

- 域内状态判断**必须**通过门禁函数（`CompleteInterviewGate`、`Status.IsCompletedOrEvaluated()` 等）
- 不要在 service / Mapper 散落 `if status == "COMPLETED"` 字符串比较
- 新加状态：先改 `shared/interview/session_status.go` → 再改 `internal/domain/interview/session_status.go` → 最后改 service / Mapper

## SQL 与 Schema

- 列名 snake_case；时间字段 `created_at` / `updated_at` / `completed_at` 一律 `timestamptz`
- 大文本（`questions_json`、`overall_feedback`、`*_json`）用 `text`
- 外键命名 `<对端单数>_id`
- **特例**：`interview_answers.session_id` 指 `interview_sessions.id`（**主键**）而非对外 `session_id` 字符串。新加查询时延续此约定，并在 Mapper 注释中重申
- 索引：高频读字段（`session_id`、`status`、`resume_id`、`created_at`）建 BTree

## 测试

- application/service：用 fake repository 覆盖正常路径 + 主要错误分支 + 边界
- domain：纯函数 100% 覆盖（gates / status helpers）
- 命名 `Test<Func>_<场景>`；断言用原生 `t.Errorf`，不强制引入 `testify`
- 集成测试 build tag：`//go:build integration`
- LLM 在 CI **不真打**，使用 stub（`NewStubInterviewQuestionGenerator`）

## 安全

- multipart 上传走 `binding.Handle` 默认 32MB 上限；文件大小校验在 service（`MAX_RESUME_UPLOAD_BYTES`）
- 不在日志/错误体中回显完整文件名、外发 URL
- 对象存储 key：`<domain>/<id>/<uuid>.<ext>` 受控前缀，不允许用户自选
- LLM 输出**不**直接返回前端，必须落库后再透出（防 prompt 注入回流）
- 错误消息不暴露内部实现（栈、SQL）

## 反模式（一眼即拒）

- ❌ 业务包内 `os.Getenv`
- ❌ service 直接 import `internal/infrastructure/...`
- ❌ controller / service / Mapper 写 `"COMPLETED"` 字面量
- ❌ 中文文案散落在 service
- ❌ Stream key/group 字面量
- ❌ wire 重生成与业务改动塞同一 commit
- ❌ 长跑路径直接用 `r.Context()`
- ❌ 整型必填字段用 `int`（0 是合法值场景）
- ❌ Mapper 返回 GORM 模型给 application 层
- ❌ 新加端点不加访问日志抑制就开始 high-frequency 轮询

## 自检清单（提交前）

- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 无新增警告
- [ ] `gofmt -l . && goimports -l .` 无输出
- [ ] 改了 `wire.go`：跑过 `wire ./cmd/server` 并 commit `wire_gen.go`
- [ ] 改了路由：抑制规则、`docs/开发进度.md` 端点表、错误文案常量同步
- [ ] 新增端点：`api.go` 路径常量 / `request_param.go` DTO / `Validated*` / service / 端口实现 + Mapper 编译期断言均到位
- [ ] 新增配置：`internal/config` 解析 + `LogStartup` 脱敏日志 + 文档同步
- [ ] 关键日志走 `logmsg`、错误走 `errmsg`
- [ ] 自己跑过 happy path（curl / 前端联调）
- [ ] `docs/开发进度.md` 第 6 行「文档同步日期」已更新
