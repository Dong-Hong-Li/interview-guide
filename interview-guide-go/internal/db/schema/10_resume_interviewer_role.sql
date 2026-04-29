-- 已有库升级：为简历增加「面试官角色」枚举列（与 internal/ai/promptprofile 及前端下拉对齐）。
ALTER TABLE public.resumes
	ADD COLUMN IF NOT EXISTS interviewer_role VARCHAR(32) NOT NULL DEFAULT 'FRONTEND';
