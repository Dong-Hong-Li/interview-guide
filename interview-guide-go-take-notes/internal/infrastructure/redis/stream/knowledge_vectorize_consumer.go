package redisstream

import (
	"context"
	"fmt"
	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/logmsg"
	"interview-guide-go/shared/streamkey"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func vectorizeTraceInfo(tlg *zap.Logger, step, next string, fields ...zap.Field) {
	x := []zap.Field{
		zap.String(logmsg.FieldVectorizeStep, step),
		zap.String(logmsg.FieldVectorizeNext, next),
		zap.String(logmsg.FieldOutcome, "success"),
	}
	x = append(x, fields...)
	tlg.Info(logmsg.MsgKnowledgeVectorizeTrace, x...)
}

func vectorizeTraceWarn(tlg *zap.Logger, step, next, reason string, fields ...zap.Field) {
	x := []zap.Field{
		zap.String(logmsg.FieldVectorizeStep, step),
		zap.String(logmsg.FieldVectorizeNext, next),
		zap.String(logmsg.FieldOutcome, "fail"),
		zap.String(logmsg.FieldReason, reason),
	}
	x = append(x, fields...)
	tlg.Warn(logmsg.MsgKnowledgeVectorizeTrace, x...)
}

// ensureKnowledgeVectorizeGroup 创建知识库向量化 Stream 的消费组。
func ensureKnowledgeVectorizeGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// StartKnowledgeVectorizeConsumer 启动知识库向量化消费者：读 `knowledge:vectorize:stream`，AI 分片→Embedding→写入 knowledge_base_chunks（pgvector）并回写 vector_status/chunk_count。
// 内置 panic 恢复与自动重启，确保消费者不会因单条消息或临时故障永久退出。
func StartKnowledgeVectorizeConsumer(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, chunker kbrepo.KnowledgeTextChunker, embedder kbrepo.KnowledgeTextEmbedder, lg *zap.Logger) {
	if rdb == nil || writer == nil || chunker == nil || embedder == nil {
		if lg != nil {
			lg.Error("知识库向量化消费者：依赖为 nil，消费者未启动",
				zap.Bool("redis", rdb != nil),
				zap.Bool("writer", writer != nil),
				zap.Bool("chunker", chunker != nil),
				zap.Bool("embedder", embedder != nil),
			)
		}
		return
	}
	consumer := fmt.Sprintf("knowledge-vectorize-go-%d", os.Getpid())
	go superviseKnowledgeVectorizeConsumer(ctx, rdb, writer, chunker, embedder, lg, consumer)
}

// superviseKnowledgeVectorizeConsumer 监督循环：runKnowledgeVectorizeLoop panic 或异常退出后自动重启，退避上限 30s。
func superviseKnowledgeVectorizeConsumer(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, chunker kbrepo.KnowledgeTextChunker, embedder kbrepo.KnowledgeTextEmbedder, lg *zap.Logger, consumer string) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		exitReason := runKnowledgeVectorizeLoop(ctx, rdb, writer, chunker, embedder, lg, consumer)
		if ctx.Err() != nil {
			return
		}
		lg.Error("知识库向量化消费者：主循环异常退出，将自动重启",
			zap.String("exitReason", exitReason),
			zap.Duration("restartAfter", backoff),
		)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// pelClaimMinIdle 超过此时间未 ACK 的 pending 消息会被当前消费者自动接管。
const pelClaimMinIdle = 5 * time.Minute

// runKnowledgeVectorizeLoop 实际消费循环；正常仅在 ctx 取消时返回 ""，异常时返回退出原因（供 supervisor 重启）。
func runKnowledgeVectorizeLoop(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, chunker kbrepo.KnowledgeTextChunker, embedder kbrepo.KnowledgeTextEmbedder, lg *zap.Logger, consumer string) (exitReason string) {
	defer func() {
		if r := recover(); r != nil {
			exitReason = fmt.Sprintf("panic: %v", r)
			lg.Error("知识库向量化消费者：捕获 panic",
				zap.Any("panic", r),
				zap.String(logmsg.FieldConsumer, consumer),
			)
		}
	}()

	// 创建消费组（带重试，最多 5 次，避免 Redis 短暂不可用就永久放弃）
	var groupErr error
	for attempt := 1; attempt <= 5; attempt++ {
		groupErr = ensureKnowledgeVectorizeGroup(ctx, rdb)
		if groupErr == nil {
			break
		}
		lg.Warn(logmsg.MsgKnowledgeVectorizeCreateGroup,
			zap.Error(groupErr),
			zap.Int("attempt", attempt),
		)
		if ctx.Err() != nil {
			return "ctx_canceled"
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	if groupErr != nil {
		return "create_group_failed_after_retries"
	}

	if envVectorizeAbortPendingOnStart() {
		ackN, skipped := abortPendingVectorizeOnStartup(ctx, rdb, writer, lg)
		lg.Warn(logmsg.MsgKnowledgeVectorizePendingAbortedOnStart,
			zap.Int("acked", ackN),
			zap.Int("skippedNoKbId", skipped),
		)
	}

	// ── 启动诊断：Stream 长度 + PEL 堆积 ──
	streamLen := rdb.XLen(ctx, streamkey.StreamKnowledgeVectorize).Val()
	pendingSummary, pelErr := rdb.XPending(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize).Result()
	var pelCount int64
	if pelErr == nil {
		pelCount = pendingSummary.Count
	}
	lg.Info(logmsg.MsgKnowledgeVectorizeConsumerStarted,
		zap.String(logmsg.FieldConsumer, consumer),
		zap.Int64("streamLen", streamLen),
		zap.Int64("pendingInPEL", pelCount),
		zap.Duration("pelClaimMinIdle", pelClaimMinIdle),
	)
	if pelCount > 0 {
		lg.Warn(logmsg.MsgKnowledgeVectorizePELBacklogHint,
			zap.String(logmsg.FieldConsumer, consumer),
			zap.Int64("pendingInPEL", pelCount),
			zap.Duration("pelClaimMinIdle", pelClaimMinIdle),
		)
	}

	var idlePolls int
	for {
		select {
		case <-ctx.Done():
			lg.Info(logmsg.MsgKnowledgeVectorizeConsumerStopped)
			return ""
		default:
		}

		// ── 每轮先回收 PEL 中超时的旧消息 ──
		claimKnowledgeVectorizePEL(ctx, rdb, writer, chunker, embedder, lg, consumer)

		res, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    streamkey.GroupKnowledgeVectorize,
			Consumer: consumer,
			Streams:  []string{streamkey.StreamKnowledgeVectorize, ">"},
			Count:    4,
			Block:    3 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil || err == context.Canceled {
				continue
			}
			if ctx.Err() != nil {
				return ""
			}
			if strings.Contains(strings.ToUpper(err.Error()), "NOGROUP") {
				if recErr := ensureKnowledgeVectorizeGroup(ctx, rdb); recErr != nil {
					lg.Warn(logmsg.MsgKnowledgeVectorizeXRead, zap.Error(recErr))
					time.Sleep(time.Second)
				}
				continue
			}
			lg.Warn(logmsg.MsgKnowledgeVectorizeXRead, zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		nMsgs := totalXReadMessages(res)
		if nMsgs == 0 {
			idlePolls++
			if idlePolls > 0 && idlePolls%20 == 0 {
				sl := rdb.XLen(ctx, streamkey.StreamKnowledgeVectorize).Val()
				ps, e := rdb.XPending(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize).Result()
				var pc int64
				if e == nil {
					pc = ps.Count
				}
				lg.Info(logmsg.MsgKnowledgeVectorizeIdleHint,
					zap.String("stream", streamkey.StreamKnowledgeVectorize),
					zap.Int64("streamLenApprox", sl),
					zap.Int64("pendingInPEL", pc),
					zap.Int("idlePolls", idlePolls),
				)
			}
			continue
		}
		idlePolls = 0

		for _, stream := range res {
			for _, msg := range stream.Messages {
				pulledFields := []zap.Field{
					zap.String(logmsg.FieldID, msg.ID),
					zap.String("stream", streamkey.StreamKnowledgeVectorize),
					zap.String("source", "new"),
				}
				if kbQuick, ok := parseStreamKbID(msg.Values); ok {
					pulledFields = append(pulledFields, zap.Int64("kbId", kbQuick))
				}
				lg.Info(logmsg.MsgKnowledgeVectorizePulled, pulledFields...)
				safeProcessKnowledgeVectorizeMessage(ctx, rdb, writer, chunker, embedder, lg, msg)
			}
		}
	}
}

// safeProcessKnowledgeVectorizeMessage 包裹 processKnowledgeVectorizeMessage 并捕获 panic，
// 确保单条消息的异常不会导致整个消费者 goroutine 退出。
func safeProcessKnowledgeVectorizeMessage(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, chunker kbrepo.KnowledgeTextChunker, embedder kbrepo.KnowledgeTextEmbedder, lg *zap.Logger, msg redis.XMessage) {
	defer func() {
		if r := recover(); r != nil {
			lg.Error("知识库向量化：处理单条消息时 panic，已恢复并跳过",
				zap.String(logmsg.FieldID, msg.ID),
				zap.Any("panic", r),
			)
			_ = rdb.XAck(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize, msg.ID).Err()
		}
	}()
	processKnowledgeVectorizeMessage(ctx, rdb, writer, chunker, embedder, lg, msg)
}

// claimKnowledgeVectorizePEL 用 XAUTOCLAIM 回收 PEL 中超过 pelClaimMinIdle 未被 ACK 的消息，
// 并逐条处理。避免消费者崩溃/重启后消息永远卡在 PEL。
func claimKnowledgeVectorizePEL(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, chunker kbrepo.KnowledgeTextChunker, embedder kbrepo.KnowledgeTextEmbedder, lg *zap.Logger, consumer string) {
	msgs, _, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   streamkey.StreamKnowledgeVectorize,
		Group:    streamkey.GroupKnowledgeVectorize,
		Consumer: consumer,
		MinIdle:  pelClaimMinIdle,
		Start:    "0-0",
		Count:    4,
	}).Result()
	if err != nil {
		if !strings.Contains(err.Error(), "NOGROUP") {
			lg.Warn("知识库向量化：XAUTOCLAIM 回收 PEL 失败", zap.Error(err))
		}
		return
	}
	if len(msgs) == 0 {
		return
	}
	lg.Info("知识库向量化：从 PEL 回收到滞留消息",
		zap.Int("claimedCount", len(msgs)),
		zap.Duration("minIdle", pelClaimMinIdle),
	)
	for _, msg := range msgs {
		pulledFields := []zap.Field{
			zap.String(logmsg.FieldID, msg.ID),
			zap.String("stream", streamkey.StreamKnowledgeVectorize),
			zap.String("source", "pel_reclaim"),
		}
		if kbQuick, ok := parseStreamKbID(msg.Values); ok {
			pulledFields = append(pulledFields, zap.Int64("kbId", kbQuick))
		}
		lg.Info(logmsg.MsgKnowledgeVectorizePulled, pulledFields...)
		processKnowledgeVectorizeMessage(ctx, rdb, writer, chunker, embedder, lg, msg)
	}
}

func totalXReadMessages(res []redis.XStream) int {
	n := 0
	for i := range res {
		n += len(res[i].Messages)
	}
	return n
}

func parseStreamKbID(values map[string]interface{}) (int64, bool) {
	kbIDStr := getStreamString(values, streamkey.StreamFieldKbID)
	if kbIDStr == "" && values != nil {
		if v, ok := values[streamkey.StreamFieldKbID]; ok && v != nil {
			kbIDStr = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	id, err := strconv.ParseInt(strings.TrimSpace(kbIDStr), 10, 64)
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// Env KB_VECTORIZE_ABORT_PENDING_ON_START：为 1/true/yes/on 时，启动消费循环前作废 PEL（DB 置 FAILED + XACK），便于重启后让用户重新入队。
func envVectorizeAbortPendingOnStart() bool {
	s := strings.ToLower(strings.TrimSpace(os.Getenv("KB_VECTORIZE_ABORT_PENDING_ON_START")))
	switch s {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// abortPendingVectorizeOnStartup 清空消费组 PEL：尽力按 kbId 写 FAILED，最后对每条执行 XACK。
func abortPendingVectorizeOnStartup(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, lg *zap.Logger) (acked int, skippedNoKbID int) {
	const maxRounds = 1000
	for round := 0; round < maxRounds; round++ {
		summary, err := rdb.XPending(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize).Result()
		if err != nil || summary.Count == 0 {
			break
		}
		pendings, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: streamkey.StreamKnowledgeVectorize,
			Group:  streamkey.GroupKnowledgeVectorize,
			Start:  "-",
			End:    "+",
			Count:  100,
		}).Result()
		if err != nil || len(pendings) == 0 {
			break
		}
		for i := range pendings {
			a, sk := abortOnePendingVectorizeMessage(ctx, rdb, writer, lg, pendings[i].ID)
			acked += a
			skippedNoKbID += sk
		}
	}
	return acked, skippedNoKbID
}

func abortOnePendingVectorizeMessage(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, lg *zap.Logger, id string) (acked int, skippedNoKbID int) {
	msgs := rdb.XRange(ctx, streamkey.StreamKnowledgeVectorize, id, id).Val()
	var vals map[string]interface{}
	if len(msgs) > 0 {
		vals = msgs[0].Values
	}
	kbID, ok := parseStreamKbID(vals)
	if ok && kbID >= 1 {
		if err := writer.UpdateVectorStatus(ctx, kbID, "FAILED", errmsg.KnowledgeBaseVectorizePendingDroppedOnStartup); err != nil {
			lg.Warn("知识库向量化：作废 PEL 写 FAILED 失败",
				zap.String(logmsg.FieldID, id),
				zap.Int64("kbId", kbID),
				zap.Error(err),
			)
		}
	} else {
		skippedNoKbID = 1
		lg.Warn("知识库向量化：作废 PEL 无法解析 kbId，仍将 ACK",
			zap.String(logmsg.FieldID, id),
		)
	}
	if err := rdb.XAck(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize, id).Err(); err != nil {
		lg.Warn("知识库向量化：作废 PEL XACK 失败",
			zap.String(logmsg.FieldID, id),
			zap.Error(err),
		)
		return 0, skippedNoKbID
	}
	return 1, skippedNoKbID
}

// 处理知识库向量化消息：分块 → Embedding → 写入 PG → 回写 COMPLETED。
func processKnowledgeVectorizeMessage(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, chunker kbrepo.KnowledgeTextChunker, embedder kbrepo.KnowledgeTextEmbedder, lg *zap.Logger, msg redis.XMessage) {
	ack := func() {
		_ = rdb.XAck(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize, msg.ID).Err()
	}

	// 获取知识库 ID
	kbIDStr := getStreamString(msg.Values, streamkey.StreamFieldKbID)
	if kbIDStr == "" {
		kbIDStr = fmt.Sprint(msg.Values[streamkey.StreamFieldKbID])
	}
	//知识库向量化异步任务 Stream
	body := getStreamString(msg.Values, streamkey.StreamFieldKbContent)
	// 解析知识库 ID
	kbID, err := strconv.ParseInt(strings.TrimSpace(kbIDStr), 10, 64)
	if err != nil || kbID < 1 || body == "" {
		lg.Warn(logmsg.MsgKnowledgeVectorizeSkipBad, zap.String(logmsg.FieldID, msg.ID))
		ack()
		return
	}
	tlg := lg.With(zap.Int64("kbId", kbID), zap.String(logmsg.FieldID, msg.ID))
	vectorizeTraceInfo(tlg, "msg_parsed", "load_pg_meta",
		zap.Int("bodyRunes", utf8.RuneCountInString(body)),
	)

	// 获取知识库向量化状态, 是否存在 不存在则记录日志
	st, found, err := writer.GetVectorMetaByID(ctx, kbID)
	if err != nil {
		em := errmsg.KnowledgeBaseVectorizeLoadMetaPrefix + err.Error()
		_ = writer.UpdateVectorStatus(ctx, kbID, "FAILED", truncateKnowledgeVectorError(em))
		vectorizeTraceWarn(tlg, "pg_meta", "abort", "db_error", zap.Error(err))
		tlg.Warn(logmsg.MsgKnowledgeVectorizeLoadMeta, zap.Error(err))
		ack()
		return
	}
	// 如果知识库不存在，则记录日志
	if !found {
		vectorizeTraceWarn(tlg, "pg_meta", "abort", "row_gone")
		tlg.Info(logmsg.MsgKnowledgeVectorizeRowGone)
		ack()
		return
	}
	// 如果知识库向量化状态为已完成，则记录日志
	st = strings.ToUpper(strings.TrimSpace(st))
	if st == "COMPLETED" {
		vectorizeTraceInfo(tlg, "pg_meta_skip", "redis_ack_dup", zap.String(logmsg.FieldStatus, st))
		tlg.Info(logmsg.MsgKnowledgeVectorizeAlreadyDone)
		ack()
		return
	}
	vectorizeTraceInfo(tlg, "pg_meta_ok", "split_chunks", zap.String("vectorStatus", st))

	workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Minute)
	workCtx = kbrepo.ContextWithKnowledgeBaseVectorizeID(workCtx, kbID)
	defer cancel()

	taskStart := time.Now()
	tSplit := time.Now()
	splitRes, err := chunker.SplitForVectorize(workCtx, body)
	chunkAIDur := time.Since(tSplit)
	if err != nil {
		em := errmsg.KnowledgeBaseChunkAIFailedPrefix + err.Error()
		_ = writer.UpdateVectorStatus(ctx, kbID, "FAILED", truncateKnowledgeVectorError(em))
		vectorizeTraceWarn(tlg, "chunk_ai", "abort", "chunk_ai_failed", zap.Error(err))
		tlg.Warn(logmsg.MsgKnowledgeVectorizeTaskFailed,
			zap.String(logmsg.FieldReason, "chunk_ai_failed"),
			zap.Error(err),
			zap.Duration("chunkAIDuration", chunkAIDur),
			zap.Duration(logmsg.FieldVectorizeDuration, time.Since(taskStart)),
		)
		ack()
		return
	}

	textChunks := splitRes.Chunks
	excCount := len(splitRes.Exceptions)
	tlg.Info(logmsg.MsgKnowledgeVectorizeChunkAIOutcome,
		zap.Int("chunkCount", len(textChunks)),
		zap.Int("exceptionCount", excCount),
		zap.Duration("chunkAIDuration", chunkAIDur),
	)
	for i, ex := range splitRes.Exceptions {
		tlg.Info(logmsg.MsgKnowledgeVectorizeChunkAIExceptionItem,
			zap.Int("exceptionIndex", i),
			zap.String("reason", ex.Reason),
			zap.String("raw_excerpt", ex.RawExcerpt),
		)
	}

	if len(textChunks) == 0 {
		_ = writer.UpdateVectorStatus(ctx, kbID, "FAILED", errmsg.KnowledgeBaseChunkAIEmptyChunks)
		vectorizeTraceWarn(tlg, "split_chunks", "abort", "chunk_empty_after_ai")
		tlg.Warn(logmsg.MsgKnowledgeVectorizeTaskFailed,
			zap.String(logmsg.FieldReason, "chunk_empty_after_ai"),
			zap.Int("exceptionCount", excCount),
		)
		ack()
		return
	}
	vectorizeTraceInfo(tlg, "chunks_split", "embed_http",
		zap.Int("chunkCount", len(textChunks)),
		zap.Int("bodyRunes", utf8.RuneCountInString(body)),
		zap.Int("exceptionCount", excCount),
	)

	tlg.Info(logmsg.MsgKnowledgeVectorizeTaskBegin,
		zap.Int("chunkCount", len(textChunks)),
		zap.Int("bodyRunes", utf8.RuneCountInString(body)),
		zap.String("vectorStatus", st),
		zap.Int("exceptionCount", excCount),
		zap.Duration("chunkAIDuration", chunkAIDur),
	)
	vectorizeTraceInfo(tlg, "embed_invoke", "await_embedding_gateway",
		zap.Int("chunkCount", len(textChunks)),
	)

	tEmbed := time.Now()
	vectors, err := embedder.Embed(workCtx, textChunks)
	embedDur := time.Since(tEmbed)
	vectorizeDur := time.Since(taskStart)
	if err != nil {
		em := errmsg.KnowledgeBaseEmbeddingFailedPrefix + err.Error()
		_ = writer.UpdateVectorStatus(ctx, kbID, "FAILED", truncateKnowledgeVectorError(em))
		vectorizeTraceWarn(tlg, "embed_http", "abort", "embedding", zap.Error(err))
		tlg.Warn(logmsg.MsgKnowledgeVectorizeTaskFailed,
			zap.String(logmsg.FieldReason, "embedding"),
			zap.Error(err),
			zap.Duration(logmsg.FieldEmbeddingDuration, embedDur),
			zap.Duration(logmsg.FieldVectorizeDuration, vectorizeDur),
		)
		ack()
		return
	}
	if len(vectors) != len(textChunks) {
		_ = writer.UpdateVectorStatus(ctx, kbID, "FAILED", errmsg.KnowledgeBaseEmbeddingCountMismatch)
		vectorizeTraceWarn(tlg, "embed_verify", "abort", "embedding_count_mismatch",
			zap.Int("chunkCount", len(textChunks)),
			zap.Int("vectorCount", len(vectors)),
		)
		tlg.Warn(logmsg.MsgKnowledgeVectorizeTaskFailed,
			zap.String(logmsg.FieldReason, "embedding_count_mismatch"),
			zap.Int("chunkCount", len(textChunks)),
			zap.Int("vectorCount", len(vectors)),
			zap.Duration(logmsg.FieldEmbeddingDuration, embedDur),
			zap.Duration(logmsg.FieldVectorizeDuration, time.Since(taskStart)),
		)
		ack()
		return
	}

	tlg.Info(logmsg.MsgKnowledgeVectorizeEmbedOK,
		zap.Int("chunkCount", len(textChunks)),
		zap.Duration(logmsg.FieldEmbeddingDuration, embedDur),
	)
	dimSample := 0
	if len(vectors) > 0 && len(vectors[0]) > 0 {
		dimSample = len(vectors[0])
	}
	vectorizeTraceInfo(tlg, "embed_ok", "pg_save_start",
		zap.Int("vectorRows", len(vectors)),
		zap.Int("responseVectorDimSample", dimSample),
	)

	rows := make([]kbrepo.KnowledgeBaseChunkInsert, len(textChunks))
	for i := range textChunks {
		rows[i] = kbrepo.KnowledgeBaseChunkInsert{
			ChunkIndex: i,
			Content:    textChunks[i],
			Embedding:  vectors[i],
		}
	}
	vectorizeTraceInfo(tlg, "pg_save_begin", "transaction_commit",
		zap.Int("chunkRowsToWrite", len(rows)),
	)
	if err := writer.SaveKnowledgeBaseVectorChunks(ctx, kbID, rows); err != nil {
		em := errmsg.KnowledgeBasePersistChunksFailedPrefix + err.Error()
		_ = writer.UpdateVectorStatus(ctx, kbID, "FAILED", truncateKnowledgeVectorError(em))
		vectorizeTraceWarn(tlg, "pg_save", "abort", "persist", zap.Error(err))
		tlg.Warn(logmsg.MsgKnowledgeVectorizePersist,
			zap.Error(err),
			zap.Duration(logmsg.FieldEmbeddingDuration, embedDur),
			zap.Duration(logmsg.FieldVectorizeDuration, time.Since(taskStart)),
		)
		ack()
		return
	}
	vectorizeTraceInfo(tlg, "pg_save_ok", "redis_xack",
		zap.Int("chunkCountWritten", len(textChunks)),
	)
	tlg.Info(logmsg.MsgKnowledgeVectorizeDone,
		zap.Int("chunkCount", len(textChunks)),
		zap.Duration(logmsg.FieldEmbeddingDuration, embedDur),
		zap.Duration(logmsg.FieldVectorizeDuration, time.Since(taskStart)),
	)
	if err := rdb.XAck(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize, msg.ID).Err(); err != nil {
		vectorizeTraceWarn(tlg, "redis_xack", "done_with_ack_err", "xack_failed", zap.Error(err))
		return
	}
	vectorizeTraceInfo(tlg, "redis_xack_ok", "pipeline_done",
		zap.Duration(logmsg.FieldVectorizeDuration, time.Since(taskStart)),
	)
}

func truncateKnowledgeVectorError(s string) string {
	if len(s) <= 500 {
		return s
	}
	return s[:500]
}

// 从 Redis Stream 消息中获取字符串值。
func getStreamString(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	v, ok := values[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
