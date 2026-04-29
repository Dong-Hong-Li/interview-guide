package mapper

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"interview-guide-go/internal/application/knowledgebase/model/results"
	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	grommodel "interview-guide-go/internal/infrastructure/postgres/grom"

	"github.com/pgvector/pgvector-go"
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
var _ kbrepo.KnowledgeBaseReader = (*KnowledgeBaseMapper)(nil)

// GetKnowledgeBaseFileRef 取存储 key 与元数据；不存在返回 (nil, nil)。
func (m *KnowledgeBaseMapper) GetKnowledgeBaseFileRef(ctx context.Context, id int64) (*kbrepo.KnowledgeBaseFileRef, error) {
	if m == nil || m.db == nil || id < 1 {
		return nil, errors.New("invalid id")
	}
	var row grommodel.KnowledgeBase
	err := m.db.WithContext(ctx).
		Select("storage_key", "original_filename", "content_type").
		Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &kbrepo.KnowledgeBaseFileRef{
		StorageKey:       strings.TrimSpace(row.StorageKey),
		OriginalFilename: strings.TrimSpace(row.OriginalFilename),
		ContentType:      strings.TrimSpace(row.ContentType),
	}, nil
}

// GetKnowledgeBaseByID 单条详情；不存在返回 (nil, nil)。
func (m *KnowledgeBaseMapper) GetKnowledgeBaseByID(ctx context.Context, id int64) (*results.KnowledgeBaseListItem, error) {
	if m == nil || m.db == nil || id < 1 {
		return nil, errors.New("invalid id")
	}
	var row grommodel.KnowledgeBase
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item := knowledgeBaseRowToListItem(&row)
	return &item, nil
}

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
	// 使用 Find + RowsAffected：First 在未命中时会触发 ErrRecordNotFound，
	// GORM 默认 Logger 会打出「record not found」，易被误判为故障；此处「无记录」为首次上传的正常路径。
	tx := m.db.WithContext(ctx).Where("file_hash = ?", h).Limit(1).Find(&row)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, nil
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

// SaveKnowledgeBaseVectorChunks 事务内替换该 KB 的分块向量并将 knowledge_bases 置 COMPLETED。
func (m *KnowledgeBaseMapper) SaveKnowledgeBaseVectorChunks(ctx context.Context, knowledgeBaseID int64, chunks []kbrepo.KnowledgeBaseChunkInsert) error {
	if m == nil || m.db == nil || knowledgeBaseID < 1 {
		return errors.New("invalid knowledge base chunk save")
	}
	if len(chunks) == 0 {
		return errors.New("empty chunks")
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("knowledge_base_id = ?", knowledgeBaseID).Delete(&grommodel.KnowledgeBaseChunk{}).Error; err != nil {
			return err
		}
		for _, row := range chunks {
			if strings.TrimSpace(row.Content) == "" || len(row.Embedding) == 0 {
				return fmt.Errorf("invalid chunk index=%d", row.ChunkIndex)
			}
			rec := grommodel.KnowledgeBaseChunk{
				KnowledgeBaseID: knowledgeBaseID,
				ChunkIndex:      row.ChunkIndex,
				Content:         row.Content,
				Embedding:       pgvector.NewVector(row.Embedding),
			}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		}
		return tx.Model(&grommodel.KnowledgeBase{}).Where("id = ?", knowledgeBaseID).Updates(map[string]interface{}{
			"vector_status": "COMPLETED",
			"vector_error":  "",
			"chunk_count":   len(chunks),
		}).Error
	})
}

// DeleteKnowledgeBaseByID 删除 `knowledge_bases` 行；未删到行时返回 ErrKnowledgeBaseDeleteNoRow（并发已删场景）。
func (m *KnowledgeBaseMapper) DeleteKnowledgeBaseByID(ctx context.Context, id int64) error {
	if m == nil || m.db == nil || id < 1 {
		return errors.New("invalid id")
	}
	res := m.db.WithContext(ctx).Delete(&grommodel.KnowledgeBase{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return kbrepo.ErrKnowledgeBaseDeleteNoRow
	}
	return nil
}

// UpdateKnowledgeBaseCategory 更新 category；空串表示未分类（与 ListUncategorized 条件一致）。
func (m *KnowledgeBaseMapper) UpdateKnowledgeBaseCategory(ctx context.Context, id int64, category string) error {
	if m == nil || m.db == nil || id < 1 {
		return errors.New("invalid id")
	}
	tx := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).
		Where("id = ?", id).
		Update("category", category)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return kbrepo.ErrKnowledgeBaseUpdateNoRow
	}
	return nil
}

// ListKnowledgeBases 列表：默认按 uploaded_at 倒序；可选按 vector_status 过滤。
func (m *KnowledgeBaseMapper) ListKnowledgeBases(ctx context.Context, vectorStatus *string) ([]results.KnowledgeBaseListItem, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("db not configured")
	}
	q := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{})
	if vectorStatus != nil {
		vs := strings.ToUpper(strings.TrimSpace(*vectorStatus))
		if vs != "" {
			q = q.Where("UPPER(TRIM(COALESCE(vector_status,''))) = ?", vs)
		}
	}
	var rows []grommodel.KnowledgeBase
	if err := q.Order("uploaded_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]results.KnowledgeBaseListItem, 0, len(rows))
	for i := range rows {
		out = append(out, knowledgeBaseRowToListItem(&rows[i]))
	}
	return out, nil
}

// SearchKnowledgeBases 与 JPA searchByKeyword 语义：LOWER 模糊匹配 name 与 original_filename，按上传时间倒序。
func (m *KnowledgeBaseMapper) SearchKnowledgeBases(ctx context.Context, keyword string) ([]results.KnowledgeBaseListItem, error) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return m.ListKnowledgeBases(ctx, nil)
	}
	if m == nil || m.db == nil {
		return nil, errors.New("db not configured")
	}
	pat := "%" + kw + "%"
	var rows []grommodel.KnowledgeBase
	err := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).
		Where("name ILIKE ? OR original_filename ILIKE ?", pat, pat).
		Order("uploaded_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]results.KnowledgeBaseListItem, 0, len(rows))
	for i := range rows {
		out = append(out, knowledgeBaseRowToListItem(&rows[i]))
	}
	return out, nil
}

// ListByCategory TRIM(category) 与入参精确一致；uploaded_at 倒序。
func (m *KnowledgeBaseMapper) ListByCategory(ctx context.Context, category string) ([]results.KnowledgeBaseListItem, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("db not configured")
	}
	cat := strings.TrimSpace(category)
	if cat == "" {
		return nil, errors.New("empty category")
	}
	var rows []grommodel.KnowledgeBase
	err := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).
		Where("TRIM(COALESCE(category, '')) = ?", cat).
		Order("uploaded_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]results.KnowledgeBaseListItem, 0, len(rows))
	for i := range rows {
		out = append(out, knowledgeBaseRowToListItem(&rows[i]))
	}
	return out, nil
}

// ListUncategorized 与 categories 中「空串/纯空白 = 无分类」一致；uploaded_at 倒序。
func (m *KnowledgeBaseMapper) ListUncategorized(ctx context.Context) ([]results.KnowledgeBaseListItem, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("db not configured")
	}
	var rows []grommodel.KnowledgeBase
	err := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).
		Where("NULLIF(TRIM(COALESCE(category, '')), '') IS NULL").
		Order("uploaded_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]results.KnowledgeBaseListItem, 0, len(rows))
	for i := range rows {
		out = append(out, knowledgeBaseRowToListItem(&rows[i]))
	}
	return out, nil
}

// ListDistinctCategories 非空分类去重、字典序。
func (m *KnowledgeBaseMapper) ListDistinctCategories(ctx context.Context) ([]string, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("db not configured")
	}
	var cats []string
	// 与 JPA "WHERE category IS NOT NULL ORDER BY category" 对齐；空串视为无分类
	err := m.db.WithContext(ctx).Raw(`
		SELECT DISTINCT TRIM(category) AS c
		FROM knowledge_bases
		WHERE NULLIF(TRIM(category), '') IS NOT NULL
		ORDER BY 1
	`).Scan(&cats).Error
	if err != nil {
		return nil, err
	}
	// 避免 nil slice 被 json 编码为 null；前端 categories.map 会抛错白屏
	if cats == nil {
		return []string{}, nil
	}
	return cats, nil
}

// GetStatistics 与 Java getStatistics 字段一致。
func (m *KnowledgeBaseMapper) GetStatistics(ctx context.Context) (*results.KnowledgeBaseStats, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("db not configured")
	}
	var total, completed, processing, userMsgs, sumAccess int64
	if err := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).Count(&total).Error; err != nil {
		return nil, err
	}
	if err := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).Where("UPPER(TRIM(vector_status)) = ?", "COMPLETED").Count(&completed).Error; err != nil {
		return nil, err
	}
	if err := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).Where("UPPER(TRIM(vector_status)) = ?", "PROCESSING").Count(&processing).Error; err != nil {
		return nil, err
	}
	if err := m.db.WithContext(ctx).Model(&grommodel.RagChatMessage{}).
		Where("LOWER(TRIM(COALESCE(type,''))) = ?", "user").Count(&userMsgs).Error; err != nil {
		return nil, err
	}
	if err := m.db.WithContext(ctx).Raw(`SELECT COALESCE(SUM(access_count),0) FROM knowledge_bases`).Scan(&sumAccess).Error; err != nil {
		return nil, err
	}
	return &results.KnowledgeBaseStats{
		TotalCount:         total,
		TotalQuestionCount: userMsgs,
		TotalAccessCount:   sumAccess,
		CompletedCount:     completed,
		ProcessingCount:    processing,
	}, nil
}

func knowledgeBaseRowToListItem(row *grommodel.KnowledgeBase) results.KnowledgeBaseListItem {
	var cat *string
	if t := strings.TrimSpace(row.Category); t != "" {
		cat = &t
	}
	fs := int64(0)
	if row.FileSize != nil {
		fs = *row.FileSize
	}
	ve := strings.TrimSpace(row.VectorError)
	return results.KnowledgeBaseListItem{
		ID:               row.ID,
		Name:             row.Name,
		Category:         cat,
		OriginalFilename: row.OriginalFilename,
		FileSize:         fs,
		ContentType:      strings.TrimSpace(row.ContentType),
		UploadedAt:       row.UploadedAt,
		LastAccessedAt:   row.LastAccessedAt,
		AccessCount:      row.AccessCount,
		QuestionCount:    row.QuestionCount,
		VectorStatus:     strings.TrimSpace(row.VectorStatus),
		VectorError:      ve,
		ChunkCount:       row.ChunkCount,
	}
}
