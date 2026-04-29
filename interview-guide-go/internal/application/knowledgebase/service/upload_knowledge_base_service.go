package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"path/filepath"
	"strings"

	"interview-guide-go/internal/application/knowledgebase/model"
	"interview-guide-go/internal/application/knowledgebase/model/results"
	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/logmsg"
	"interview-guide-go/shared/response"

	"go.uber.org/zap"
)

// UploadKnowledgeBaseService 知识库上传用例：去重 → 解析 → 存储 → 落库 → 入队向量化。
// 只接收 controller 已校验并封装的 ValidatedKnowledgeBaseUpload；**不校验** HTTP 原始入参；正文抽取在 Upload 内完成。
type UploadKnowledgeBaseService struct {
	lg        *zap.Logger
	storage   kbrepo.ObjectStoragePort
	writer    kbrepo.KnowledgeBaseWriter
	vectorPub kbrepo.VectorizeTaskPublisher
	text      kbrepo.KnowledgeTextExtractor
}

// NewUploadKnowledgeBaseService Wire 注入。
func NewUploadKnowledgeBaseService(
	logger *zap.Logger,
	store kbrepo.ObjectStoragePort,
	writer kbrepo.KnowledgeBaseWriter,
	vectorPub kbrepo.VectorizeTaskPublisher,
	text kbrepo.KnowledgeTextExtractor,
) *UploadKnowledgeBaseService {
	return &UploadKnowledgeBaseService{
		lg:        logger,
		storage:   store,
		writer:    writer,
		vectorPub: vectorPub,
		text:      text,
	}
}

// Upload 处理已校验的上传请求（见 controller 与 ValidatedKnowledgeBaseUpload）。
func (s *UploadKnowledgeBaseService) Upload(ctx context.Context, in *model.ValidatedKnowledgeBaseUpload) (*results.UploadKnowledgeBaseResponse, error) {
	if s == nil || s.writer == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseWriterNotConfigured)
	}
	if in == nil {
		return nil, response.Err(http.StatusInternalServerError, "internal: validated upload payload is nil")
	}
	if s.text == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseTextExtractorNotConfigured)
	}
	filename := in.Filename
	contentType := in.ContentType
	content := in.Content
	name := in.Name
	category := in.Category
	// 4 去重：对原始字节做 hash（查 knowledge_bases.file_hash 唯一索引；若已存在则做 incrementAccessCount、保存，
	// 返回 duplicate=true、contentLength=0、不再次上传、不重新入队向量化（假定向量已存在）。
	fileHash := hashContent(content)
	// 5. 解析：从文件中抽取纯文本；若为空则返回业务错误「无法从文件中提取文本内容」。
	//    解析结果仅用于入队向量化，不在 knowledge_bases 表落库大文本，节省主表体积。
	parsedText := strings.TrimSpace(s.text.ExtractKnowledgeBaseText(content, filename, contentType))
	if parsedText == "" {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseUploadFailed,
				zap.String(logmsg.FieldReason, "extract_empty"),
				zap.String("filename", filename),
			)
		}
		return nil, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseExtractTextEmpty)
	}

	// 6. 对象存储：与简历上传同栈（S3/MinIO 等）写入 knowledge 前缀 key，取 fileUrl；
	// 失败则中止并清理。
	existing, err := s.writer.FindByFileHash(ctx, fileHash)
	if err != nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseUploadFailed,
				zap.String(logmsg.FieldReason, "hash_lookup"),
				zap.Error(err),
				zap.String("filename", filename),
			)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.FindKnowledgeBaseByHashFailed+err.Error())
	}

	// 重复上传时直接返回，不再重复上传与落库。
	if existing != nil {
		_ = s.writer.IncrementAccessCount(ctx, existing.ID)
		if s.lg != nil {
			s.lg.Info(logmsg.MsgKnowledgeBaseUploadDuplicate,
				zap.Int64("kbId", existing.ID),
				zap.String("filename", filename),
				zap.String(logmsg.FieldStatus, existing.VectorStatus),
			)
		}
		return &results.UploadKnowledgeBaseResponse{
			KnowledgeBase: results.UploadKBInfo{
				ID:            existing.ID,
				Name:          existing.Name,
				Category:      existing.Category,
				FileSize:      existing.FileSize,
				ContentLength: 0,
				VectorStatus:  existing.VectorStatus,
			},
			Storage: results.UploadKBStorage{
				FileKey: existing.StorageKey,
				FileURL: existing.StorageURL,
			},
			Duplicate: true,
		}, nil
	}
	if s.vectorPub == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseVectorPublisherNotConfigured)
	}
	// 获取存储 key
	storageKey := buildKnowledgeBaseObjectKey(fileHash, filename)
	if s.storage == nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseUploadFailed, zap.String(logmsg.FieldReason, "storage_nil"))
		}
		return nil, response.Err(http.StatusServiceUnavailable, "object storage not configured")
	}

	if s.lg != nil {
		s.lg.Info(logmsg.MsgKnowledgeBaseUploadBegin,
			zap.String("filename", filename),
			zap.String("name", name),
			zap.String("category", category),
			zap.Int("parsedTextRunes", len([]rune(parsedText))),
		)
	}

	// 上传：与简历上传同栈（S3/MinIO 等）写入 knowledge 前缀 key，取 fileUrl；失败则中止并清理。
	if err := s.storage.Upload(ctx, storageKey, bytes.NewReader(content), contentType); err != nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseUploadFailed,
				zap.String(logmsg.FieldReason, "storage"),
				zap.Error(err),
				zap.String("filename", filename),
			)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.UploadKnowledgeBaseFileFailed+err.Error())
	}

	// 预签名：取 presigned url。
	fileURL, err := s.storage.GetObjectPresignedURL(ctx, storageKey)
	if err != nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseUploadFailed,
				zap.String(logmsg.FieldReason, "presign"),
				zap.Error(err),
				zap.String("filename", filename),
			)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.GetKnowledgeBaseURLFailed+err.Error())
	}

	insert := &kbrepo.KnowledgeBaseInsert{
		FileHash:         fileHash,
		Name:             name,
		Category:         category,
		OriginalFilename: filename,
		FileSize:         int64(len(content)),
		ContentType:      contentType,
		StorageKey:       storageKey,
		StorageURL:       fileURL,
		VectorStatus:     "PENDING",
	}
	// 事务落库
	id, err := s.writer.InsertKnowledgeBase(ctx, insert)
	if err != nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseUploadFailed,
				zap.String(logmsg.FieldReason, "insert"),
				zap.Error(err),
				zap.String("filename", filename),
			)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.SaveKnowledgeBaseFailed+err.Error())
	}
	// 11.  主键小于 1 时 返回错误
	if id < 1 {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseUploadFailed,
				zap.String(logmsg.FieldReason, "invalid_id"),
				zap.Int64("kbId", id),
			)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.SaveKnowledgeBaseFailed+"invalid id")
	}

	// 12. 异步向量化：向 Redis Stream 发送任务（Producer 失败则 FAILED）。
	if err := s.vectorPub.SendVectorizeTask(ctx, id, parsedText); err != nil {
		_ = s.writer.UpdateVectorStatus(ctx, id, "FAILED", err.Error())
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgKnowledgeBaseUploadFailed,
				zap.String(logmsg.FieldReason, "enqueue"),
				zap.Int64("kbId", id),
				zap.Error(err),
			)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.SendVectorizeTaskFailed+err.Error())
	}
	if s.lg != nil {
		s.lg.Info(logmsg.MsgKnowledgeBaseUploadOK,
			zap.Int64("kbId", id),
			zap.Int("parsedTextRunes", len([]rune(parsedText))),
			zap.String("filename", filename),
		)
	}
	return &results.UploadKnowledgeBaseResponse{
		KnowledgeBase: results.UploadKBInfo{
			ID:            id,
			Name:          name,
			Category:      category,
			FileSize:      int64(len(content)),
			ContentLength: len(parsedText),
			VectorStatus:  "PENDING",
		},
		Storage: results.UploadKBStorage{
			FileKey: storageKey,
			FileURL: fileURL,
		},
		Duplicate: false,
	}, nil
}

func hashContent(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// knowledgebase/fileHash/originalFilename。
func buildKnowledgeBaseObjectKey(fileHash, original string) string {
	safe := filepath.Base(strings.TrimSpace(original))
	if safe == "" || safe == "." {
		safe = "file"
	}
	safe = strings.ReplaceAll(safe, "\\", "_")
	return "knowledgebase/" + fileHash + "/" + safe
}
