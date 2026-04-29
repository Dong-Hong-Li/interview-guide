-- 一次性迁移：embedding 列从 vector(1536) → vector(1024)，与阿里云 DashScope「text-embedding-v4」等 1024 维输出对齐。
-- 会先删 HNSW 索引、清空分块表（请确认可接受）。
-- Docker 示例（仓库根目录，服务名 postgres 按 docker compose ps 调整）：
--   docker compose exec -T postgres psql -U postgres -d interview-guide -v ON_ERROR_STOP=1 -f /dev/stdin \
--     < interview-guide-go/internal/db/schema/migrate_kb_chunks_embedding_to_1024.sql

BEGIN;
DROP INDEX IF EXISTS public.idx_kb_chunks_embedding;
TRUNCATE public.knowledge_base_chunks;
ALTER TABLE public.knowledge_base_chunks
  ALTER COLUMN embedding TYPE vector(1024);
CREATE INDEX IF NOT EXISTS idx_kb_chunks_embedding ON public.knowledge_base_chunks
  USING hnsw (embedding vector_cosine_ops);
COMMIT;
