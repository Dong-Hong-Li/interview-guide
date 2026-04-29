# Docker 本地运行说明

在**仓库根目录**（含 `compose.yaml`、`.env` 的那一层）操作。

### 环境变量从哪里来

- **`compose.yaml` 不写死业务配置**：镜像名、端口、卷路径、`server` 的构建路径等，一律写成 **`${VAR}`**，值只来自**系统环境**或仓库根目录的 **`.env`**（Compose 会自动读取该文件做插值）。
- **应用进程环境**：`server` 服务使用 **`env_file`**，按顺序加载 **`.env`** 再加载 **`.env.docker`**（后者覆盖前者）。本机 `go run` / 调试一般只读 `.env`（连 `localhost`）；容器内需要连 `postgres` / `redis` / `rustfs` 服务名，由 **`.env.docker`** 覆盖连接串。
- 修改 **`POSTGRES_PASSWORD`** 时，请同时改 **`.env`**、**`.env.docker`** 里带密码的 **`DATABASE_URL`**，保持一致。

## 一键启动

```bash
docker compose up --build
```

- **不要**在整条命令外加反引号 `` ` ``，否则 zsh 可能做命令替换并报错。
- 根目录需有 **`.env`**（gitignore，勿提交密钥）：既用于 **`compose.yaml` 插值**（如 `RUSTFS_*`、`POSTGRES_IMAGE`、`SERVER_PUBLISH_PORT` 等），也会通过 `env_file` 进入 **`server` 容器**（因此进程环境里也会看到这类「仅给 Compose 用」的键名，一般可忽略）。

### 常用命令含义

| 命令 | 含义 |
|------|------|
| `docker compose` | 读取当前目录的 **`compose.yaml`**，管理其中定义的一组服务。 |
| `up` | 按配置**创建并启动**容器（未创建过的会创建）。 |
| `--build` | 启动前先**构建**需要 `Dockerfile` 的镜像（**`server`** 的上下文由 **`.env` 的 `SERVER_BUILD_CONTEXT`** 决定，默认 **`./interview-guide-go`**）。 |
| `-d` | **后台**运行（detached），终端不挂日志。 |

示例：

```bash
# 前台看日志（默认）
docker compose up --build

# 后台运行
docker compose up --build -d

# 停止并删除本次 compose 项目下的容器（网络等）
docker compose down

# 同时删掉命名数据卷（Postgres 等数据会清空，慎用）
docker compose down -v
```

### 只起基础设施（本机 `go run` 调试后端时）

不构建/启动 `server`，只起 Postgres、Redis、RustFS：

```bash
docker compose up -d postgres redis rustfs
```

本机跑 **`interview-guide-go`** 时，请把连接指向 **localhost** 与下方端口（见 `.env` 示例）。

---

## 服务与端口

| 服务 | 说明 | 宿主机访问 |
|------|------|------------|
| **server** | Go API（`Dockerfile` 在 **`SERVER_BUILD_CONTEXT`** 目录下） | **http://localhost:8081**（映射容器内 `8080`） |
| **postgres** | `pgvector/pgvector:pg16`，库 `interview-guide` | `localhost:5432`，用户 `postgres`，密码 `123456` |
| **redis** | Redis 7 | `localhost:6379` |
| **rustfs** | S3 兼容对象存储 | API **http://localhost:19000**，控制台 **http://localhost:19001** |

容器内 `server` 连数据库/缓存/对象存储时使用 **服务名**：`postgres`、`redis`、`rustfs`（例如 `APP_STORAGE_ENDPOINT=http://rustfs:9000`），**不要**写 `localhost`。

---

## 前端 `interview-guide-frontend` 本地联调

- 开发模式一般走 **Vite 代理**：浏览器 **http://localhost:5173**，API 为 **`/api/*`**，默认代理到 **`http://127.0.0.1:8081`**（与上面 Docker 映射一致）。
- 若后端改在本机 **8080**（例如本机 Delve/`go run`），在 **`interview-guide-frontend`** 下建 **`.env.development`**：`VITE_DEV_PROXY_TARGET=http://127.0.0.1:8080`。
- 也可设置 **`VITE_API_BASE_URL=http://127.0.0.1:8081`**（与代理二选一即可）。

---

## PostgreSQL 与初始化脚本

- 首次启动且 **`pgdata` 卷为空**时，会执行挂载的 **`POSTGRES_INIT_SCHEMA_REL_PATH`**（默认指向 **`interview-guide-go/internal/db/schema`**）。
- **应用内不负责 AutoMigrate**；改 SQL 后若要**重新跑初始化**，需先 `docker compose down -v` 再 `up`（会清空 Postgres 数据）。

---

## RustFS（S3 兼容）

- **访问密钥**：`rustfsadmin` / `rustfsadmin`（与 `RUSTFS_ACCESS_KEY` / `RUSTFS_SECRET_KEY` 一致）。
- **数据目录**：宿主机 **`/tmp/rustfs/data`** 挂载到容器 `/data`。若报权限错误，可执行：`sudo chown -R 10001:10001 /tmp/rustfs/data`（`rustfs` 进程用户 uid **10001**）。
- 首次使用需在控制台或 `mc` 等客户端**创建桶** **`interview-guide`**（与 `APP_STORAGE_BUCKET` 一致）。

---

## 简历上传说明

服务端当前对简历正文以 **DOCX 提取**为主；**PDF** 建议先用 [PDF24](https://tools.pdf24.org/zh/) 转为 DOCX 或可复制文本后再上传。

---

## 部署到云端（构建镜像）

在仓库内指定上下文与 Dockerfile，例如：

```bash
docker build -t myapp -f interview-guide-go/Dockerfile interview-guide-go
```

若构建机与运行机 CPU 架构不同（例如在 Apple Silicon 上构建、云上为 amd64）：

```bash
docker build --platform=linux/amd64 -t myapp -f interview-guide-go/Dockerfile interview-guide-go
```

推送到镜像仓库后按平台文档部署即可。

---

## 参考

- [Docker 官方 Go 语言指南](https://docs.docker.com/language/golang/)

> 文中数据库密码等仅适合本地开发；勿用于公网或未隔离环境。
