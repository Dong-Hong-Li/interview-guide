package main

import (
	"context"

	"interview-guide-go/internal/application/interview/service"
	ragctl "interview-guide-go/internal/application/ragchat/controller"
	"interview-guide-go/internal/application/resume/repository"
	"interview-guide-go/internal/config"
	"interview-guide-go/internal/infrastructure/ai"
	"interview-guide-go/internal/infrastructure/postgres"
	"interview-guide-go/internal/infrastructure/postgres/mapper"
	"interview-guide-go/internal/infrastructure/redis"
	redisadapter "interview-guide-go/internal/infrastructure/redis/adapter"
	redisstream "interview-guide-go/internal/infrastructure/redis/stream"
	"interview-guide-go/internal/infrastructure/storage"
	httpserver "interview-guide-go/internal/interfaces/controller"
	"interview-guide-go/shared/logmsg"

	"go.uber.org/zap"
)

// StartDeps 装配所有适配器与应用服务，返回 HTTP API 控制器列表与 cleanup。
// 「Deps」= dependencies：此函数只做「启动外部资源 + 启动异步消费者 + 委托 wire 构造控制器」，
// 纯构造注入交给 wire_gen.go 的 initializeResumeController；详情见 wire.go。
func StartDeps(ctx context.Context, lg *zap.Logger, cfg *config.Config) ([]httpserver.RouteRegistrar, func()) {
	var cleanups []func()
	cleanup := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
	if cfg == nil {
		lg.Warn(logmsg.MsgServerConfigNilSkipWiring)
	}

	// ── 基础设施：对象存储 / Postgres / Redis / OpenAI 客户端 ──
	storageService, err := storage.StartStorageService(ctx, cfg)
	if err != nil {
		lg.Error(logmsg.MsgStorageStartFailed, zap.Error(err))
		return nil, cleanup
	}

	postgresService, err := postgres.StartPostgresService(ctx, cfg)
	if err != nil {
		lg.Error(logmsg.MsgPostgresStartFailed, zap.Error(err))
		return nil, cleanup
	}
	cleanups = append(cleanups, func() { _ = postgresService.Close() })

	redisService, err := redis.StartRedisService(ctx, cfg)
	if err != nil {
		lg.Error(logmsg.MsgRedisStartFailed, zap.Error(err))
		return nil, cleanup
	}
	cleanups = append(cleanups, func() { _ = redisService.Close() })

	oaSvc, err := ai.NewOpenAIService(ctx, cfg)
	if err != nil {
		lg.Error(logmsg.MsgOpenAIStartFailed, zap.Error(err))
		return nil, cleanup
	}

	// ── 简历 / 面试模块：控制器由 wire 生成的 injector 装配 ──
	resumeController := initializeResumeController(cfg, lg, postgresService.DB, redisService.Client, storageService)

	// ── 面试模块：题目生成器、CreateInterview 用例、控制器由 wire 生成的 injector 装配 ──
	interviewController := initializeInterviewController(lg, postgresService.DB, redisService.Client, oaSvc, cfg)

	// ── 知识库：上传（存储/落库/向量化入队）由 wire 装配，其余端点多为 501 占位 ──
	knowledgeBaseController := initializeKnowledgeBaseController(cfg, lg, postgresService.DB, redisService.Client, storageService)

	// ── 异步：简历分析 Redis Stream 消费者
	startResumeAnalyzeConsumerIfReady(ctx, cfg, lg, redisService, oaSvc, mapper.NewResumeMapper(postgresService.DB))

	// ── 异步：面试 LLM 评估（最后一题后 evaluate_status=PENDING + 入队；与主项目 take-notes 能力对齐）
	startInterviewEvaluateConsumerIfReady(ctx, cfg, lg, redisService, oaSvc, postgresService)

	// ── 异步：知识库向量化（Upload 后入队 knowledge:vectorize:stream，消费者分块并回写 vector_status / chunk_count）
	startKnowledgeVectorizeConsumerIfReady(ctx, lg, redisService, postgresService)

	return []httpserver.RouteRegistrar{
		resumeController,
		interviewController,
		knowledgeBaseController,
		&ragctl.RagChatController{},
	}, cleanup
}

// startResumeAnalyzeConsumerIfReady 当 Redis / Postgres / OpenAI 都就绪时启动简历分析消费者；否则仅记录跳过原因。
func startResumeAnalyzeConsumerIfReady(
	ctx context.Context,
	cfg *config.Config,
	lg *zap.Logger,
	redisService *redis.RedisService,
	oaSvc *ai.OpenAIService,
	resumeWriter repository.ResumeWriter,
) {
	if cfg == nil || redisService == nil || oaSvc == nil || resumeWriter == nil {
		lg.Info(logmsg.MsgResumeAIConsumerDisabled,
			zap.Bool(logmsg.FieldRedis, redisService != nil),
			zap.Bool(logmsg.FieldPostgres, resumeWriter != nil),
			zap.Bool(logmsg.FieldAPIKey, oaSvc != nil),
		)
		return
	}
	o := cfg.Openai
	// 简历分析器
	grader := ai.NewResumeGrader(
		oaSvc.Client(), o.AIModel,
		o.ResumeAIMaxRunes, o.ResumeAIMaxCompletionTokens, o.ResumeAITemperature, lg,
	)
	// 启动简历分析消费者
	redisstream.StartResumeAnalyzeConsumer(ctx, redisService.Client, resumeWriter, grader, lg)
	lg.Info(logmsg.MsgResumeAIConsumerEnabled,
		zap.String(logmsg.FieldOpenAIBaseURL, o.OpenAIBaseURL),
		zap.String(logmsg.FieldModel, o.AIModel),
	)
}

// startInterviewEvaluateConsumerIfReady 当 Redis / Postgres / OpenAI 就绪时启动「面试整卷 LLM 评估」消费者；否则仅记录并跳过（与主项目 deps 条件一致）。
func startInterviewEvaluateConsumerIfReady(
	ctx context.Context,
	cfg *config.Config,
	lg *zap.Logger,
	redisService *redis.RedisService,
	oaSvc *ai.OpenAIService,
	pg *postgres.PostgresService,
) {
	if cfg == nil || redisService == nil || oaSvc == nil || pg == nil || pg.DB == nil {
		return
	}
	o := cfg.Openai
	// 面试会话 mapper
	im := mapper.NewInterviewMapper(pg.DB)
	// 会话缓存
	sc := redisadapter.NewSessionCache(redisService.Client)
	// 简历 mapper
	rm := mapper.NewResumeMapper(pg.DB)
	// 评估处理器
	proc := service.NewEvaluateProcessor(im, sc, rm)
	// 面试评估器
	ivEval, err := ai.NewInterviewEvaluator(oaSvc.Client(), o.AIModel, o.ResumeAIMaxRunes, o.ResumeAIMaxCompletionTokens, o.ResumeAITemperature, 8, lg)
	if err != nil {
		lg.Warn(logmsg.MsgInterviewEvaluatePromptsLoad, zap.Error(err))
		return
	}
	// 启动面试评估消费者
	redisstream.StartInterviewEvaluateConsumer(ctx, redisService.Client, proc, ivEval, lg)

	lg.Info(logmsg.MsgInterviewEvaluateConsumerEnabled,
		zap.String(logmsg.FieldOpenAIBaseURL, o.OpenAIBaseURL),
		zap.String(logmsg.FieldModel, o.AIModel),
	)
}

// startKnowledgeVectorizeConsumerIfReady 当 Redis 与 Postgres 就绪时启动知识库向量化消费者（不依赖 OpenAI，与 Upload 入队成对）。
func startKnowledgeVectorizeConsumerIfReady(
	ctx context.Context,
	lg *zap.Logger,
	redisService *redis.RedisService,
	pg *postgres.PostgresService,
) {
	if redisService == nil || pg == nil || pg.DB == nil {
		return
	}
	redisstream.StartKnowledgeVectorizeConsumer(ctx, redisService.Client, mapper.NewKnowledgeBaseMapper(pg.DB), lg)
	lg.Info(logmsg.MsgKnowledgeVectorizeConsumerEnabled,
		zap.String(logmsg.FieldRedis, redisService.Client.String()),
		zap.String(logmsg.FieldPostgres, pg.DB.Name()),
	)
}
