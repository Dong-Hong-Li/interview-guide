-- 对齐 example KnowledgeBaseEntity / 表 knowledge_bases

CREATE TABLE IF NOT EXISTS public.knowledge_bases (
	id BIGSERIAL PRIMARY KEY,
	file_hash VARCHAR(64) NOT NULL,
	name TEXT NOT NULL,
	category VARCHAR(100),
	original_filename TEXT NOT NULL,
	file_size BIGINT,
	content_type TEXT,
	storage_key VARCHAR(500),
	storage_url VARCHAR(1000),
	uploaded_at TIMESTAMP NOT NULL DEFAULT NOW(),
	last_accessed_at TIMESTAMP,
	access_count INTEGER NOT NULL DEFAULT 1,
	question_count INTEGER NOT NULL DEFAULT 0,
	vector_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
	vector_error VARCHAR(500),
	chunk_count INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_hash ON public.knowledge_bases (file_hash);
CREATE INDEX IF NOT EXISTS idx_kb_category ON public.knowledge_bases (category);
