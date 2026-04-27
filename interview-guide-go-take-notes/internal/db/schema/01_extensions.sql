-- 首次初始化时由 Postgres 容器执行（仓库根目录 compose.yaml 挂载 docker-entrypoint-initdb.d → 本目录）。
-- 需与 compose 中镜像一致：使用带 pgvector 的镜像（如 pgvector/pgvector:pg16）。

CREATE EXTENSION IF NOT EXISTS vector;
