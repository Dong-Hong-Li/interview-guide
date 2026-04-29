# `internal/db/schema` — Postgres DDL

仓库根目录 **`compose.yaml`** 可将本目录挂载到 Postgres 容器的 **`/docker-entrypoint-initdb.d`**（见环境变量 **`POSTGRES_INIT_SCHEMA_REL_PATH`**）。

## 仅在「空数据目录」全自动

`docker-entrypoint-initdb.d` **只在 Postgres 初次初始化卷时执行一次**。若在加入 `07_knowledge_base_chunks.sql` 等平台已存在旧 **`pgdata` 卷**，新 SQL **不会**自动补上表 → 出现 `relation "knowledge_base_chunks" does not exist` 等与缺表等价错误。

**修复**：对已连接的数据库由宿主机手动执行 **`../apply_schema.sh`**（需 **`DATABASE_URL`**），见上级目录 **`internal/db/README.md`**。

## DDL 文件名顺序说明

编号表示大致依赖顺序：**01** extensions → resume / interview → **05** knowledge_bases → **06** rag（引用 knowledge_bases）→ **07** knowledge_base_chunks（引用 knowledge_bases）→ **08** resumes 增补 **`pdf24_output_key`** → **10** 增补 **`interviewer_role`** 列 → **09** 数据清洗与会话默认值（依赖 **10** 已存在列）。

**勿**仅用 `ls *.sql` 排序：历史上曾有两个 `07_*.sql`，手工执行时应以 **`internal/db/apply_schema.sh`** 内列表为准。

## 向量维度

`07_knowledge_base_chunks.sql` 中 **`vector(N)`** 须与 **`KB_EMBEDDING_DIMENSIONS`**、`knowledge_embedder` 里传给网关的 **`dimensions`**（DashScope `text-embedding-v3`/`v4` 等）以及 `grom/knowledge_base_chunk.go` 的 **`vector(N)`** 一致；若不同请先改 DDL 再建表。
