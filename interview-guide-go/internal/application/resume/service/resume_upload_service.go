package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"interview-guide-go/internal/application/resume/model"
	result "interview-guide-go/internal/application/resume/model/results"
	"interview-guide-go/internal/application/resume/repository"
	domain_resume "interview-guide-go/internal/domain/resume"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
	sharedresume "interview-guide-go/shared/resume"
	"net/http"
	"strings"
	"time"
)

type ResumeUploadService struct {
	storage              repository.ObjectStoragePort
	MaxResumeUploadBytes int64
	parseService         repository.TextExtractor
	resumeWriter         repository.ResumeWriter
	analyzePublisher     repository.AnalyzePublisher
}

// NewResumeUploadService 组合根注入依赖；字段为小写故只能通过构造函数装配。
func NewResumeUploadService(
	store repository.ObjectStoragePort,
	maxResumeUploadBytes int64,
	text repository.TextExtractor,
	writer repository.ResumeWriter,
	analyzePublisher repository.AnalyzePublisher,
) *ResumeUploadService {
	return &ResumeUploadService{
		storage:              store,
		MaxResumeUploadBytes: maxResumeUploadBytes,
		parseService:         text,
		resumeWriter:         writer,
		analyzePublisher:     analyzePublisher,
	}
}

// UploadAndAnalyze 上传至对象存储、落库、投递简历分析队列（异步由消费者打模型分）。
func (o *ResumeUploadService) UploadAndAnalyze(ctx context.Context, request model.UploadResumeRequest) (*result.UploadResumeResult, error) {
	if o == nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.ResumeUploadServiceNil)
	}
	if o.resumeWriter == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.ResumePersistenceNotConfigured)
	}
	var (
		fileKey     string
		contentType string
		content     []byte
	)
	// 1. 验证请求参数
	fileKey, contentType, content, err := domain_resume.ValidateUploadResumeRequest(request)
	if err != nil {
		return nil, response.Err(http.StatusBadRequest, errmsg.ValidateUploadResumeRequestFailed+err.Error())
	}
	if !domain_resume.ValidateContentType(contentType) {
		return nil, response.Err(http.StatusBadRequest, errmsg.ResumeUnsupportedContentType)
	}
	if !domain_resume.ValidateContentSize(content, o.MaxResumeUploadBytes) {
		return nil, response.Err(http.StatusRequestEntityTooLarge, errmsg.ResumeFileExceedsSizeLimit)
	}
	// 2. 解析简历文本
	resumeText := o.parseService.ExtractResumeText(content, request.Filename, contentType)
	if resumeText == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.ResumeExtractTextEmpty)
	}

	// 3. 去重：若相同 file_hash 已存在则直接复用，不再重复上传与落库
	fileHash := hashResumeContent(content)
	existing, err := o.resumeWriter.FindByFileHash(ctx, fileHash)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.FindExistingResumeFailed+err.Error())
	}
	if existing != nil {
		return &result.UploadResumeResult{
			Storage: result.UploadStorage{
				FileKey:  existing.StorageKey,
				FileURL:  existing.StorageURL,
				ResumeID: existing.ID,
			},
			Duplicate: true,
		}, nil
	}

	// 4. 上传简历文件到对象存储
	err = o.storage.Upload(ctx, fileKey, bytes.NewReader(content), contentType)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.UploadFileFailed+err.Error())
	}

	fileURL, err := o.storage.GetObjectPresignedURL(ctx, fileKey)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.GetFileURLFailed+err.Error())
	}

	// 5. 保存简历到数据库（状态为 PENDING）
	size := int64(len(content))
	role := strings.TrimSpace(request.InterviewerRole)
	if role == "" {
		role = string(sharedresume.DefaultInterviewerRole)
	}
	resumeID, err := o.resumeWriter.InsertResume(ctx, &repository.ResumeInsert{
		FileHash:         fileHash,
		OriginalFilename: request.Filename,
		FileSize:         size,
		ContentType:      contentType,
		StorageKey:       fileKey,
		StorageURL:       fileURL,
		ResumeText:       resumeText,
		InterviewerRole:  role,
		AnalyzeStatus:    string(sharedresume.AnalyzeStatusPending),
		AnalyzeError:     "",
		AnalyzeTime:      time.Now().UTC(),
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), sharedresume.DuplicateDBErrorSubstr) {
			return nil, response.Err(http.StatusConflict, errmsg.DuplicateResumeSameFileHash)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.SaveResumeFailed+err.Error())
	}
	if resumeID < 1 {
		return nil, response.Err(http.StatusInternalServerError, errmsg.SaveResumeInvalidID)
	}

	// 6.添加简历分析队列
	if o.analyzePublisher != nil {
		err = o.analyzePublisher.SendAnalyzeTask(ctx, resumeID, resumeText)
		if err != nil {
			return nil, response.Err(http.StatusInternalServerError, errmsg.SendAnalyzeTaskFailed+err.Error())
		}
	}

	return &result.UploadResumeResult{
		Storage: result.UploadStorage{
			FileKey:  fileKey,
			FileURL:  fileURL,
			ResumeID: resumeID,
		},
		Duplicate: false,
	}, nil
}

// 计算简历内容的 SHA-256 哈希值
func hashResumeContent(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
