package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"interview-guide-go/internal/application/knowledgebase/model"
	"interview-guide-go/internal/application/knowledgebase/model/results"
	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/logmsg"
	"interview-guide-go/shared/response"

	"go.uber.org/zap"
)

const (
	// kbQueryTopK 每个问题从 PG 拉取的候选分块条数上限（再做距离阈值过滤）。
	kbQueryTopK = 16
	// kbQueryMaxCosineDistance pgvector 余弦距离 <=> 的上限（越小越相似）；与 Java app.ai.rag.search.min-score-default≈0.28 对应约 (1-0.28)。
	kbQueryMaxCosineDistance = 0.72
	kbQueryLLMTimeout        = 4 * time.Minute

	// kbRetrievalLogTopHits 检索汇总日志中带摘要的原始 Top 命中条数（已按距离升序）。
	kbRetrievalLogTopHits = 5
	// kbRetrievalPreviewRunes 每条命中打进日志的正文前缀长度（Unicode rune）。
	kbRetrievalPreviewRunes = 72
	// kbQueryQuestionPreviewRunes 「开始检索」日志中问题前缀长度，便于与 Embedding 请求对账。
	kbQueryQuestionPreviewRunes = 48

	// kbQuerySystemPrompt 约束模型仅依据检索片段作答；当用户问题宽泛/多义/仅部分命中资料时，
	// 必须先基于实际命中的角度作答，再以「说明」段点出未覆盖方向并向用户反问具体场景，
	// 避免单方向输出后用户得不到真正想要的答案（与 Java prompts/knowledgebase-query-system 同意图，并强化澄清式追问）。
	kbQuerySystemPrompt = `你是一个严谨的助手，只根据下面给出的「参考资料」回答用户问题。

回答要求：
1. 仅依据「参考资料」作答，不要编造资料中不存在的事实；回答语言与用户问题一致（多为中文），可使用 Markdown。
2. 当用户问题表述较宽泛、存在多种含义、或与「参考资料」只有部分匹配时：
   - 先基于资料中**确实命中**的角度，给出尽可能完整、有条理的回答（按主题分小节，必要时举例）；
   - 在正文之后追加「说明」小节，**明确指出**资料未覆盖的子方向（例如：不同语义、不同使用场景、不同实现层面、客户端 vs 服务端、构建期 vs 运行期等）；
   - 然后**主动反问用户**，请其聚焦到具体的子问题/场景，便于后续给出更精准的答案。
3. 若「参考资料」完全无法支撑作答，请直接说明无法从资料中推断，并反问用户希望了解的具体方向，禁止编造或臆测。
4. 反问应具体、可选项化（建议列出 2-4 个候选方向供用户选择），避免空泛的「请问您想了解什么？」。`

	kbQueryUserPromptTemplate = `参考资料：

%s

用户问题：

%s`
)

// KnowledgeBaseQueryService 实现 POST /api/knowledgebase/query 与 /query/stream：
// 对问题做 Embedding → pgvector 检索 knowledge_base_chunks → 拼装 prompt → Chat Completions（非流式或 SSE）。
type KnowledgeBaseQueryService struct {
	lg               *zap.Logger
	embedder         kbrepo.KnowledgeTextEmbedder
	reader           kbrepo.KnowledgeBaseReader
	searcher         kbrepo.KnowledgeVectorSearcher
	writer           kbrepo.KnowledgeBaseWriter
	chat             kbrepo.KnowledgeBaseQueryChat
	maxQuestionRunes int
}

// NewKnowledgeBaseQueryService Wire 注入；maxQuestionRunes 通常来自 RESUME_AI_MAX_RUNES（与上传侧全文上限口径一致）。
func NewKnowledgeBaseQueryService(
	lg *zap.Logger,
	embedder kbrepo.KnowledgeTextEmbedder,
	reader kbrepo.KnowledgeBaseReader,
	searcher kbrepo.KnowledgeVectorSearcher,
	writer kbrepo.KnowledgeBaseWriter,
	chat kbrepo.KnowledgeBaseQueryChat,
	maxQuestionRunes int,
) *KnowledgeBaseQueryService {
	return &KnowledgeBaseQueryService{
		lg:               lg,
		embedder:         embedder,
		reader:           reader,
		searcher:         searcher,
		writer:           writer,
		chat:             chat,
		maxQuestionRunes: maxQuestionRunes,
	}
}

// Query 非流式 JSON（Result.Data = KBQueryResponse）。
func (s *KnowledgeBaseQueryService) Query(ctx context.Context, v *model.ValidatedKBQuery) (*results.KBQueryResponse, error) {
	if err := s.checkDeps(); err != nil {
		return nil, err
	}
	workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), kbQueryLLMTimeout)
	defer cancel()

	sys, user, primaryID, namesJoined, noHit, err := s.buildPrompt(workCtx, v)
	if err != nil {
		return nil, err
	}
	if noHit {
		return &results.KBQueryResponse{
			Answer:            errmsg.KnowledgeBaseQueryNoHitResponse,
			KnowledgeBaseID:   primaryID,
			KnowledgeBaseName: namesJoined,
		}, nil
	}
	answer, err := s.chat.Complete(workCtx, sys, user)
	if err != nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseQueryFailed, zap.Error(err))
		}
		return nil, response.Err(http.StatusBadGateway, errmsg.KnowledgeBaseQueryFailedPrefix+err.Error())
	}
	if s.lg != nil {
		s.lg.Info(logmsg.MsgKnowledgeBaseQueryOK,
			zap.Int64("primaryKbId", primaryID),
			zap.Int("answerRunes", utf8.RuneCountInString(strings.TrimSpace(answer))),
		)
	}
	return &results.KBQueryResponse{
		Answer:            strings.TrimSpace(answer),
		KnowledgeBaseID:   primaryID,
		KnowledgeBaseName: namesJoined,
	}, nil
}

// QueryStream 写入 text/event-stream（每段正文一条或多条 data: 行 + 空行）；flush 在写完每个事件后调用（可选）。
// assistantAccumulator 非 nil 时追加模型输出的原始片段（不含 SSE 前缀），供 RAG 会话落库助手消息。
func (s *KnowledgeBaseQueryService) QueryStream(ctx context.Context, v *model.ValidatedKBQuery, w io.Writer, flush func(), assistantAccumulator *strings.Builder) error {
	if err := s.checkDeps(); err != nil {
		return err
	}
	workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), kbQueryLLMTimeout)
	defer cancel()

	sys, user, primaryID, _, noHit, err := s.buildPrompt(workCtx, v)
	if err != nil {
		return err
	}
	_ = primaryID
	if noHit {
		msg := errmsg.KnowledgeBaseQueryNoHitResponse
		if assistantAccumulator != nil {
			assistantAccumulator.WriteString(msg)
		}
		return writeSSEEvent(w, flush, msg)
	}
	err = s.chat.Stream(workCtx, sys, user, func(fragment string) error {
		if assistantAccumulator != nil {
			assistantAccumulator.WriteString(fragment)
		}
		return writeSSEEvent(w, flush, fragment)
	})
	if err != nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseQueryFailed, zap.Error(err))
		}
		return response.Err(http.StatusBadGateway, errmsg.KnowledgeBaseQueryFailedPrefix+err.Error())
	}
	return nil
}

func (s *KnowledgeBaseQueryService) checkDeps() error {
	if s == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseQueryServiceNil)
	}
	if s.embedder == nil || s.reader == nil || s.searcher == nil || s.writer == nil || s.chat == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseQueryDepsNil)
	}
	return nil
}

// buildPrompt 校验元数据、更新提问计数、检索上下文并拼装 system/user。noHit=true 时表示无需调用 LLM（固定话术）。
func (s *KnowledgeBaseQueryService) buildPrompt(ctx context.Context, v *model.ValidatedKBQuery) (systemPrompt, userPrompt string, primaryID int64, namesJoined string, noHit bool, err error) {
	if v == nil || len(v.KnowledgeBaseIDs) == 0 {
		return "", "", 0, "", false, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseQueryKnowledgeBaseIDsEmpty)
	}
	q := strings.TrimSpace(v.Question)
	if q == "" {
		return "", "", 0, "", false, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseQueryQuestionEmpty)
	}
	if s.maxQuestionRunes > 0 && utf8.RuneCountInString(q) > s.maxQuestionRunes {
		r := []rune(q)
		q = string(r[:s.maxQuestionRunes])
	}

	names := make([]string, 0, len(v.KnowledgeBaseIDs))
	for _, id := range v.KnowledgeBaseIDs {
		item, e := s.reader.GetKnowledgeBaseByID(ctx, id)
		if e != nil {
			return "", "", 0, "", false, e
		}
		if item == nil {
			return "", "", 0, "", false, response.Err(http.StatusNotFound, errmsg.KnowledgeBaseNotFound)
		}
		if strings.ToUpper(strings.TrimSpace(item.VectorStatus)) != "COMPLETED" {
			return "", "", 0, "", false, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseVectorNotReadyForQuery)
		}
		names = append(names, strings.TrimSpace(item.Name))
	}
	primaryID = v.KnowledgeBaseIDs[0]
	namesJoined = strings.Join(names, "、")

	if err := s.writer.IncrementQuestionCounts(ctx, v.KnowledgeBaseIDs); err != nil {
		return "", "", 0, "", false, err
	}

	if s.lg != nil {
		s.lg.Info(logmsg.MsgKnowledgeBaseQueryBegin,
			zap.Any("knowledgeBaseIds", v.KnowledgeBaseIDs),
			zap.Int("questionRunes", utf8.RuneCountInString(q)),
			zap.String("questionPreview", truncateRunes(q, kbQueryQuestionPreviewRunes)),
		)
	}

	vecs, e := s.embedder.Embed(ctx, []string{q})
	if e != nil {
		return "", "", 0, "", false, response.Err(http.StatusBadGateway, errmsg.KnowledgeBaseEmbeddingFailedPrefix+e.Error())
	}
	if len(vecs) != 1 || len(vecs[0]) == 0 {
		return "", "", 0, "", false, response.Err(http.StatusBadGateway, errmsg.KnowledgeBaseEmbeddingCountMismatch)
	}

	hits, e := s.searcher.SearchSimilarChunks(ctx, v.KnowledgeBaseIDs, vecs[0], kbQueryTopK)
	if e != nil {
		return "", "", 0, "", false, e
	}
	parts, rdiag := selectHitContents(hits, kbQueryMaxCosineDistance)
	if s.lg != nil {
		fields := []zap.Field{
			zap.Int("embeddingDim", len(vecs[0])),
			zap.Int("searchTopK", kbQueryTopK),
			zap.Float64("maxCosineDistance", kbQueryMaxCosineDistance),
			zap.Int("rawHitCount", rdiag.RawHitCount),
			zap.Int("keptChunkCount", rdiag.KeptChunkCount),
			zap.Int("droppedOverDistance", rdiag.DroppedOverDistance),
			zap.Int("droppedEmptyContent", rdiag.DroppedEmptyContent),
			zap.Any("topRawHits", rdiag.TopRawHits),
		}
		if rdiag.BestRawDistance != nil {
			fields = append(fields, zap.Float64("bestRawDistance", *rdiag.BestRawDistance))
		}
		if rdiag.WorstKeptDistance != nil {
			fields = append(fields, zap.Float64("worstKeptDistance", *rdiag.WorstKeptDistance))
		}
		s.lg.Info(logmsg.MsgKnowledgeBaseQueryRetrieval, fields...)
	}
	if len(parts) == 0 {
		return kbQuerySystemPrompt, "", primaryID, namesJoined, true, nil
	}
	ctxBlock := strings.Join(parts, "\n\n---\n\n")
	userPrompt = fmt.Sprintf(kbQueryUserPromptTemplate, ctxBlock, q)
	return kbQuerySystemPrompt, userPrompt, primaryID, namesJoined, false, nil
}

// kbHitLogPreview 检索汇总日志中单条命中（不做完整正文落日志）。
type kbHitLogPreview struct {
	ChunkID    int64   `json:"chunkId"`
	KBID       int64   `json:"kbId"`
	ChunkIndex int     `json:"chunkIndex"`
	Distance   float64 `json:"distance"`
	Preview    string  `json:"contentPreview"`
}

// kbRetrievalDiag 距阈值过滤前后的计数，便于判断「无命中」是检索空还是阈值过严。
type kbRetrievalDiag struct {
	RawHitCount         int
	KeptChunkCount      int
	DroppedOverDistance int
	DroppedEmptyContent int
	BestRawDistance     *float64
	WorstKeptDistance   *float64
	TopRawHits          []kbHitLogPreview
}

func selectHitContents(hits []kbrepo.KnowledgeChunkHit, maxDist float64) ([]string, kbRetrievalDiag) {
	diag := kbRetrievalDiag{
		RawHitCount: len(hits),
		TopRawHits:  make([]kbHitLogPreview, 0, min(kbRetrievalLogTopHits, len(hits))),
	}
	if len(hits) > 0 {
		d := hits[0].Distance
		diag.BestRawDistance = &d
	}
	out := make([]string, 0, len(hits))
	var worstKept float64
	var haveWorst bool
	for i, h := range hits {
		if i < kbRetrievalLogTopHits {
			diag.TopRawHits = append(diag.TopRawHits, kbHitLogPreview{
				ChunkID:    h.ChunkID,
				KBID:       h.KnowledgeBaseID,
				ChunkIndex: h.ChunkIndex,
				Distance:   h.Distance,
				Preview:    truncateRunes(strings.TrimSpace(h.Content), kbRetrievalPreviewRunes),
			})
		}
		if h.Distance > maxDist {
			diag.DroppedOverDistance++
			continue
		}
		t := strings.TrimSpace(h.Content)
		if t == "" {
			diag.DroppedEmptyContent++
			continue
		}
		out = append(out, t)
		if !haveWorst || h.Distance > worstKept {
			worstKept = h.Distance
			haveWorst = true
		}
	}
	diag.KeptChunkCount = len(out)
	if haveWorst {
		w := worstKept
		diag.WorstKeptDistance = &w
	}
	return out, diag
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "…"
}

// writeSSEEvent 输出一条 SSE「事件」：正文含换行时拆成多条 data:，最后以空行结束。
func writeSSEEvent(w io.Writer, flush func(), fragment string) error {
	if fragment == "" {
		return nil
	}
	for _, line := range strings.Split(fragment, "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}
	if flush != nil {
		flush()
	}
	return nil
}
