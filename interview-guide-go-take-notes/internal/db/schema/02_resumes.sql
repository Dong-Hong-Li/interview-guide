-- 与 internal/db/model（Resume）、internal/db/mapper（CRUD）及 example ResumeEntity 对齐。
-- 库名由 compose 环境变量 POSTGRES_DB=interview-guide 创建，本脚本在该库上下文中执行。

CREATE TABLE IF NOT EXISTS public.resumes (
	id BIGSERIAL PRIMARY KEY,
	file_hash VARCHAR(64) NOT NULL,
	original_filename TEXT NOT NULL,
	file_size BIGINT,
	content_type TEXT,
	storage_key VARCHAR(500),
	storage_url VARCHAR(1000),
	resume_text TEXT,
	uploaded_at TIMESTAMP NOT NULL DEFAULT NOW(),
	last_accessed_at TIMESTAMP,
	access_count INTEGER NOT NULL DEFAULT 1,
	analyze_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
	analyze_error VARCHAR(500),
	interviewer_role VARCHAR(32) NOT NULL DEFAULT 'FRONTEND'
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_resume_hash ON public.resumes (file_hash);
