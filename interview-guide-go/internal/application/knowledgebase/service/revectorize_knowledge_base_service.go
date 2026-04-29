package service

import (
	"context"
	"net/http"
	"strings"

	"interview-guide-go/internal/application/knowledgebase/model/results"
	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/logmsg"
	"interview-guide-go/shared/response"

	"go.uber.org/zap"
)

// RevectorizeKnowledgeBaseService POST /api/knowledgebase/{id}/revectorize：对已落库条目从对象存储拉原文、复抽正文，重置向量化状态并入队（与 Upload 后 SendVectorizeTask 语义一致）。
// 典型场景：向量任务 FAILED、更换 KB_CHUNK/KB_EMBEDDING 配置后需整库重算；不换 file_hash、不重传 multipart。
type RevectorizeKnowledgeBaseService struct {
	lg        *zap.Logger
	reader    kbrepo.KnowledgeBaseReader
	storage   kbrepo.ObjectStoragePort
	writer    kbrepo.KnowledgeBaseWriter
	vectorPub kbrepo.VectorizeTaskPublisher
	text      kbrepo.KnowledgeTextExtractor
}

// NewRevectorizeKnowledgeBaseService Wire 注入。
func NewRevectorizeKnowledgeBaseService(
	logger *zap.Logger,
	r kbrepo.KnowledgeBaseReader,
	store kbrepo.ObjectStoragePort,
	writer kbrepo.KnowledgeBaseWriter,
	vectorPub kbrepo.VectorizeTaskPublisher,
	text kbrepo.KnowledgeTextExtractor,
) *RevectorizeKnowledgeBaseService {
	return &RevectorizeKnowledgeBaseService{
		lg:        logger,
		reader:    r,
		storage:   store,
		writer:    writer,
		vectorPub: vectorPub,
		text:      text,
	}
}

// Revectorize 按 id 重新入队向量化；正文不落库，仅经队列传给消费者。
func (s *RevectorizeKnowledgeBaseService) Revectorize(ctx context.Context, id int64) (*results.RevectorizeKnowledgeBaseResponse, error) {
	if s == nil || s.reader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseRevectorizeServiceNil)
	}
	if s.storage == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseObjectStorageNotConfigured)
	}
	if s.writer == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseWriterNotConfigured)
	}
	if s.vectorPub == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseVectorPublisherNotConfigured)
	}
	if s.text == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseTextExtractorNotConfigured)
	}
	if id < 1 {
		return nil, response.Err(http.StatusBadRequest, "invalid knowledge base id")
	}

	// 获取知识库文件引用
	ref, err := s.reader.GetKnowledgeBaseFileRef(ctx, id)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.KnowledgeBaseNotFound)
	}
	key := strings.TrimSpace(ref.StorageKey)
	if key == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseNoStorageKey)
	}

	// 从对象存储获取文件数据
	data, _, err := s.storage.GetObject(ctx, key)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.GetKnowledgeBaseObjectFailed+err.Error())
	}

	filename := strings.TrimSpace(ref.OriginalFilename)
	ct := strings.TrimSpace(ref.ContentType)
	if ct == "" {
		ct = errmsg.ApplicationOctetStream
	}

	// 抽取正文
	parsedText := strings.TrimSpace(s.text.ExtractKnowledgeBaseText(data, filename, ct))
	if parsedText == "" {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseRevectorizeFailed,
				zap.Int64("kbId", id),
				zap.String(logmsg.FieldReason, "extract_empty"),
			)
		}
		return nil, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseExtractTextEmpty)
	}

	// 必须先置 PENDING：消费者若见 COMPLETED 会直接 ACK 跳过；与 Upload 初始态一致。
	if err := s.writer.UpdateVectorStatus(ctx, id, "PENDING", ""); err != nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseRevectorizeFailed,
				zap.Int64("kbId", id),
				zap.String(logmsg.FieldReason, "update_pending"),
				zap.Error(err),
			)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.KnowledgeBaseRevectorizeResetStatusFailed+err.Error())
	}

	// 发送向量化任务
	if err := s.vectorPub.SendVectorizeTask(ctx, id, parsedText); err != nil {
		_ = s.writer.UpdateVectorStatus(ctx, id, "FAILED", err.Error())
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseRevectorizeFailed,
				zap.Int64("kbId", id),
				zap.String(logmsg.FieldReason, "enqueue"),
				zap.Error(err),
			)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.SendVectorizeTaskFailed+err.Error())
	}

	// 返回结果
	runes := len([]rune(parsedText))
	if s.lg != nil {
		s.lg.Info(logmsg.MsgKnowledgeBaseRevectorizeOK,
			zap.Int64("kbId", id),
			zap.Int("parsedTextRunes", runes),
		)
	}
	return &results.RevectorizeKnowledgeBaseResponse{
		ID:              id,
		VectorStatus:    "PENDING",
		ParsedTextRunes: runes,
	}, nil
}
