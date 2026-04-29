-- 对齐 example InterviewSessionEntity / InterviewAnswerEntity

CREATE TABLE IF NOT EXISTS public.interview_sessions (
	id BIGSERIAL PRIMARY KEY,
	session_id VARCHAR(36) NOT NULL UNIQUE,
	resume_id BIGINT NOT NULL REFERENCES public.resumes (id) ON DELETE CASCADE,
	total_questions INTEGER,
	current_question_index INTEGER NOT NULL DEFAULT 0,
	status VARCHAR(20) NOT NULL DEFAULT 'CREATED',
	questions_json TEXT,
	overall_score INTEGER,
	overall_feedback TEXT,
	strengths_json TEXT,
	improvements_json TEXT,
	reference_answers_json TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	completed_at TIMESTAMP,
	evaluate_status VARCHAR(20),
	evaluate_error VARCHAR(500)
);

CREATE INDEX IF NOT EXISTS idx_interview_session_resume_created ON public.interview_sessions (resume_id, created_at);
CREATE INDEX IF NOT EXISTS idx_interview_session_resume_status_created ON public.interview_sessions (resume_id, status, created_at);

CREATE TABLE IF NOT EXISTS public.interview_answers (
	id BIGSERIAL PRIMARY KEY,
	session_id BIGINT NOT NULL REFERENCES public.interview_sessions (id) ON DELETE CASCADE,
	question_index INTEGER NOT NULL,
	question TEXT,
	category TEXT,
	user_answer TEXT,
	score INTEGER,
	feedback TEXT,
	reference_answer TEXT,
	key_points_json TEXT,
	answered_at TIMESTAMP NOT NULL DEFAULT NOW(),
	CONSTRAINT uk_interview_answer_session_question UNIQUE (session_id, question_index)
);

CREATE INDEX IF NOT EXISTS idx_interview_answer_session_question ON public.interview_answers (session_id, question_index);
