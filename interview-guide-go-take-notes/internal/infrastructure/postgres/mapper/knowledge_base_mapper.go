package mapper

import (
	"context"
	"errors"
	"strings"

	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	grommodel "interview-guide-go/internal/infrastructure/postgres/grom"

	"gorm.io/gorm"
)

// KnowledgeBaseMapper 知识库表访问（knowledge_bases）。
type KnowledgeBaseMapper struct {
	db *gorm.DB
}

// NewKnowledgeBaseMapper 由 Wire 注入。
func NewKnowledgeBaseMapper(db *gorm.DB) *KnowledgeBaseMapper {
	return &KnowledgeBaseMapper{db: db}
}

var _ kbrepo.KnowledgeBaseWriter = (*KnowledgeBaseMapper)(nil)

// FindByFileHash 按 file_hash 唯一键查重；未命中返回 (nil, nil)。
func (m *KnowledgeBaseMapper) FindByFileHash(ctx context.Context, fileHash string) (*kbrepo.ExistingKnowledgeBase, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("db not configured")
	}
	h := strings.TrimSpace(fileHash)
	if h == "" {
		return nil, nil
	}
	var row grommodel.KnowledgeBase
	err := m.db.WithContext(ctx).Where("file_hash = ?", h).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fs := int64(0)
	if row.FileSize != nil {
		fs = *row.FileSize
	}
	return &kbrepo.ExistingKnowledgeBase{
		ID:           row.ID,
		Name:         row.Name,
		Category:     row.Category,
		FileSize:     fs,
		StorageKey:   row.StorageKey,
		StorageURL:   row.StorageURL,
		VectorStatus: row.VectorStatus,
	}, nil
}

// InsertKnowledgeBase 插入一行并返回主键。
func (m *KnowledgeBaseMapper) InsertKnowledgeBase(ctx context.Context, in *kbrepo.KnowledgeBaseInsert) (int64, error) {
	if m == nil || m.db == nil || in == nil {
		return 0, errors.New("invalid insert")
	}
	fs := in.FileSize
	row := grommodel.KnowledgeBase{
		FileHash:         in.FileHash,
		Name:             in.Name,
		Category:         strings.TrimSpace(in.Category),
		OriginalFilename: in.OriginalFilename,
		FileSize:         &fs,
		ContentType:      in.ContentType,
		StorageKey:       in.StorageKey,
		StorageURL:       in.StorageURL,
		VectorStatus:     in.VectorStatus,
		VectorError:      "",
	}
	if row.VectorStatus == "" {
		row.VectorStatus = "PENDING"
	}
	if err := m.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// IncrementAccessCount 重复上传时与 Java handleDuplicateKnowledgeBase 一致：access_count+1。
func (m *KnowledgeBaseMapper) IncrementAccessCount(ctx context.Context, id int64) error {
	if m == nil || m.db == nil || id < 1 {
		return errors.New("invalid id")
	}
	return m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).
		Where("id = ?", id).
		UpdateColumn("access_count", gorm.Expr("access_count + ?", 1)).Error
}

// UpdateVectorStatus 入队失败或消费者回写时用。
func (m *KnowledgeBaseMapper) UpdateVectorStatus(ctx context.Context, id int64, status, errMsg string) error {
	if m == nil || m.db == nil || id < 1 {
		return errors.New("invalid id")
	}
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	return m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"vector_status": status,
			"vector_error":  errMsg,
		}).Error
}

// GetVectorMetaByID 供向量化消费者判断记录是否存在及当前 vector_status。
func (m *KnowledgeBaseMapper) GetVectorMetaByID(ctx context.Context, id int64) (vectorStatus string, found bool, err error) {
	if m == nil || m.db == nil || id < 1 {
		return "", false, errors.New("invalid id")
	}
	var row grommodel.KnowledgeBase
	e := m.db.WithContext(ctx).Select("id", "vector_status").Where("id = ?", id).First(&row).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if e != nil {
		return "", false, e
	}
	return strings.TrimSpace(row.VectorStatus), true, nil
}

// MarkVectorizationComplete 消费者分块成功后回写 COMPLETED、chunk_count，并清空 vector_error。
func (m *KnowledgeBaseMapper) MarkVectorizationComplete(ctx context.Context, id int64, chunkCount int) error {
	if m == nil || m.db == nil || id < 1 {
		return errors.New("invalid id")
	}
	if chunkCount < 0 {
		chunkCount = 0
	}
	return m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"vector_status": "COMPLETED",
			"vector_error":  "",
			"chunk_count":   chunkCount,
		}).Error
}
