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
	resume "interview-guide-go/internal/application/resume/controller"
	resumerepo "interview-guide-go/internal/application/resume/repository"
	resumesvc "interview-guide-go/internal/application/resume/service"
	"interview-guide-go/internal/config"
	"interview-guide-go/internal/infrastructure/ai"
	aiq "interview-guide-go/internal/infrastructure/ai/adapter"
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

// provideMaxResumeUploadBytes 抽取 cfg 字段给 ResumeUploadService 用，避免 wire 直接把 cfg.Xxx 当 int64 provider。
func provideMaxResumeUploadBytes(cfg *config.Config) int64 {
	return cfg.MaxResumeUploadBytes
}

// provideInterviewSessionCache 将 *redisadapter.SessionCache 以接口形式注入，避免与 NewSessionCache 的重复 interface 绑定。
func provideInterviewSessionCache(rdb *redis.Client) ivrepo.InterviewSessionCache {
	return redisadapter.NewSessionCache(rdb)
}

// provideInterviewQuestionGenerator OpenAI 客户端未就绪时退回 Stub，与 deps 中 oa 启动失败时一致。
func provideInterviewQuestionGenerator(oa *ai.OpenAIService, cfg *config.Config, lg *zap.Logger) ivrepo.InterviewQuestionGenerator {
	if oa == nil {
		return ai.NewStubInterviewQuestionGenerator()
	}
	return aiq.NewOpenAIInterviewQuestionGenerator(oa, cfg, lg)
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
