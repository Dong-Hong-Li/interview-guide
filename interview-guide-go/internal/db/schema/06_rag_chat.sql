-- 对齐 example RagChatSessionEntity / RagChatMessageEntity / 关联表 rag_session_knowledge_bases

CREATE TABLE IF NOT EXISTS public.rag_chat_sessions (
	id BIGSERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP,
	message_count INTEGER NOT NULL DEFAULT 0,
	is_pinned BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_rag_session_updated ON public.rag_chat_sessions (updated_at);

CREATE TABLE IF NOT EXISTS public.rag_chat_messages (
	id BIGSERIAL PRIMARY KEY,
	session_id BIGINT NOT NULL REFERENCES public.rag_chat_sessions (id) ON DELETE CASCADE,
	type VARCHAR(20) NOT NULL,
	content TEXT NOT NULL,
	message_order INTEGER NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP,
	completed BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_rag_message_session ON public.rag_chat_messages (session_id);
CREATE INDEX IF NOT EXISTS idx_rag_message_order ON public.rag_chat_messages (session_id, message_order);

CREATE TABLE IF NOT EXISTS public.rag_session_knowledge_bases (
	session_id BIGINT NOT NULL REFERENCES public.rag_chat_sessions (id) ON DELETE CASCADE,
	knowledge_base_id BIGINT NOT NULL REFERENCES public.knowledge_bases (id) ON DELETE CASCADE,
	PRIMARY KEY (session_id, knowledge_base_id)
);
