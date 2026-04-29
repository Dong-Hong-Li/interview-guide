// Package adapter 实现知识库模块对外模型网关的适配层。
//
// 本文件负责「文本 → 向量」：用户上传文档并经 AI 分块后，Redis Stream 消费者按批调用本适配器，
// 将每个文本块编码为浮点向量，写入 knowledge_base_chunks 并与配置的维度一致，供 pgvector 相似检索。
// 注意：Embeddings 使用的网关（KB_EMBEDDING_*）与主对话聊天网关（OPENAI_*）分离，避免混用配额与路由。
package adapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/internal/config"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/logmsg"

	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"go.uber.org/zap"
)

// OpenAIKnowledgeEmbedder 对接任意 OpenAI 兼容的 Embeddings HTTP 接口（含阿里云 DashScope 兼容模式等）。
//
// 业务含义：知识库检索不是全文 LIKE，而是「问题 embedding → 与库里块向量比距离」，因此每个可检索块必须先经过本组件得到向量。
type OpenAIKnowledgeEmbedder struct {
	client openaisdk.Client
	model  openaisdk.EmbeddingModel
	// dims 期望向量维度，须与数据库向量列、索引及检索侧查询向量一致；部分模型支持请求里传 dimensions 截断/指定维数。
	dims int64
	// lg 按「每次 Embed 调用」打结构化日志（runId、批次、kbId），便于在生产环境对照异步向量化失败与网关侧日志。
	lg *zap.Logger
	// batchSize 单次 HTTP 请求的文本条数上限；块很多时分批调用，降低单次超时与网关限流风险。
	batchSize int
}

var _ kbrepo.KnowledgeTextEmbedder = (*OpenAIKnowledgeEmbedder)(nil)

// NewOpenAIKnowledgeEmbedder 构造嵌入适配器；由 Wire 注入知识库查询服务与 Redis 向量化消费者共用同一工厂函数签名。
//
// 配置来源于 KB_EMBEDDING_MODEL、KB_EMBEDDING_DIMENSIONS、批量大小等（见 config.OpenAIConfig.KnowledgeEmbedding）。
func NewOpenAIKnowledgeEmbedder(client openaisdk.Client, cfg config.OpenAIConfig, lg *zap.Logger) kbrepo.KnowledgeTextEmbedder {
	bs := cfg.KnowledgeEmbedding.BatchSize
	if bs < 1 {
		bs = 10
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	return &OpenAIKnowledgeEmbedder{
		client:    client,
		model:     openaisdk.EmbeddingModel(strings.TrimSpace(cfg.KnowledgeEmbedding.Model)),
		dims:      int64(cfg.KnowledgeEmbedding.Dimensions),
		batchSize: bs,
		lg:        lg.Named("knowledge_embedding_http"),
	}
}

// Embed 实现端口 KnowledgeTextEmbedder：输入与上游分块顺序一致，输出 [][]float32 须逐行对齐，供 Mapper 写入 chunk 行。
//
// Context 中若通过 KnowledgeBaseVectorizeIDFromContext 注入了知识库 id，日志中会带 kbId，便于区分并行消费者的多条流水线。
func (e *OpenAIKnowledgeEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if e == nil || len(texts) == 0 {
		return nil, nil
	}
	out := make([][]float32, len(texts))
	bs := e.batchSize
	n := len(texts)
	totalBatches := (n + bs - 1) / bs
	runID := embeddingRunID(time.Now())
	runFields := []zap.Field{
		zap.String("runId", runID),
		zap.Int("inputChunks", n),
		zap.Int("embeddingBatchTotal", totalBatches),
	}
	if kbID, ok := kbrepo.KnowledgeBaseVectorizeIDFromContext(ctx); ok {
		runFields = append(runFields, zap.Int64("kbId", kbID))
	}
	e.lg.Info(logmsg.MsgKnowledgeEmbedBatchRun, runFields...)
	batchIdx := 0
	for start := 0; start < n; start += bs {
		batchIdx++
		end := start + bs
		if end > n {
			end = n
		}
		batch := texts[start:end]
		sliceEnd := end - 1
		batchIndex0 := batchIdx - 1
		part, err := e.embedBatch(ctx, batch, batchIdx, totalBatches, runID, start, sliceEnd, batchIndex0)
		if err != nil {
			return nil, err
		}
		copy(out[start:], part)
	}
	return out, nil
}

// embeddingRunID 生成一次 Embed 调用的可读编号，贯穿同一次「多批次 HTTP」的日志关联（非业务主键）。
func embeddingRunID(t time.Time) string {
	return fmt.Sprintf("%04d%02d%02d_%02d_%02d_%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(),
	)
}

// embeddingModelSupportsDimensionsParameter 判断请求里是否应携带 OpenAI 风格的 dimensions 字段。
//
// 背景：text-embedding-3-small/large、DashScope text-embedding-v3/v4 等支持多档维度；而 v1/v2 等为固定维度，
// 乱传 dimensions 可能导致网关报错，故对已知固定维模型关闭该参数。
func embeddingModelSupportsDimensionsParameter(model string) bool {
	m := strings.TrimSpace(model)
	if m == "text-embedding-v1" || m == "text-embedding-v2" {
		return false
	}
	switch {
	case strings.HasPrefix(m, "text-embedding-v4"):
		return true
	case strings.HasPrefix(m, "text-embedding-v3"):
		return true
	case strings.HasPrefix(m, "text-embedding-3"):
		return true
	default:
		return false
	}
}

// embedBatch 对一批文本发起一次 Embeddings.New；负责维度校验、按 index 回填向量及失败日志。
//
// sliceStart/sliceEnd 表示本批在整个 Embed 调用中的全局下标区间（便于日志排查「第几块」出问题）。
func (e *OpenAIKnowledgeEmbedder) embedBatch(ctx context.Context, batch []string, batchIdx, batchTotal int, runID string, sliceStart, sliceEnd int, batchIndex0 int) ([][]float32, error) {
	// 单批最多三分钟：向量网关偶发排队时略放宽，避免大面积误判失败。
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	tRound := time.Now()

	// 组装 Embeddings 请求：多字符串一次提交，网关返回与输入等长的 data[]，按 index 对齐。
	params := openaisdk.EmbeddingNewParams{
		Model: e.model,
		Input: openaisdk.EmbeddingNewParamsInputUnion{OfArrayOfStrings: batch},
	}
	reqDimsSet := e.dims > 0 && embeddingModelSupportsDimensionsParameter(string(e.model))
	if reqDimsSet {
		params.Dimensions = param.NewOpt(e.dims)
	}

	outFields := []zap.Field{
		zap.String("runId", runID),
		zap.String(logmsg.FieldModel, string(e.model)),
		zap.Int(logmsg.FieldEmbeddingBatchIndex, batchIdx),
		zap.Int(logmsg.FieldEmbeddingBatchTotal, batchTotal),
		zap.Int(logmsg.FieldEmbeddingSliceStart, sliceStart),
		zap.Int(logmsg.FieldEmbeddingSliceEnd, sliceEnd),
		zap.Int("inputCount", len(batch)),
		zap.Int("embeddingBatchOrdinal0", batchIndex0),
		zap.String("phase", "http_request_send"),
	}
	if kbID, ok := kbrepo.KnowledgeBaseVectorizeIDFromContext(ctx); ok {
		outFields = append(outFields, zap.Int64("kbId", kbID))
	}
	if reqDimsSet {
		outFields = append(outFields, zap.Int64(logmsg.FieldEmbeddingDimensionsRequest, e.dims))
	}
	e.lg.Info(logmsg.MsgKnowledgeEmbedBatchOutgoing, outFields...)

	resp, err := e.client.Embeddings.New(reqCtx, params)

	// roundTrip：用于衡量网关延迟，向量化积压时可辅助判断是否 Embeddings 侧变慢。
	roundTrip := time.Since(tRound)
	if err != nil {
		failFields := []zap.Field{
			zap.String("runId", runID),
			zap.Int(logmsg.FieldEmbeddingBatchIndex, batchIdx),
			zap.Int(logmsg.FieldEmbeddingBatchTotal, batchTotal),
			zap.Int(logmsg.FieldEmbeddingSliceStart, sliceStart),
			zap.Int(logmsg.FieldEmbeddingSliceEnd, sliceEnd),
			zap.Int("inputCount", len(batch)),
			zap.Int("embeddingBatchOrdinal0", batchIndex0),
			zap.Duration(logmsg.FieldEmbeddingBatchRoundTripDuration, roundTrip),
			zap.Error(err),
		}
		if kbID, ok := kbrepo.KnowledgeBaseVectorizeIDFromContext(ctx); ok {
			failFields = append(failFields, zap.Int64("kbId", kbID))
		}
		e.lg.Warn(logmsg.MsgKnowledgeEmbedBatchHTTPFailed, failFields...)
		return nil, err
	}

	if resp == nil || len(resp.Data) != len(batch) {
		cntErr := fmt.Errorf("%s", errmsg.KnowledgeBaseEmbeddingCountMismatch)
		cntFields := []zap.Field{
			zap.String("runId", runID),
			zap.Int(logmsg.FieldEmbeddingBatchIndex, batchIdx),
			zap.Duration(logmsg.FieldEmbeddingBatchRoundTripDuration, roundTrip),
			zap.Error(cntErr),
		}
		if kbID, ok := kbrepo.KnowledgeBaseVectorizeIDFromContext(ctx); ok {
			cntFields = append(cntFields, zap.Int64("kbId", kbID))
		}
		e.lg.Warn(logmsg.MsgKnowledgeEmbedBatchHTTPFailed, cntFields...)
		return nil, cntErr
	}

	// 按网关返回的 Index 把向量放回 batch 内对应位置，避免依赖响应顺序与请求顺序一致（防御性编程）。
	out := make([][]float32, len(batch))
	for _, item := range resp.Data {
		idx := int(item.Index)
		if idx < 0 || idx >= len(batch) {
			parseErr := fmt.Errorf("%s", errmsg.KnowledgeBaseEmbeddingCountMismatch)
			parseFields := []zap.Field{
				zap.String("runId", runID),
				zap.Int(logmsg.FieldEmbeddingBatchIndex, batchIdx),
				zap.Duration(logmsg.FieldEmbeddingBatchRoundTripDuration, roundTrip),
				zap.Error(parseErr),
			}
			if kbID, ok := kbrepo.KnowledgeBaseVectorizeIDFromContext(ctx); ok {
				parseFields = append(parseFields, zap.Int64("kbId", kbID))
			}
			e.lg.Warn(logmsg.MsgKnowledgeEmbedBatchHTTPFailed, parseFields...)
			return nil, parseErr
		}
		embVals := item.Embedding
		if e.dims > 0 && int64(len(embVals)) != e.dims {
			return nil, fmt.Errorf("gateway returned embedding length %d but KB_EMBEDDING_DIMENSIONS=%d: 请与模型实际输出维数一致",
				len(embVals), e.dims)
		}
		vec := make([]float32, len(embVals))
		for j, v := range embVals {
			vec[j] = float32(v)
		}
		out[idx] = vec
	}

	// 再次扫描：任一 index 未填充则说明网关返回残缺，禁止静默入库错误维度向量。
	for i := range out {
		if out[i] == nil {
			fillErr := fmt.Errorf("%s", errmsg.KnowledgeBaseEmbeddingCountMismatch)
			fillFields := []zap.Field{
				zap.String("runId", runID),
				zap.Int(logmsg.FieldEmbeddingBatchIndex, batchIdx),
				zap.Duration(logmsg.FieldEmbeddingBatchRoundTripDuration, roundTrip),
				zap.Error(fillErr),
			}
			if kbID, ok := kbrepo.KnowledgeBaseVectorizeIDFromContext(ctx); ok {
				fillFields = append(fillFields, zap.Int64("kbId", kbID))
			}
			e.lg.Warn(logmsg.MsgKnowledgeEmbedBatchHTTPFailed, fillFields...)
			return nil, fillErr
		}
	}

	respDim := 0
	if len(out) > 0 && len(out[0]) > 0 {
		respDim = len(out[0])
	}

	okFields := []zap.Field{
		zap.String("runId", runID),
		zap.String(logmsg.FieldModel, string(e.model)),
		zap.Int(logmsg.FieldEmbeddingBatchIndex, batchIdx),
		zap.Int(logmsg.FieldEmbeddingBatchTotal, batchTotal),
		zap.Int(logmsg.FieldEmbeddingSliceStart, sliceStart),
		zap.Int(logmsg.FieldEmbeddingSliceEnd, sliceEnd),
		zap.Int("inputCount", len(batch)),
		zap.Int(logmsg.FieldEmbeddingResponseCount, len(resp.Data)),
		zap.Int(logmsg.FieldEmbeddingResponseVectorDim, respDim),
		zap.Int("embeddingBatchOrdinal0", batchIndex0),
		zap.Duration(logmsg.FieldEmbeddingBatchRoundTripDuration, roundTrip),
		zap.String("phase", "http_response_ok"),
	}
	if kbID, ok := kbrepo.KnowledgeBaseVectorizeIDFromContext(ctx); ok {
		okFields = append(okFields, zap.Int64("kbId", kbID))
	}
	e.lg.Info(logmsg.MsgKnowledgeEmbedBatchReturned, okFields...)
	return out, nil
}
