# architecture（蒸馏自 `docs/项目架构.md`）

> 只列「修改代码必须知道」的事实，详细背景看源文档。

## 1. 顶层目录

```
interview-guide-go-take-notes/
├─ cmd/server/                # main / wire / wire_gen / deps
├─ docs/                      # 项目文档（本 skill 的源头）
├─ internal/
│  ├─ application/            # 业务用例
│  │  ├─ resume/              # 简历域（已实现）
│  │  ├─ interview/           # 面试域（已实现）
│  │  ├─ knowledgebase/       # 知识库（501 占位）
│  │  └─ ragchat/             # RAG 对话（501 占位）
│  ├─ domain/                 # 纯函数门禁 / 状态常量
│  ├─ infrastructure/         # 适配器：DB/Redis/Storage/AI/PDF/File
│  ├─ interfaces/             # 站点壳：路由聚合/中间件/HTTP 绑定
│  ├─ config/                 # 启动期 ENV 解析
│  └─ logger/                 # zap 初始化
├─ shared/                    # 跨域共享：errmsg/logmsg/streamkey/uuid/interview/resume
├─ Dockerfile / go.mod / go.sum / .dockerignore
```

## 2. 分层与依赖方向

```
interfaces  → application/<domain>/controller
                ↓ inject services
              application/<domain>/service
                ↓ depends on repository ports
              application/<domain>/repository  ← interfaces 端口
                ↑ implements
              infrastructure/<group>/<adapter>
```

铁律：

- `application/*` 禁止 import `infrastructure/*`（除非 controller 装配时由 deps 显式注入，但生产代码里没有此用法）。
- `domain/*` 不依赖 application 与 infrastructure。
- `shared/*` 不依赖 `internal/*`。
- `interfaces/*` 不写业务，仅做路由聚合、全局中间件、HTTP 绑定。

## 3. 入口装配（`cmd/server/`）

| 文件 | 角色 |
|------|------|
| `main.go` | logger → `LoadEnvironmentVariables` → `StartDeps` → `RegistrationRoutes` → `http.Server`；SIGINT/SIGTERM → `Shutdown(15s)` |
| `deps.go::StartDeps` | 起 Storage/PG/Redis/OpenAI；调 wire 初始化控制器；启动 Redis Stream 消费者（前置不满足则 Fatal） |
| `wire.go` | `wireinject` build；`resumeModuleSet` / `interviewModuleSet`；`provideMaxResumeUploadBytes` / `provideInterviewSessionCache` / `provideInterviewQuestionGenerator`（OpenAI 缺失则 panic） |
| `wire_gen.go` | `wire` 生成产物，**勿手改**；改 `wire.go` 后跑 `wire ./cmd/server` |

启动失败策略：

- logger / config 致命缺失 → `log.Fatalf`
- Storage / PG / Redis / OpenAI 任一失败 → 仅 `lg.Error` 后 `return nil, cleanup`，HTTP 不起来
- Redis Stream 消费者：`StartDeps` 已成功拉起依赖后须启动；前置不满足 → **Fatal**
- `provideInterviewQuestionGenerator`：OpenAI 不可用 → **panic**（无 Stub 运行时兜底）

## 4. HTTP 壳（`internal/interfaces/controller/router.go`）

```
chi.NewRouter()
  ├─ MountGlobal: chimw.RequestID/RealIP/Recoverer + middleware.RequestLogger(zap, suppress)
  └─ Route("/api"):
       ├─ MountAPIHooks: 405 → APIMethodNotAllowed; 404 → APINotFound
       ├─ system.RootController.Register(r)        # GET /, GET /health, GET /api/meta
       └─ for c in apiControllers: c.Register(r)
```

各域控制器在 `Register` 内部 `r.Route(APIMountPath, ...)` 嵌套子路由：路径常量集中放 `controller/api.go`。

## 5. 单请求处理链路

```
HTTP → 全局 mw → /api 子路由 → InterviewController/...
     → binding.Handle[Req,Resp](fn)
        ├─ bindRequest: JSON / multipart / path / query
        ├─ Validate: 反射读 validate:"required"，零值即 400
        └─ fn(ctx, Req) → resp 或 *response.Error / *response.BizError
     → 成功: WriteJSON 200 + Result{200, "success", resp}
     → 失败: 按错误类型写 Result（HTTP 码与 body.code 对齐 / 业务级 200+code）
```

错误三类：

- `response.Err(code, msg)` → `*response.Error`，HTTP=code, body.code=code
- `response.BizErr(code, msg)` → `*response.BizError`，HTTP=200, body.code=自定义
- 其他 `error` → 500 + `errmsg.InternalServerError`

PDF/二进制端点 **不**走 binding；自管 header（参考 `handleExportInterviewPDF` / `handleExportAnalysisPDF`）。

访问日志抑制：`config.parseHTTPAccessLogSuppress` 默认抑制 `GET /api/interview/sessions/*` 与 `GET /api/resumes/*/detail`。新增高频轮询接口同步加规则。

## 6. 业务域

### 6.1 简历域 `internal/application/resume`

- 8 个端点（`controller/api.go`）：upload / interviewer-roles / list / statistics / {id}/reanalyze / {id}/detail / {id}/export / DELETE {id}
- 服务：`ResumeUploadService` / `InterviewerRolesService` / `ResumeListService` / `ResumeDeleteService` / `ResumeDetailService` / `ReanalyzeResumeService` / `ExportAnalysisPDFService`
- 端口：`ResumeWriter` / `ObjectStorage` / `AnalyzeJobPublisher` / `TextExtractor` / `ParseService`
- 上传分析流：multipart → `binding.bindMultipart` → 校验大小（`MAX_RESUME_UPLOAD_BYTES`）→ 抽取文本 → 落库 → 上传对象存储 → 入 `streamkey.StreamResumeAnalyze` → `StartResumeAnalyzeConsumer` 消费

### 6.2 面试域 `internal/application/interview`

- 12 个端点：POST/GET /sessions、unfinished/{resumeId}、{sessionId}/(question|report|details|export)、DELETE {sessionId}、POST/PUT {sessionId}/answers、{sessionId}/complete、GET {sessionId}
- 服务（一文件一用例）：`CreateInterviewService` / `UnfinishedSessionService` / `CurrentQuestionService` / `SubmitAnswerService` / `ListInterviewSessionsService` / `ReportService` / `GetInterviewDetailService` / `GetSessionService` / `CompleteSessionService` / `DeleteInterviewService` / `EvaluateProcessor`
- 端口（`repository/`）：
  - `InterviewSessionWriter` 写库 + 历史题 + 评估流水线 CAS
  - `InterviewSessionCache` Redis 缓存 + `TryAcquireCreatingLock`
  - `InterviewQuestionGenerator` 运行时仅为 OpenAI 实现（wire 注入）
  - `InterviewEvaluateEnqueuer` 入队
  - `InterviewerRoleReader` / `ResumeTextSource` 由 `*ResumeMapper` 同时实现，靠 `wire.Bind` 复用

### 6.3 知识库 / RAG（占位）

- 全部 501：`response.Err(http.StatusNotImplemented, errmsg.NotImplemented + ": <handler>")`
- 控制器、`api.go` 路径常量、`model/*.go` 入参 DTO 已就位；缺 service / repository / infrastructure
- 落地时按面试域形状即可，不要重新设计目录

## 7. 状态机

唯一真理：`shared/interview/session_status.go` + `internal/domain/interview/`。

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

提交答案语义：

- `POST .../answers`：写答案 → 推进游标；最后一题置 `COMPLETED` + `evaluate_status=PENDING` + `enqueue.EnqueueInterviewEvaluate`
- `PUT  .../answers`：写答案 → 刷缓存；不改游标、不入队、`answer` 可空（清空草稿）

## 8. 异步链路（Redis Streams）

定义全在 `shared/streamkey/streamkey.go`：

| Stream | Group | 入队点 | 消费者 |
|--------|-------|--------|--------|
| `resume:analyze:stream` | `analyze-group` | `ResumeUploadService` / `ReanalyzeResumeService` | `StartResumeAnalyzeConsumer` |
| `interview:evaluate:stream` | `evaluate-group` | `SubmitAnswerService` / `CompleteSessionService` | `StartInterviewEvaluateConsumer` |
| `interview:generate-questions:stream` | `interview:generate-questions:group` | （预留，当前同步出题） | （预留） |

消费者通用模式（必须保持）：

1. 启动期 `XGROUP CREATE MKSTREAM ... 0`，`BUSYGROUP` 视为已存在
2. 运行中遇 `NOGROUP` 自动重建
3. `XReadGroup` 单批小（`Count=4`），`Block=3s`
4. 消息处理：解析 → 状态门禁（如 `evaluate_status=PENDING`）→ `TryMarkXxxProcessing`（CAS）→ 业务 → 持久化 → `XAck`
5. 任一关键失败：`MarkXxxFailed(...)` → `XAck`（**不**让消息无限循环）
6. 消费者名带 PID：`evaluate-consumer-go-<pid>`

## 9. 持久化模型（`internal/infrastructure/postgres/grom`）

| 文件 | 表 | 关键字段 |
|------|------|----------|
| `interview_session.go` | `interview_sessions` | `session_id`(对外 UUID16)、`resume_id`、`status`、`evaluate_status`、`questions_json`(text)、`current_question_index`、`overall_score`、`*_json` |
| `interview_answer.go` | `interview_answers` | `session_id`(**外键 → interview_sessions.id 主键**，非对外 UUID)、`question_index`、`user_answer`、`score`、`feedback` |
| `resume.go` / `resume_analysis.go` | `resumes` / `resume_analyses` | 简历元数据 + LLM 分析快照 |
| `knowledge_base.go` / `rag_chat.go` | （预留） | 占位阶段未读写 |

Mapper 即 Adapter（`infrastructure/postgres/mapper`）：

- 一聚合根一文件
- `var _ repository.X = (*XMapper)(nil)` 编译期断言
- 任何方法首行 `if m.gdb == nil { return errors.New("xxx: nil db") }`
- GORM 模型只在 `mapper` / `grom` 包内出现，应用层不感知

## 10. AI 子系统（`internal/infrastructure/ai`）

```
ai/
├─ openai_service.go               客户端工厂
├─ adapter/                        端口适配（包装为 InterviewQuestionGenerator）
├─ stub_interview_questions.go     OpenAI 不就绪时的离线 stub
├─ interview_questions.go          题目生成（带历史题去重）
├─ interview_evaluator.go          整卷评估
├─ resume_grader.go                简历分析
├─ promptprofile/                  interviewerRole 解析
└─ prompts/                        提示词模板（按域分文件）
```

切换 LLM：在 `wire.go::provideInterviewQuestionGenerator` 加分支或新 ProviderSet；**不要**在业务 service 里 import `ai`。

## 11. 关键时序

### 11.1 创建面试会话

```
Client → POST /api/interview/sessions → controller
       → ValidatedCreateInterviewSession → CreateInterviewService
         ├─ FindUnfinishedSession (resumeId)               # 不强制时复用
         ├─ TryAcquireCreatingLock (Redis SETNX, 10min)    # 防并发
         ├─ GetHistoricalQuestionsByResumeID
         ├─ InterviewerRoleByResumeID                       # *ResumeMapper 复用
         ├─ GenerateQuestions (OpenAI，ctx 已 WithoutCancel + 10min)
         ├─ Cache.SaveSession
         └─ InsertInterviewSession
       ← Result{200, "success", InterviewSession}
```

### 11.2 提交答案 + 触发评估

```
POST /sessions/{id}/answers → SubmitAnswerService
  ├─ GetSessionRecordForSubmit
  ├─ 状态门禁（QUESTIONS_PENDING/FAILED → 400; COMPLETED/EVALUATED → 400）
  ├─ SaveInterviewAnswer
  ├─ 若最后一题：UpdateInterviewSessionEvaluatePending + Enqueue(stream)
  ├─ UpdateInterviewSessionProgress
  └─ Cache.SaveSession（合并新答案后）

[evaluate-consumer-go-<pid>]
XReadGroup → processInterviewEvaluateMessage
  → 双检 status=COMPLETED & evaluate_status=PENDING
  → TryMarkEvaluateProcessing (CAS)
  → 解析 questions_json + ListInterviewAnswers + 简历正文 + interviewerRole
  → InterviewEvaluator.EvaluateInterview (ctx WithTimeout 15min)
  → SaveEvaluationResult / WarmSessionCacheAfterEvaluate
  → XAck
```

## 12. 配置

`internal/config/*.go` 在 `LoadEnvironmentVariables()` 一次性解析所有 ENV：

| 子配置 | 关键 ENV |
|--------|---------|
| ServerConfig | `SERVER_HOST` / `SERVER_PORT` / `SERVER_READ_TIMEOUT_SEC` |
| DatabaseConfig | `DATABASE_URL` |
| RedisConfig | `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` / `REDIS_DB` |
| StorageConfig | S3/OSS endpoint/region/bucket/access keys |
| OpenAIConfig | `OPENAI_API_KEY` / `OPENAI_BASE_URL` / `AI_MODEL` / `RESUME_AI_*` token 上限 |
| 其他 | `MAX_RESUME_UPLOAD_BYTES`（**必填**，否则 `log.Fatalf`）、`HTTP_ACCESS_LOG_SUPPRESS`、`LOG_LEVEL` / `DEBUG` |

业务包内**禁止** `os.Getenv`。

## 13. 可观测与韧性

- HTTP 写超时 15min（适配 PDF 与同步出题）；读 header 超时 10s
- 关闭：SIGINT/SIGTERM → `httpServer.Shutdown(15s)`；`StartDeps.cleanup` 逆序关 Redis/PG
- 消费者完全独立 goroutine，停服由 `ctx.Done()` 驱动；未在 PEL 处理的消息留在 Redis（下次启动从 `>` 继续）
- 所有 INFO/ERROR 走 `shared/logmsg`；不散落字符串字面量

## 14. 已知风险（编辑时勿踩）

1. 知识库 14 + RAG 8 全部 501，控制器路径已就位，落地时直接补 service/repository/infrastructure，**不要**重新组织目录
2. 出题异步流 `StreamInterviewGenerateQuestions` 字段已在 `streamkey` 定义但当前同步出题。要切异步出题，按面试评估消费者形状新增 `interview_questions_consumer.go`
3. `GET /api/resumes/health` 当前**未**注册；与主项目部分文档不一致，由站点 `system.RootController` 的 `GET /health` 作为站点级替代
4. `internal/db/` 是历史/空目录占位；`infrastructure` 不得依赖 `internal/interfaces`；与 LLM/落库共用的面试报告/题目结构以 `application/interview/model/results` 为准
