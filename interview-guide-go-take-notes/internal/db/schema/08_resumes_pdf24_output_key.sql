-- PDF24 异步产物在 RustFS/S3 中的对象键（与原始 uploads/resumes/... 并存）
ALTER TABLE public.resumes ADD COLUMN IF NOT EXISTS pdf24_output_key VARCHAR(500);
