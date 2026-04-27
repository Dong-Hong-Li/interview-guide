---
name: interview-guide-go-take-notes
description: Encodes architecture, coding conventions, and workflow rules for the Go backend at interview-guide-go-take-notes/. Use when editing any file under interview-guide-go-take-notes/ (cmd/server, internal/, shared/, docs/, Dockerfile, go.mod), adding/changing HTTP endpoints, services, repositories or wire providers, touching Postgres/GORM mappers, Redis Streams consumers, AI prompts, PDF export, configuration in internal/config, error/log constants in shared/errmsg or shared/logmsg, or when reviewing PRs / generating commit messages for this Go module. Trigger words include interview-guide-go-take-notes, chi 路由, wire, GORM, Redis Stream, 面试评估, 简历分析, controller/service/repository, application/<domain>, infrastructure/postgres/grom, RouteRegistrar, binding.Handle.
---

# interview-guide-go-take-notes 工程守则

> 作用域：`interview-guide-go-take-notes/` 下所有 Go 代码与文档。  
> 信息来源：`docs/项目架构.md` / `docs/开发规范.md` / `docs/项目管理.md` / `docs/开发进度.md`。本文件是它们的执行视图，**必须**遵守。

## 0. 启动前必读

- 业务接口统一挂在 `/api`，**没有** `/api/v1` 前缀。
- `internal/db/` 当前为空目录；GORM 模型实际在 `internal/infrastructure/postgres/grom/`（拼写就是 `grom`，不要改）。
- `internal/infrastructure/*` **不得** import `internal/interfaces/*`；面试报告/题目等共用结构以 `application/.../model/results` 为准。
- 状态字面量唯一真理：`shared/interview/session_status.go`；状态判断走 `internal/domain/interview/` 的门禁函数。
- 知识库 14 个 + RAG 8 个端点为 501 占位，控制器已到位，只能补 service/repository/infrastructure。
- 主项目是 Java 后端；语义对齐 Java，但**实现细节以本仓库代码为准**，禁止套 Java 的 `modules/*` / `router/router.go` 目录习惯。

## 1. 三铁律

1. **层界单向**：`application/*` 禁止 import `infrastructure/*`；`domain/*` 禁止 import 应用层与基础设施；`shared/*` 禁止 import `internal/*`；`infrastructure/*` 禁止 import `internal/interfaces/*`。具体实现一律由 `repository/*Port` 接口反转 + `cmd/server/wire.go` 装配。
2. **零字面量散落**：所有错误文案放 `shared/errmsg/*.go`；所有日志 message/key 放 `shared/logmsg/logmsg.go`；所有 Stream 名/Group 名/字段名放 `shared/streamkey/streamkey.go`；所有 ENV 解析在 `internal/config/*`，业务包内**禁止** `os.Getenv`。
3. **文档同步硬绑定**：改了路由/端口/状态机/配置 → 同 PR 改 `docs/项目架构.md` 与 `docs/开发进度.md`，并把 `开发进度.md` 第 6 行「文档同步日期」更新为合并日。

## 2. 任务路由（按编辑意图选）

| 编辑意图 | 必读 | 关键约束 |
|---------|------|---------|
| 新增/修改 HTTP 端点 | `conventions.md §HTTP` + `architecture.md §HTTP` | path 常量进 `controller/api.go`；用 `binding.Handle[Req,Resp]`；入参在 `model/request_param.go`；过校验后封 `Validated*` 传 service；错误用 `response.Err`/`BizErr` |
| 新增应用服务/用例 | `conventions.md §服务` | 一用例一文件；构造函数 `NewXService(...)`；只依赖 `repository/*Port`；长跑路径 `context.WithoutCancel(ctx) + WithTimeout` |
| 新增仓储端口/实现 | `conventions.md §端口与适配器` | 端口在 `application/<domain>/repository/*.go`；Mapper 在 `infrastructure/postgres/mapper/`；末尾加 `var _ repository.X = (*XMapper)(nil)`；GORM 模型只在 `grom/` 包内出现 |
| 改 Wire 装配 | `conventions.md §Wire` | 改 `cmd/server/wire.go` 后必须在 `cmd/server/` 下跑 `wire`；`wire_gen.go` 与业务变更**拆分** commit；接口冲突走 `wire.Bind`，nil 兜底走专门 `provideXxx` |
| 改 Redis Stream 消费者 | `architecture.md §异步链路` | 文件 `<scenario>_consumer.go`；`ensure*Group` 处理 BUSYGROUP/NOGROUP；处理失败必须 `MarkXxxFailed` 后再 `XAck`；消费者名带 PID |
| 改配置 | `conventions.md §配置` | `internal/config/<topic>_config.go` 加结构体 + `validateXxxConfig()`；`LogStartup` 加脱敏日志；必填缺失直接 `log.Fatalf` |
| 改 PDF/二进制端点 | `conventions.md §HTTP §PDF` | 不走 `binding.Handle`；自管 `Content-Type: application/pdf` + `pdfexport.ContentDispositionRFC5987(name)`；错误仍 `response.WriteErr(w, err)` |
| 改 SQL/Schema | `conventions.md §SQL` | 一表一文件 `grom/`；列名 `column:xxx`；时间字段 `timestamptz`；注意 `interview_answers.session_id` 指向 `interview_sessions.id`（**主键**），非对外 UUID |
| 提交/开 PR | `workflow.md` | Conventional Commits（type+scope）；自检清单；`wire_gen.go` 单独 commit；改文档同步「同步日期」 |

## 3. 启动装配速查（`cmd/server/`）

| 文件 | 角色 | 改动须知 |
|------|------|---------|
| `main.go` | 启动 logger → 加载 config → `StartDeps` → 路由 → `http.Server`（SIGINT/SIGTERM 优雅停机） | HTTP 写超时已设 15min，不要再缩短 |
| `deps.go` | 组合根：起 Storage/PG/Redis/OpenAI；委托 wire 构造控制器；条件启动 2 个消费者 | 新基础设施加在此处；任意失败 `return nil, cleanup` |
| `wire.go` | `wireinject` build tag；`resumeModuleSet` / `interviewModuleSet`；接口绑定与 stub 回退 | `*ResumeMapper` 同时实现 `InterviewerRoleReader` + `ResumeTextSource`，复用而非新建包装 |
| `wire_gen.go` | wire 生成产物，**勿手改** | 改完 `wire.go` 必须重新生成 |

## 4. 状态机（面试会话）

```
CREATED → QUESTIONS_PENDING → QUESTIONS_FAILED ┐
                            ↓                   │
                        IN_PROGRESS  ───────────┤
                            ↓                   │
                        COMPLETED → EVALUATED   │
                            ↑                   │
                       (PUT 草稿不变状态) ────────┘

evaluate_status: PENDING → PROCESSING → COMPLETED / FAILED
```

- `POST /sessions/{id}/answers`：推进游标；最后一题置 `COMPLETED` + `evaluate_status=PENDING` + 入队 `streamkey.StreamInterviewEvaluate`。
- `PUT  /sessions/{id}/answers`：仅落库当题、刷 Redis 缓存；不推进游标、不入队、允许 `answer` 空串清空草稿。
- 任何状态判断走 `shared/interview` 常量 + `internal/domain/interview` 门禁函数；**禁止**写 `if status == "COMPLETED"` 字面量。

## 5. 创建会话防并发（重点）

`CreateInterviewService.CreateInterview` 已实现的不变量，扩展时**保持**：

- `interviewSessionCache.TryAcquireCreatingLock(resumeID, 10min)` 走 Redis SETNX；TTL 必须与 `createInterviewWorkMax` 对齐。
- `workCtx = context.WithoutCancel(ctx)` + `WithTimeout(10min)`：避免反代 ~90s 断连导致 LLM/落库 `context canceled`。
- 释放锁的 cleanup 子 ctx **必须** `WithoutCancel(ctx)` 再 `WithTimeout(5s)`。
- OpenAI 客户端为 nil 时自动 stub（`ai.NewStubInterviewQuestionGenerator`），由 `provideInterviewQuestionGenerator` 兜底。

## 6. 错误与日志（一眼模板）

```go
// 加错误文案：先在 shared/errmsg/<domain>.go 加常量
const SubmitAnswerSessionNotFound = "面试会话不存在"

// service 内
return nil, response.Err(http.StatusNotFound, errmsg.SubmitAnswerSessionNotFound)

// 业务级（HTTP 200 + 业务码）
return nil, response.BizErr(40001, errmsg.XXX)

// 加日志：先在 shared/logmsg/logmsg.go 加 Msg* / Field* 常量
lg.Info(logmsg.MsgInterviewEvaluateAIBegin,
    zap.String(logmsg.FieldSessionID, sid),
    zap.Int("questionCount", len(qs)),
)
```

错误状态码选择：入参错 400 / 鉴权 401·403 / 不存在 404·410 / 锁冲突 409 / 注入项 nil 503 / LLM 上游错 502 / 其他 500。

## 7. 提交前自检（强制）

```bash
gofmt -l .
goimports -l .
go vet ./...
go build ./...
# 改了 wire.go 才需要：
( cd cmd/server && go run github.com/google/wire/cmd/wire@v0.7.0 . )
```

PR 描述必须有「自检」勾选清单 + 「影响面」（API/DB/配置）+ 「Doc 同步」明细。详情见 `workflow.md`。

## 8. 反模式（碰即停）

- ❌ 业务包内 `os.Getenv` —— 走 `internal/config`
- ❌ service 直接 `import "internal/infrastructure/..."` —— 走端口接口反转
- ❌ 在 controller / service / Mapper 写 `"COMPLETED"` 字面量 —— 走常量 + 门禁
- ❌ 中文文案散落在 service —— 走 `shared/errmsg`
- ❌ Stream key/group 字面量 —— 走 `shared/streamkey`
- ❌ 把 wire 重生成与业务改动塞到同一 commit
- ❌ 长跑路径直接用 `r.Context()` —— 必须 `context.WithoutCancel(ctx) + WithTimeout`
- ❌ 整型必填字段用 `int` 但 0 是合法值 —— 改 `*int`/`*int64`
- ❌ Mapper 返回 GORM 模型给 application 层 —— 转换为 `model/results.*`
- ❌ 新加端点不加访问日志抑制规则就开始 high-frequency 轮询

## 9. 索引（按需展开读）

- 详细架构（分层、模块、入口、HTTP 生命周期、异步链路、状态机、关键时序）→ [`architecture.md`](./architecture.md)
- 完整开发规范（包/HTTP/服务/端口/适配器/配置/Wire/错误日志/SQL/测试/安全）→ [`conventions.md`](./conventions.md)
- 提交流程（分支/Commit/PR/CI/版本/部署/回滚）→ [`workflow.md`](./workflow.md)
- 接口实现进度对账 → `interview-guide-go-take-notes/docs/开发进度.md`
- 工程级源文档（与本 skill 同源）→ `interview-guide-go-take-notes/docs/项目架构.md` / `开发规范.md` / `项目管理.md`

## 10. 何时必须读哪个 reference

| 触发场景 | 读哪份 |
|---------|------|
| 不熟悉项目分层 / 第一次接入 | `architecture.md` |
| 写代码前/PR review 时（命名、错误、日志、Wire 等） | `conventions.md` |
| 开 PR / 写 commit / 发版 / 回滚 | `workflow.md` |
| 需要查具体接口端点是否已实现 | `interview-guide-go-take-notes/docs/开发进度.md` |
| 改完未跑 build/vet/wire | 立即跑 §7 自检命令 |
