# workflow（蒸馏自 `docs/项目管理.md`）

> 提交、PR、CI、版本、部署、回滚的执行视图。

## 分支

| 用途 | 命名 | 备注 |
|------|------|------|
| 主干 | `main` | 受保护，仅 PR 合并；禁止 force-push |
| 发布候选 | `release/<MAJOR.MINOR.x>` | 仅 cut 时短期存在；只接 cherry-pick fix |
| 特性 | `feat/<域>-<动词>-<细节>` | 例：`feat/interview-list-pagination` |
| 修复 | `fix/<域>-<现象>` | 例：`fix/interview-evaluate-context-canceled` |
| 重构 | `refactor/<范围>-<目的>` | 例：`refactor/wire-resume-module-set` |
| 文档 | `docs/<topic>` | |
| 杂活 | `chore/<topic>` | |
| 实验 | `spike/<topic>` | **不允许**合主干 |

主干约束：

- `main` 始终可发布：`go build ./...` / `go vet ./...` / wire 重生成产物 / 文档计数一致
- 分支生命周期 ≤ 7 天；超过强制拆分或与 `main` rebase

## Commit（Conventional Commits）

格式：

```
<type>(<scope>): <短描述，<= 60 chars>

<可选 body：动机 / 实现 / 影响>

<可选 footer：BREAKING CHANGE / 关联 Issue>
```

type：

| type | 用途 |
|------|------|
| `feat` | 新功能（含新端点 / 新服务） |
| `fix` | 缺陷修复 |
| `refactor` | 不改外部行为的内部重构 |
| `perf` | 性能改进 |
| `chore` | 构建/依赖/Docker/CI |
| `docs` | 仅文档变化 |
| `test` | 新增/修改测试 |
| `wire` | 仅 wire / DI 装配（含 `wire_gen.go` 重生成） |

scope：`interview` / `resume` / `kb` / `rag` / `infra` / `pdf` / `redis` / `pg` / `wire` / `cmd` / `docs` / `config` / `mw` / `binding`

示例：

```
feat(interview): 新增 GET /api/interview/sessions 分页列表

- 控制器 ListInterviewSessionsService.List 走 NormalizeListPaging
- Mapper.ListInterviewSessionsPage 按 created_at desc 返回 InterviewListItem 与 total
- 前端 history.ts 已对齐，文档同步日期 2026-04-24

Refs: #123
```

```
fix(interview): 修正 SubmitAnswer 在最后一题时未入队评估

最后一题写完未触发 evaluate_status=PENDING 与 enqueue.EnqueueInterviewEvaluate；
增加单测覆盖 hasNext=false 路径。
```

不允许：

- ❌ `WIP` / `tmp` / `aaa` 之类无意义信息
- ❌ 单条提交跨多个 type（先拆 PR）
- ❌ `wire_gen.go` 与业务变更混同一 commit（拆成 `wire(server): regenerate after interview module change`）

## PR

### 标题

与首个 commit 一致（Conventional Commits）。

### 正文模板

```markdown
## 背景 / 动机
（引用 Issue 或现象描述）

## 改动
- application/<domain>/...
- infrastructure/<...>/...
- shared/<...>/...

## 影响面
- API：是 / 否（列出新增/调整端点）
- DB：是 / 否（列出迁移）
- 配置：是 / 否（列出 ENV）

## 自检
- [ ] go build ./...
- [ ] go vet ./...
- [ ] wire ./cmd/server（如改了 wire.go）
- [ ] 端到端 happy path（贴 curl 或日志）
- [ ] 文档同步：docs/项目架构.md / docs/开发规范.md / docs/开发进度.md

## 其他
（已知风险、回滚方法、上线前后置依赖）
```

### Reviewer 责任

- 24h 内首轮，2 个工作日内合并/驳回
- 关注：层界（application 不依赖 infrastructure）、端口与实现绑定、状态机门禁、错误/日志常量
- 非阻塞建议用 `nit:`；阻塞项写「必须」并给最小修复路径

### 合并方式

- 默认 **Squash & Merge**
- `wire` 与 `feat` 拆 PR 时如必须同 PR，使用 **Rebase & Merge** 保留 wire 单独的 commit
- **禁止** Merge Commit

## 本地 / CI 自检

### 本地 pre-push

```bash
gofmt -l .
goimports -l .
go vet ./...
go build ./...
# 如改了 wire 标记的文件
( cd cmd/server && go run github.com/google/wire/cmd/wire@v0.7.0 . )
git diff --name-only -- 'cmd/server/wire_gen.go'  # 确认产物已 commit
```

### CI（建仓后落地）

| 阶段 | 检查 | 失败处理 |
|------|------|---------|
| Lint | `gofmt -l . && goimports -l .` | 阻塞 |
| Vet | `go vet ./...` | 阻塞 |
| Build | `go build ./...` | 阻塞 |
| Test | `go test ./...` + `go test -tags integration ./...` | 阻塞 |
| Wire 一致性 | 重跑 wire，diff `wire_gen.go`，非空则失败 | 阻塞 |
| Docker | `docker buildx build --platform linux/amd64 .` | 阻塞 |
| 文档同步 | grep `开发进度.md` 同步日期与 PR 日期一致 | 警告 |

> 当前若无 CI，Reviewer 必须要求作者贴本地输出。

## 版本

- 规则：[SemVer](https://semver.org/lang/zh-CN/) `MAJOR.MINOR.PATCH`
- pre-release：`-rc.N`（候选）/ `-alpha.N`（早期）

| 变更 | 升 |
|------|------|
| 不兼容 API（路径/方法/字段语义） | MAJOR |
| 新增端点 / 新增字段（向后兼容） / 新增配置带默认值 | MINOR |
| 修复、重构、性能、依赖升级（无外部行为变化） | PATCH |

发布流程：

```
1. 在 main 上 cut: git switch -c release/0.2.x
2. 跑「本地 pre-push」全部检查
3. 编写 CHANGELOG（按 type 聚合，版本号置顶，含日期）
4. PR 合 release/0.2.x → main
5. main 上打 tag: git tag -a v0.2.0 -m "release v0.2.0" && git push origin v0.2.0
6. 触发镜像构建 + 推到镜像仓库
```

## 环境

| 环境 | 数据 | LLM |
|------|------|-----|
| dev（本地） | 本地 PG/Redis；可空对象存储 | 留空 `OPENAI_API_KEY` 自动 stub |
| staging | 独立库；可清；测试 bucket | 真 LLM，限额低（`AI_MODEL=gpt-4o-mini` 类） |
| prod | 独立库；备份；正式 bucket | 正式 LLM；token 上限按合同 |

最小 ENV 清单：

```
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT_SEC=120

DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=disable

REDIS_HOST=...
REDIS_PORT=6379
REDIS_PASSWORD=...
REDIS_DB=0

STORAGE_PROVIDER=s3
S3_ENDPOINT=...
S3_REGION=...
S3_BUCKET=...
S3_ACCESS_KEY=...
S3_SECRET_KEY=...

OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_API_KEY=...
AI_MODEL=gpt-4o-mini
RESUME_AI_MAX_RUNES=8000
RESUME_AI_MAX_COMPLETION_TOKENS=2048
RESUME_AI_TEMPERATURE=0.2

MAX_RESUME_UPLOAD_BYTES=20971520   # 20 MiB
LOG_LEVEL=info
DEBUG=0
HTTP_ACCESS_LOG_SUPPRESS=
```

ENV 变更必须同步 `docs/项目架构.md` §10 与 `conventions.md §配置` 清单。

## 部署

- 单镜像两阶段构建（`Dockerfile`）：`golang:1.24-alpine` → `alpine:3.21`
- 运行时：`ca-certificates`、`tzdata`、`poppler-utils`、`font-noto-cjk`
- 非 root：`appuser`（uid 10001）
- 暴露 `EXPOSE 8080`；`ENTRYPOINT ["/bin/server"]`

强制：

- 多副本部署时确认 Redis Stream 上下游一致；消费者已通过 PID 区分 consumer name
- 优雅停机：SIGTERM → `httpServer.Shutdown(15s)` + 消费者 `ctx.Done()`
- secret 走 `Secret` + `ConfigMap`；`.env*` 必须 `gitignore`，禁止入库
- Liveness：`GET /health`（站点级，永远 200）

## 监控与排障

| 维度 | 数据来源 | 关键事件键 |
|------|---------|-----------|
| HTTP 访问 | `middleware.RequestLogger` | `method` / `path` / `status` / `duration` |
| 启动 | `cmd/server/main.go` + `config.LogStartup` | `MsgServerListening`、`MsgServerStopped` |
| 简历分析消费者 | `redisstream.StartResumeAnalyzeConsumer` | `MsgResumeAIConsumerEnabled/Disabled` |
| 面试评估消费者 | `redisstream.StartInterviewEvaluateConsumer` | `MsgInterviewEvaluateConsumerStarted/Stopped` |

排障 checklist：

1. 启动日志看 `MsgPostgresStartFailed/MsgRedisStartFailed/MsgOpenAIStartFailed` 是否出现
2. 消费者日志：`MsgInterviewEvaluateXReadGroup` 频繁 → Redis 连接；`MsgInterviewEvaluateLLMFailed` → OpenAI 配额/超时
3. 错误码：503 → 注入项缺失；409 → SETNX 锁冲突（前端 90s 内重复创建）；502 → LLM 上游错

## Issue 管理

Label：

- 类型：`bug` / `enhancement` / `refactor` / `chore`
- 域：`area:interview` / `area:resume` / `area:kb` / `area:rag` / `area:infra`
- 优先级：`priority:p0` / `p1` / `p2`
- 其他：`good first issue` / `blocked` / `breaking`

PR 关联：`Refs: #123` / `Closes: #123`。

## 文档治理

- 任何路由 / 端口 / 状态机 / 配置变更 → **同 PR** 改文档
- `docs/开发进度.md` 第 6 行「文档同步日期」由 PR 作者改成本次合并日
- 冲突优先级：**代码 > `docs/开发进度.md` 表 > `docs/项目架构.md` > 主项目 Java 文档**（与主项目冲突先以本仓库为准、再补一句"与主项目差异说明"）

## 安全合规

- 任何密钥（API Key、DB password、对象存储 secret）禁止入库
- 偶发泄漏：立即旋转 + `git revert` + 强制重签 commit
- LLM 输出落库前需打分/过滤；不直接序列化进 `Result.data`
- 上传文件大小受 `MAX_RESUME_UPLOAD_BYTES` 控制，**禁止**单测/灰度临时调高（必须走配置）
- 删除接口默认硬删；如改软删需 PR 中明确并加列

## 回滚预案

| 故障 | 处置 |
|------|------|
| 镜像启动失败 / 配置错误 | 回滚到上一版镜像 tag；同时禁用最新 ConfigMap |
| 面试评估消费者持续 FAILED | 关停消费者副本 + 在 Redis 把对应 Stream 的待消费消息清理或转储；不删 `interview_sessions` |
| 简历上传写库成功但 LLM 长时间未回 | 维持现状，前端展示「分析中」；必要时手动 reanalyze |
| DB 迁移导致字段缺失 | 回滚 SQL（必须有 down 脚本）；应用回滚到上一版本 |
| OpenAI 限额/欠费 | 切 `OPENAI_BASE_URL` 到备用兼容 endpoint，或临时无 API_KEY 自动 stub |

回滚后必须开 RCA Issue（`area:infra` + `priority:p0`），48h 内输出根因与防复发措施。

## 升级周期

- Go 主版本：每年随官方支持窗口升级（保留 N、N-1）
- 关键依赖（`chi` / `gorm` / `go-redis` / `zap` / `openai-go` / `wire`）：每季度评估；PATCH 直接升、MINOR 评估、MAJOR 单 PR 单升
- 自动化：可启 Renovate/Dependabot，但**不**自动合并 wire / gorm / redis 这类核心库
