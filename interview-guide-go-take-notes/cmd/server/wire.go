// wireinject
//go:build wireinject
// +build wireinject

// package
package main

// Package main 的 wire 声明文件：由 wire 工具读取 Build(...) 指令生成 wire_gen.go。
//
// 使用：安装一次 `go install github.com/google/wire/cmd/wire@latest`，
// 之后在本目录执行 `wire` 即可重新生成 wire_gen.go。

import (
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	ivctl "interview-guide-go/internal/application/interview/controller"
	ivrepo "interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/internal/application/interview/service"
	kbctl "interview-guide-go/internal/application/knowledgebase/controller"
	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	kbsvc "interview-guide-go/internal/application/knowledgebase/service"
	ragctl "interview-guide-go/internal/application/ragchat/controller"
	ragrepo "interview-guide-go/internal/application/ragchat/repository"
	ragsvc "interview-guide-go/internal/application/ragchat/service"
	resume "interview-guide-go/internal/application/resume/controller"
	resumerepo "interview-guide-go/internal/application/resume/repository"
	resumesvc "interview-guide-go/internal/application/resume/service"
	"interview-guide-go/internal/config"
	"interview-guide-go/internal/infrastructure/ai"
	aiadapter "interview-guide-go/internal/infrastructure/ai/adapter"
	fileadapter "interview-guide-go/internal/infrastructure/file/adapter"
	"interview-guide-go/internal/infrastructure/postgres/mapper"
	redisadapter "interview-guide-go/internal/infrastructure/redis/adapter"
	"interview-guide-go/internal/infrastructure/storage"
	storageadapter "interview-guide-go/internal/infrastructure/storage/adapter"
)

// 一捆「能生产东西的函数」，Wire 会在这捆里解依赖图。
var resumeModuleSet = wire.NewSet(
	provideMaxResumeUploadBytes,
	mapper.NewResumeMapper,
	wire.Bind(new(resumerepo.ResumeWriter), new(*mapper.ResumeMapper)),
	storageadapter.NewObjectStorageAdapter,
	redisadapter.NewAnalyzePublisher,
	fileadapter.NewTextExtractorAdapter,
	resumesvc.NewResumeUploadService,
	resumesvc.NewInterviewerRolesService,
	resumesvc.NewResumeListService,
	resumesvc.NewResumeDeleteService,
	resumesvc.NewResumeDetailService,
	resumesvc.NewReanalyzeResumeService,
	resumesvc.NewExportAnalysisPDFService,
	wire.Struct(new(resume.ResumeController), "*"),
)

// 面试模块：会话写库、简历人设（*ResumeMapper 实现 InterviewerRoleReader）、缓存、出题、CreateInterview 用例、控制器。
var interviewModuleSet = wire.NewSet(
	mapper.NewResumeMapper,
	wire.Bind(new(ivrepo.InterviewerRoleReader), new(*mapper.ResumeMapper)),
	wire.Bind(new(ivrepo.ResumeTextSource), new(*mapper.ResumeMapper)),
	mapper.NewInterviewMapper,
	wire.Bind(new(ivrepo.InterviewSessionWriter), new(*mapper.InterviewMapper)),
	redisadapter.NewEvaluateEnqueue,
	wire.Bind(new(ivrepo.InterviewEvaluateEnqueuer), new(*redisadapter.EvaluateEnqueue)),
	provideInterviewSessionCache,
	provideInterviewQuestionGenerator,
	service.NewCreateInterviewService,
	service.NewUnfinishedSessionService,
	service.NewCurrentQuestionService,
	service.NewSubmitAnswerService,
	service.NewListInterviewSessionsService,
	service.NewReportService,
	service.NewGetInterviewDetailService,
	service.NewGetSessionService,
	service.NewCompleteSessionService,
	service.NewDeleteInterviewService,
	wire.Struct(new(ivctl.InterviewController), "*"),
)

// 知识库：上传、列表、向量检索问答（query/query/stream）、下载、重向量化等。
var knowledgeModuleSet = wire.NewSet(
	mapper.NewKnowledgeBaseMapper,
	wire.Bind(new(kbrepo.KnowledgeBaseWriter), new(*mapper.KnowledgeBaseMapper)),
	wire.Bind(new(kbrepo.KnowledgeBaseReader), new(*mapper.KnowledgeBaseMapper)),
	wire.Bind(new(kbrepo.KnowledgeVectorSearcher), new(*mapper.KnowledgeBaseMapper)),
	storageadapter.NewObjectStorageAdapter,
	fileadapter.NewKnowledgeTextExtractor,
	redisadapter.NewKnowledgeVectorizePublisher,
	provideKnowledgeBaseEmbedder,
	provideKnowledgeBaseQueryChat,
	provideKBQueryMaxQuestionRunes,
	kbsvc.NewUploadKnowledgeBaseService,
	kbsvc.NewKnowledgeBaseListService,
	kbsvc.NewDeleteKnowledgeBaseService,
	kbsvc.NewDownloadKnowledgeBaseService,
	kbsvc.NewUpdateKnowledgeBaseCategoryService,
	kbsvc.NewRevectorizeKnowledgeBaseService,
	kbsvc.NewKnowledgeBaseQueryService,
	wire.Struct(new(kbctl.KnowledgeBaseController), "*"),
)

// RAG 对话：会话 CRUD + messages/stream（复用 KnowledgeBaseQueryService RAG 链）。
var ragModuleSet = wire.NewSet(
	mapper.NewKnowledgeBaseMapper,
	wire.Bind(new(kbrepo.KnowledgeBaseWriter), new(*mapper.KnowledgeBaseMapper)),
	wire.Bind(new(kbrepo.KnowledgeBaseReader), new(*mapper.KnowledgeBaseMapper)),
	wire.Bind(new(kbrepo.KnowledgeVectorSearcher), new(*mapper.KnowledgeBaseMapper)),
	provideKnowledgeBaseEmbedder,
	provideKnowledgeBaseQueryChat,
	provideKBQueryMaxQuestionRunes,
	kbsvc.NewKnowledgeBaseQueryService,
	mapper.NewRagChatMapper,
	wire.Bind(new(ragrepo.RagChatRepository), new(*mapper.RagChatMapper)),
	ragsvc.NewRagChatSessionService,
	ragsvc.NewRagChatStreamService,
	wire.Struct(new(ragctl.RagChatController), "*"),
)

// provideMaxResumeUploadBytes 抽取 cfg 字段给 ResumeUploadService 用，避免 wire 直接把 cfg.Xxx 当 int64 provider。
func provideMaxResumeUploadBytes(cfg *config.Config) int64 {
	return cfg.MaxResumeUploadBytes
}

// provideInterviewSessionCache 将 *redisadapter.SessionCache 以接口形式注入，避免与 NewSessionCache 的重复 interface 绑定。
func provideInterviewSessionCache(rdb *redis.Client) ivrepo.InterviewSessionCache {
	return redisadapter.NewSessionCache(rdb)
}

// provideInterviewQuestionGenerator 依赖可用的 OpenAI 客户端；不满足则 panic（禁止 Stub 兜底）。
func provideInterviewQuestionGenerator(oa *ai.OpenAIService, cfg *config.Config, lg *zap.Logger) ivrepo.InterviewQuestionGenerator {
	if oa == nil {
		panic("interview-guide-go: OpenAIService required for InterviewQuestionGenerator")
	}
	return aiadapter.NewOpenAIInterviewQuestionGenerator(oa, cfg, lg)
}

// provideKnowledgeBaseEmbedder 与向量化消费者同源：KB_EMBEDDING_* 网关 + OpenAIKnowledgeEmbedder。
func provideKnowledgeBaseEmbedder(cfg *config.Config, lg *zap.Logger, oa *ai.OpenAIService) kbrepo.KnowledgeTextEmbedder {
	c, err := ai.EmbeddingHTTPClient(cfg, oa)
	if err != nil {
		panic("interview-guide-go: EmbeddingHTTPClient: " + err.Error())
	}
	return aiadapter.NewOpenAIKnowledgeEmbedder(c, cfg.Openai, lg)
}

// provideKnowledgeBaseQueryChat 主对话网关 Chat Completions（OPENAI_*），供 query / query/stream 生成答案。
func provideKnowledgeBaseQueryChat(cfg *config.Config, oa *ai.OpenAIService) kbrepo.KnowledgeBaseQueryChat {
	if cfg == nil || oa == nil {
		panic("interview-guide-go: KnowledgeBaseQueryChat prerequisites")
	}
	o := cfg.Openai
	return aiadapter.NewKnowledgeBaseQueryChatAdapter(oa, o.AIModel, o.ResumeAIMaxCompletionTokens, o.ResumeAITemperature)
}

// provideKBQueryMaxQuestionRunes 抽 cfg，供注入 KnowledgeBaseQueryService（问题过长按 rune 截断）。
func provideKBQueryMaxQuestionRunes(cfg *config.Config) int {
	if cfg == nil {
		return 0
	}
	return cfg.Openai.ResumeAIMaxRunes
}

// initializeResumeController 生成简历控制器。
// 消费者启动所需的 repository.ResumeWriter 由 deps.go 单独调 mapper.NewResumeMapper 得到（Mapper 只是 *gorm.DB 的薄封装，多实例化无副作用）。
func initializeResumeController(
	cfg *config.Config,
	lg *zap.Logger,
	db *gorm.DB,
	rdb *redis.Client,
	storeSvc *storage.StorageService,
) *resume.ResumeController {
	//wire.Build 表示：用 resumeModuleSet 里登记的所有 provider，
	// 把 initializeResumeController 的返回值 *resume.ResumeController 拼出来。
	panic(wire.Build(resumeModuleSet))
}

// initializeInterviewController 生成面试模块控制器（/api 下 /interview/... 由控制器自行 Route）。
func initializeInterviewController(
	lg *zap.Logger,
	db *gorm.DB,
	rdb *redis.Client,
	oa *ai.OpenAIService,
	cfg *config.Config,
) *ivctl.InterviewController {
	panic(wire.Build(interviewModuleSet))
}

// initializeKnowledgeBaseController 知识库域（/api/knowledgebase/*），与 initializeResumeController 同形参以复用 StartDeps 注入。
func initializeKnowledgeBaseController(
	cfg *config.Config,
	lg *zap.Logger,
	db *gorm.DB,
	rdb *redis.Client,
	storeSvc *storage.StorageService,
	oa *ai.OpenAIService,
) *kbctl.KnowledgeBaseController {
	panic(wire.Build(knowledgeModuleSet))
}

// initializeRagChatController RAG 对话域（/api/rag-chat/*），依赖 Postgres + OpenAI（KB 问答链）。
func initializeRagChatController(cfg *config.Config, lg *zap.Logger, db *gorm.DB, oa *ai.OpenAIService) *ragctl.RagChatController {
	panic(wire.Build(ragModuleSet))
}
