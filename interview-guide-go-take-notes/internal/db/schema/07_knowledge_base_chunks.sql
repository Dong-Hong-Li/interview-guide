-- 知识库分块向量：与知识库异步向量化消费者配套。
-- embedding 维度须与 KB_EMBEDDING_DIMENSIONS、grom/knowledge_base_chunk.go 的 vector(N) 一致。
-- DashScope text-embedding-v4 支持多档维度：须在请求中带 dimensions（见 OpenAIKnowledgeEmbedder），并与本列一致（常用 1536）。
-- 若库由旧卷初始化且从未跑过本文件，宿主须执行 internal/db/apply_schema.sh。

CREATE TABLE IF NOT EXISTS public.knowledge_base_chunks (
    id BIGSERIAL PRIMARY KEY,
    knowledge_base_id BIGINT NOT NULL REFERENCES public.knowledge_bases(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_kb_chunk_per_kb UNIQUE (knowledge_base_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_kb_chunks_kb_id ON public.knowledge_base_chunks (knowledge_base_id);

-- 后续 POST /query 做 top-k 时使用；空表建索引合法（需 pgvector ≥ 0.5 以支持 hnsw）。
CREATE INDEX IF NOT EXISTS idx_kb_chunks_embedding ON public.knowledge_base_chunks
    USING hnsw (embedding vector_cosine_ops);
