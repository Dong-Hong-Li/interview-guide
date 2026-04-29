package ai

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

// OpenAIKnowledgeEmbedder 使用 OpenAI 兼容 Embeddings API 为知识库分块生成向量。
type OpenAIKnowledgeEmbedder struct {
	client openaisdk.Client
	model  openaisdk.EmbeddingModel
	dims   int64
	// lg 每批请求/响应打到进程 stdout（与 cmd/server 共用 zap 配置），便于调试台可见。
	lg        *zap.Logger
	batchSize int
}

var _ kbrepo.KnowledgeTextEmbedder = (*OpenAIKnowledgeEmbedder)(nil)

// NewOpenAIKnowledgeEmbedder 由 deps 注入 Redis Stream 消费者。
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

// Embed 实现 KnowledgeTextEmbedder；与 texts 同序返回向量。
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

func embeddingRunID(t time.Time) string {
	return fmt.Sprintf("%04d%02d%02d_%02d_%02d_%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(),
	)
}

// embeddingModelSupportsDimensionsParameter 是否带 OpenAI 兼容字段 `dimensions`：
// - OpenAI：`text-embedding-3-small` / `large`
// - 阿里云 DashScope 兼容模式：仅官方写明支持的 **`text-embedding-v3`、`text-embedding-v4`**（见控制台「dimensions」说明）
// `text-embedding-v1` / `text-embedding-v2` 为固定维度，不传以免网关未定义。
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

// embedBatch 实现 KnowledgeTextEmbedder；与 batch 同序返回向量。
func (e *OpenAIKnowledgeEmbedder) embedBatch(ctx context.Context, batch []string, batchIdx, batchTotal int, runID string, sliceStart, sliceEnd int, batchIndex0 int) ([][]float32, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	tRound := time.Now()

	// 创建请求参数
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

	// 发送请求
	resp, err := e.client.Embeddings.New(reqCtx, params)

	// 计算请求耗时
	roundTrip := time.Since(tRound)
	if err != nil {
		// 请求失败
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

	// 检查响应数据
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

	// 处理响应数据
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

	// 检查输出数据
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

	// 记录日志
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
