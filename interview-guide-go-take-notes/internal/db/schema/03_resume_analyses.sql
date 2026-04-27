-- 对齐 example ResumeAnalysisEntity / 表 resume_analyses

CREATE TABLE IF NOT EXISTS public.resume_analyses (
	id BIGSERIAL PRIMARY KEY,
	resume_id BIGINT NOT NULL REFERENCES public.resumes (id) ON DELETE CASCADE,
	overall_score INTEGER,
	content_score INTEGER,
	structure_score INTEGER,
	skill_match_score INTEGER,
	expression_score INTEGER,
	project_score INTEGER,
	summary TEXT,
	strengths_json TEXT,
	suggestions_json TEXT,
	analyzed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_resume_analyses_resume_id ON public.resume_analyses (resume_id);
