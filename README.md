# interview-guide（monorepo）

> 一套**模拟面试 + 简历分析**的全栈应用：浏览器端的 React 控制台用于上传简历、生成面试题、查看评估报告与 RAG 对话；Go 后端负责简历解析、出题、评估、知识库检索、RAG 流式问答与 PDF 报告导出。
>
> 本仓库是 **monorepo**，包含前端、Go 后端，以及一份用于本地一键起服务的 Docker Compose 工程。

---

## 1. 仓库结构

```
.
├─ interview-guide-frontend/       # 前端：Vite + React + TypeScript（pnpm）
├─ interview-guide-go/             # 后端：Go（chi + GORM + Wire + zap）
│  ├─ cmd/server/                  #   应用入口与依赖装配
│  ├─ internal/                    #   application / domain / infrastructure / interfaces / config
│  ├─ shared/                      #   跨域共享：错误文案、日志键、Stream 键、UUID、状态枚举
│  ├─ docs/                        #   后端文档（项目架构 / 开发规范 / 项目管理 / 开发进度）
│  └─ Dockerfile
├─ compose.yaml                    # 一键起 server + postgres + redis + rustfs
├─ Dockerfile                      # 备份/演示用，实际由 compose.yaml 引用 server 子目录构建
├─ README.md                       # 你正在读的文件：monorepo 入口
├─ README.Docker.md                # Docker / Compose 详细说明（端口、卷、初始化脚本）
├─ .env / .env.docker / .env.compose   # 本地与容器环境变量（.env 不提交）
└─ .gitignore / .dockerignore
```

> 子项目内部目录约定见各自 README 与 `interview-guide-go/docs/项目架构.md`。

---

## 2. 技术栈一览

| 层 | 选型 | 备注 |
|----|------|------|
| 前端框架 | **Vite 8** + **React 18** + **TypeScript 5.6** | 包管理器 **pnpm**；UI 走 **Tailwind 4** + **lucide-react** + **framer-motion** |
| 前端数据 | **axios** + **react-router 7** + **react-markdown** + **recharts** | SSE 走原生 `fetch` 流式读取（见 `src/api/ragChat.ts`） |
| 后端框架 | **Go 1.24** + **chi v5** + **google/wire** | HTTP 壳在 `internal/interfaces`，业务在 `internal/application/<domain>` |
| 后端 ORM | **GORM v1.31** + **pgx v5** | Mapper 即 Adapter，所在层为 `internal/infrastructure/postgres/grom` |
| 日志 | **zap** | 错误文案与日志键统一在 `shared/errmsg`、`shared/logmsg` |
| 关系库 | **PostgreSQL 16** (`pgvector/pgvector:pg16`) | 知识库向量检索使用 `pgvector` |
| 缓存/队列 | **Redis 7** | 字符串缓存 + **Streams** 异步消费者 + SETNX 锁 |
| 对象存储 | **RustFS**（S3 兼容） | 简历原件等文件存储；本地默认起 19000/19001 端口 |
| LLM 客户端 | **openai-go**（OpenAI 兼容） | 出题、评估、RAG 全部走该客户端 |
| 文档/PDF | **gopdf** + **tabula** + **gosseract**（OCR） | 简历正文以 DOCX 提取为主；PDF 报告进程内同步生成 |

---

## 3. 快速开始

> 推荐 **Docker Compose 一键起**；只在调试 Go 后端时用「仅基础设施 + 本机 `go run`」。

### 3.1 一键起（推荐）

在仓库根目录（含 `compose.yaml` 与 `.env`）执行：

```bash
docker compose up --build
```

启动完成后：

- 后端 API：http://localhost:8081
- PostgreSQL：localhost:5432（`postgres` / `123456`，库 `interview-guide`）
- Redis：localhost:6379
- RustFS API / 控制台：http://localhost:19000 / http://localhost:19001
- 前端：另起 `pnpm dev` 后默认访问 http://localhost:5173（详见 §3.3）

详细操作（包括 `down`、`down -v`、构建参数、端口/卷/初始化脚本规则）见 [`README.Docker.md`](./README.Docker.md)。

### 3.2 仅起基础设施 + 本机调试 Go 后端

```bash
docker compose up -d postgres redis rustfs
cd interview-guide-go && go run ./cmd/server
```

本机 `go run` 时，应用只读 **`.env`**（连 `localhost`），**不**读 `.env.docker`。

### 3.3 启动前端

```bash
cd interview-guide-frontend
cp .env.development.example .env.development   # 首次：按需改 VITE_DEV_PROXY_TARGET
pnpm install
pnpm dev
```

- 浏览器：http://localhost:5173
- 接口走 Vite 代理：`/api/*` → `VITE_DEV_PROXY_TARGET`（默认 `http://127.0.0.1:8081`，对齐 compose 暴露端口）
- 也可直接设置 `VITE_API_BASE_URL`（与代理二选一）

更多说明见 [`interview-guide-frontend/README.md`](./interview-guide-frontend/README.md)。

---

## 4. 服务与端口

| 服务 | 镜像 / 来源 | 宿主机访问 | 容器内服务名 |
|------|-------------|------------|--------------|
| **server**（Go API） | 由 `interview-guide-go/Dockerfile` 构建 | http://localhost:**8081**（容器内 8080） | `server` |
| **postgres** | `pgvector/pgvector:pg16` | localhost:**5432**，库 `interview-guide` | `postgres` |
| **redis** | Redis 7 | localhost:**6379** | `redis` |
| **rustfs** | S3 兼容对象存储 | API http://localhost:**19000**，控制台 http://localhost:**19001** | `rustfs` |

容器内 `server` 连依赖务必使用 **服务名**（`postgres` / `redis` / `rustfs`），**不要**写 `localhost`。

---

## 5. 环境变量约定

| 文件 | 是否提交 | 谁会读 | 作用 |
|------|----------|--------|------|
| `.env` | **否**（`.gitignore`） | Compose 插值 + `server` 容器 `env_file` + 本机 `go run` | 默认连 `localhost`，含密钥；唯一事实来源 |
| `.env.docker` | 是（不含密钥） | `server` 容器 `env_file`（在 `.env` 之后加载，**覆盖**前者） | 把连接串改写为 `postgres` / `redis` / `rustfs` 服务名 |
| `.env.compose` | 是 | Compose 默认变量（如有） | 仅做 Compose 侧补充 |
| `interview-guide-frontend/.env.development` | **否** | Vite dev server | 仅供前端开发代理（`VITE_DEV_PROXY_TARGET` 等） |

铁律：
- 修改 `POSTGRES_PASSWORD` 时，**`.env` 与 `.env.docker` 中带密码的 `DATABASE_URL` 必须同步**；
- `compose.yaml` **不写死业务配置**，全部走 `${VAR}` 插值，值来自环境或 `.env`；
- 后端启动时若 `CORS_ALLOWED_ORIGINS` 未配置（含逗号分隔的源；本地常见 `http://localhost:5173`），**进程会启动失败**——见 `interview-guide-go/docs/项目架构.md` §3.1。

---

## 6. 业务能力一览

| 能力 | 入口 | 实现关键点 |
|------|------|------------|
| 简历上传与解析 | `POST /api/resumes/...` | DOCX 优先；PDF 建议先转 DOCX；OCR 兜底（`gosseract`） |
| 简历评估打分 | `POST /api/resumes/.../grade` | LLM 评估，结果落库 + 报告 |
| 面试出题（异步） | `POST /api/interviews/...` | 入库后发 **Redis Stream**，由后端消费者拉 LLM 生成 |
| 面试评估（异步） | 完成面试后回填答案触发 | `shared/interview/session_status.go` 的状态机门禁；消费者出报告 |
| 知识库管理 + 向量检索 | `/api/knowledge-bases/*` | `pgvector` 存向量；分块器在 `infrastructure/ai/adapter/knowledge_text_chunker.go` |
| RAG 对话（流式） | `POST /api/rag-chat/.../messages/stream` | **SSE** 推送；会话 CRUD 同前缀 |
| PDF 报告导出 | `GET /api/.../report.pdf` | `gopdf` 进程内同步生成 |

接口的实现状态、端点列表与对账关系见 `interview-guide-go/docs/开发进度.md`。

---

## 7. 文档地图

| 你想做什么 | 看哪份文档 |
|-----------|------------|
| 用 Docker 把环境跑起来 | [`README.Docker.md`](./README.Docker.md) |
| 配置/启动前端开发环境 | [`interview-guide-frontend/README.md`](./interview-guide-frontend/README.md) |
| 看懂 Go 后端整体架构（分层、HTTP 壳、Wire 装配、异步链路） | [`interview-guide-go/docs/项目架构.md`](./interview-guide-go/docs/项目架构.md) |
| 写 Go 代码前的规范（包/命名/HTTP 绑定/错误日志/SQL/自检清单） | [`interview-guide-go/docs/开发规范.md`](./interview-guide-go/docs/开发规范.md) |
| 分支/提交/PR/CI/部署/回滚流程 | [`interview-guide-go/docs/项目管理.md`](./interview-guide-go/docs/项目管理.md) |
| 当前已实现 / 占位 / 与主项目 Java 后端的差异对账 | [`interview-guide-go/docs/开发进度.md`](./interview-guide-go/docs/开发进度.md) |
| Go 后端文档总览 | [`interview-guide-go/docs/README.md`](./interview-guide-go/docs/README.md) |
| Docker 镜像构建失败排查与 Scout CVE 纪要 | [`interview-guide-go/docs/Docker镜像构建与安全扫描纪要.md`](./interview-guide-go/docs/Docker镜像构建与安全扫描纪要.md) |

---

## 8. 维护守则

- 改了路由 / 端口 / Compose 服务 / 环境变量 / 状态机 → **同一 PR** 内同步本 README 与 `interview-guide-go/docs/*`；
- 新增/删除/重命名文档 → 同步本 README §7「文档地图」与 `interview-guide-go/docs/README.md` 的「文档清单」；
- `interview-guide-go/docs/开发进度.md` 第 6 行的「文档同步日期」在合并日更新；
- 端口/服务名/镜像名变更 → 同步本 README §4 与 `README.Docker.md`；
---
