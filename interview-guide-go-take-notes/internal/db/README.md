# Postgres Schema（`internal/db`）

## 为什么会出现「表不存在」？

- **新知识库表**（例如 `knowledge_base_chunks`）DDL 位于 `schema/07_knowledge_base_chunks.sql`。
- 使用 Docker 时，挂载到 **`docker-entrypoint-initdb.d`** 的脚本 **只在 Postgres 首次创建数据目录时**执行；你本地若在加入该 DDL **之前**就已有 **`pgdata` 卷**，库内就**永远没有**这张表——向量化消费者在写向量时会报错：  
  `relation "knowledge_base_chunks" does not exist`。

## 对已有数据库补表 / 对齐结构

在项目内执行（把连接串换成你的库）：

```bash
cd interview-guide-go-take-notes/internal/db
chmod +x apply_schema.sh
export DATABASE_URL='postgres://USER:PASS@HOST:5432/DBNAME?sslmode=disable'
./apply_schema.sh
```

脚本 **`apply_schema.sh`** 按依赖顺序依次执行 **`schema/`** 下 SQL（以脚本内数组为准）；多数语句为 **`IF NOT EXISTS`**，可对同一库重复执行以便增量升级。

详情请阅 **`schema/README.md`**。

## `apply_schema.sh` 与 compose 的差别

| 场景 | 行为 |
|------|------|
| 空卷，`initdb.d` 已挂载整个 `schema/` | 按**文件名字典序**执行（需注意旧版两个 `07_*.sql` 顺序；建议仍用手动脚本校准） |
| 已有卷的库 | **必须**用宿主机 `./apply_schema.sh` 指向该库 |

## 同类遗漏检查清单（人工）

有新表 / 改列时自检：

1. `internal/db/schema/` 增量 SQL；
2. `internal/infrastructure/postgres/grom/` 模型列类型（尤其 **`vector(N)`**）与 DDL、`KB_EMBEDDING_DIMENSIONS` 一致；
3. 已有持久化库的开发者执行一次 `./apply_schema.sh`。
