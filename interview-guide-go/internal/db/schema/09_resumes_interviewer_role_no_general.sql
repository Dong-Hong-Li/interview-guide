-- 移除「通用」角色：历史 GENERAL 与异常空值统一为 FRONTEND，默认改为 FRONTEND
UPDATE public.resumes
SET interviewer_role = 'FRONTEND'
WHERE interviewer_role IS NULL
   OR TRIM(interviewer_role) = ''
   OR UPPER(TRIM(interviewer_role)) = 'GENERAL';

ALTER TABLE public.resumes
	ALTER COLUMN interviewer_role SET DEFAULT 'FRONTEND';
