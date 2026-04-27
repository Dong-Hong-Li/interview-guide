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

// 单块最大字符数（与后续接入 embedding 的 chunk 策略对齐；当前仅用于计数与状态回写）。
const knowledgeVectorizeChunkMaxRunes = 1500

// ensureKnowledgeVectorizeGroup 创建知识库向量化 Stream 的消费组。
func ensureKnowledgeVectorizeGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// StartKnowledgeVectorizeConsumer 启动知识库向量化消费者：读 `knowledge:vectorize:stream`，分块计数并回写 vector_status/chunk_count。
// 说明：向量写入 pgvector / embedding 尚未接入时，仍先将状态置为 COMPLETED 并写入 chunk_count，避免任务积压在 PEL；后续批次在消费者内接 embedding 即可。
func StartKnowledgeVectorizeConsumer(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, lg *zap.Logger) {
	if rdb == nil || writer == nil {
		return
	}
	consumer := fmt.Sprintf("knowledge-vectorize-go-%d", os.Getpid())
	go runKnowledgeVectorizeConsumer(ctx, rdb, writer, lg, consumer)
}

func runKnowledgeVectorizeConsumer(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, lg *zap.Logger, consumer string) {
	if err := ensureKnowledgeVectorizeGroup(ctx, rdb); err != nil {
		lg.Error(logmsg.MsgKnowledgeVectorizeCreateGroup, zap.Error(err))
		return
	}
	lg.Info(logmsg.MsgKnowledgeVectorizeConsumerStarted, zap.String(logmsg.FieldConsumer, consumer))

	for {
		select {
		case <-ctx.Done():
			lg.Info(logmsg.MsgKnowledgeVectorizeConsumerStopped)
			return
		default:
		}

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
				return
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

		for _, stream := range res {
			for _, msg := range stream.Messages {
				processKnowledgeVectorizeMessage(ctx, rdb, writer, lg, msg)
			}
		}
	}
}

// processKnowledgeVectorizeMessage 处理知识库向量化消息：分块计数并回写 vector_status/chunk_count。
func processKnowledgeVectorizeMessage(ctx context.Context, rdb *redis.Client, writer kbrepo.KnowledgeBaseWriter, lg *zap.Logger, msg redis.XMessage) {
	ack := func() {
		_ = rdb.XAck(ctx, streamkey.StreamKnowledgeVectorize, streamkey.GroupKnowledgeVectorize, msg.ID).Err()
	}

	kbIDStr := getStreamString(msg.Values, streamkey.StreamFieldKbID)
	if kbIDStr == "" {
		kbIDStr = fmt.Sprint(msg.Values[streamkey.StreamFieldKbID])
	}
	body := getStreamString(msg.Values, streamkey.StreamFieldKbContent)
	kbID, err := strconv.ParseInt(strings.TrimSpace(kbIDStr), 10, 64)
	if err != nil || kbID < 1 || body == "" {
		lg.Warn(logmsg.MsgKnowledgeVectorizeSkipBad, zap.String(logmsg.FieldID, msg.ID))
		ack()
		return
	}

	st, found, err := writer.GetVectorMetaByID(ctx, kbID)
	if err != nil {
		lg.Warn(logmsg.MsgKnowledgeVectorizeLoadMeta, zap.Int64("kbId", kbID), zap.Error(err))
		return
	}
	if !found {
		lg.Info(logmsg.MsgKnowledgeVectorizeRowGone, zap.Int64("kbId", kbID))
		ack()
		return
	}
	st = strings.ToUpper(strings.TrimSpace(st))
	if st == "COMPLETED" {
		lg.Info(logmsg.MsgKnowledgeVectorizeAlreadyDone, zap.Int64("kbId", kbID))
		ack()
		return
	}

	chunks := splitTextForVectorizeChunks(body)
	if len(chunks) == 0 {
		_ = writer.UpdateVectorStatus(ctx, kbID, "FAILED", errmsg.KnowledgeBaseVectorizeChunkEmpty)
		ack()
		return
	}

	// TODO: 在此对 chunks 调 embedding 并写入向量库后，再 MarkVectorizationComplete；当前先完成 Stream → DB 的闭环。
	if err := writer.MarkVectorizationComplete(ctx, kbID, len(chunks)); err != nil {
		lg.Warn(logmsg.MsgKnowledgeVectorizePersist, zap.Int64("kbId", kbID), zap.Error(err))
		return
	}
	lg.Info(logmsg.MsgKnowledgeVectorizeDone,
		zap.Int64("kbId", kbID),
		zap.Int("chunkCount", len(chunks)),
		zap.String(logmsg.FieldID, msg.ID),
	)
	ack()
}

func splitTextForVectorizeChunks(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if utf8.RuneCountInString(s) <= knowledgeVectorizeChunkMaxRunes {
		return []string{s}
	}
	var out []string
	runes := []rune(s)
	for i := 0; i < len(runes); i += knowledgeVectorizeChunkMaxRunes {
		end := i + knowledgeVectorizeChunkMaxRunes
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
	}
	return out
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
