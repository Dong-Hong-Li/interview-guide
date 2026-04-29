package main

import (
	"context"

	"interview-guide-go/internal/application/interview/service"
	"interview-guide-go/internal/application/resume/repository"
	"interview-guide-go/internal/config"
	"interview-guide-go/internal/infrastructure/ai"
	aiadapter "interview-guide-go/internal/infrastructure/ai/adapter"
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
		lg.Fatal(logmsg.MsgServerConfigNilFatal)
	}

	// ── 基础设施：对象存储 / Postgres / Redis / OpenAI ──
	storageService, err := storage.StartStorageService(ctx, cfg)
	if err != nil {
		lg.Error(logmsg.MsgStorageStartFailed, zap.Error(err))
		return nil, cleanup
	}

	// ── 数据库：Postgres ──
	postgresService, err := postgres.StartPostgresService(ctx, cfg)
	if err != nil {
		lg.Error(logmsg.MsgPostgresStartFailed, zap.Error(err))
		return nil, cleanup
	}
	cleanups = append(cleanups, func() { _ = postgresService.Close() })

	// ── 缓存：Redis ──
	redisService, err := redis.StartRedisService(ctx, cfg)
	if err != nil {
		lg.Error(logmsg.MsgRedisStartFailed, zap.Error(err))
		return nil, cleanup
	}
	cleanups = append(cleanups, func() { _ = redisService.Close() })

	// ── 大语言模型：简历/面试/知识库/RAG 对话 ──
	oaSvc, err := ai.NewOpenAIService(ctx, cfg)
	if err != nil {
		lg.Error(logmsg.MsgOpenAIStartFailed, zap.Error(err))
		return nil, cleanup
	}

	// ── 简历 / 面试模块：控制器由 wire 生成的 injector 装配 ──
	resumeController := initializeResumeController(cfg, lg, postgresService.DB, redisService.Client, storageService)

	// ── 面试模块：题目生成器、CreateInterview 用例、控制器由 wire 生成的 injector 装配 ──
	interviewController := initializeInterviewController(lg, postgresService.DB, redisService.Client, oaSvc, cfg)

	// ── 知识库：上传（存储/落库/向量化入队）由 wire 装配，
	knowledgeBaseController := initializeKnowledgeBaseController(cfg, lg, postgresService.DB, redisService.Client, storageService, oaSvc)

	// ── RAG 对话：会话 CRUD + messages/stream（复用 KnowledgeBaseQueryService RAG 链）。
	ragChatController := initializeRagChatController(cfg, lg, postgresService.DB, oaSvc)

	// ── 异步：简历分析 Redis Stream 消费者
	startResumeAnalyzeConsumerIfReady(ctx, cfg, lg, redisService, oaSvc, mapper.NewResumeMapper(postgresService.DB))

	// ── 异步：面试 LLM 评估（最后一题后 evaluate_status=PENDING + 入队；与主项目 Java 版能力对齐）
	startInterviewEvaluateConsumerIfReady(ctx, cfg, lg, redisService, oaSvc, postgresService)

	// ── 异步：知识库向量化（Upload 后入队 knowledge:vectorize:stream；消费者分块→Embedding→knowledge_base_chunks）
	startKnowledgeVectorizeConsumerIfReady(ctx, cfg, lg, redisService, postgresService, oaSvc)

	return []httpserver.RouteRegistrar{
		resumeController,
		interviewController,
		knowledgeBaseController,
		ragChatController,
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
		lg.Fatal("resume analyze consumer: prerequisites missing",
			zap.Bool(logmsg.FieldRedis, redisService != nil),
			zap.Bool(logmsg.FieldPostgres, resumeWriter != nil),
			zap.Bool(logmsg.FieldAPIKey, oaSvc != nil),
		)
	}
	o := cfg.Openai
	// 简历分析器
	grader := aiadapter.NewResumeGrader(
		oaSvc.Client(), o.AIModel,
		o.ResumeAIMaxRunes, o.ResumeAIMaxCompletionTokens, o.ResumeAITemperature, lg,
	)
	// 启动简历分析消费者
	redisstream.StartResumeAnalyzeConsumer(ctx, redisService.Client, resumeWriter, grader, lg)
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
		lg.Fatal("interview evaluate consumer: prerequisites missing",
			zap.Bool(logmsg.FieldRedis, redisService != nil),
			zap.Bool("postgres_db", pg != nil && pg.DB != nil),
			zap.Bool(logmsg.FieldAPIKey, oaSvc != nil),
		)
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
	ivEval, err := aiadapter.NewInterviewEvaluator(oaSvc.Client(), o.AIModel, o.ResumeAIMaxRunes, o.ResumeAIMaxCompletionTokens, o.ResumeAITemperature, 8, lg)
	if err != nil {
		lg.Fatal(logmsg.MsgInterviewEvaluatePromptsLoad, zap.Error(err))
	}
	// 启动面试评估消费者
	redisstream.StartInterviewEvaluateConsumer(ctx, redisService.Client, proc, ivEval, lg)

}

// startKnowledgeVectorizeConsumerIfReady 当 Redis、Postgres、OpenAI 就绪时启动知识库向量化消费者（Embedding 写入 PG）。
func startKnowledgeVectorizeConsumerIfReady(
	ctx context.Context,
	cfg *config.Config,
	lg *zap.Logger,
	redisService *redis.RedisService,
	pg *postgres.PostgresService,
	oaSvc *ai.OpenAIService,
) {
	if cfg == nil || redisService == nil || pg == nil || pg.DB == nil || oaSvc == nil {
		lg.Fatal("knowledge vectorize consumer: prerequisites missing",
			zap.Bool(logmsg.FieldRedis, redisService != nil),
			zap.Bool("postgres_db", pg != nil && pg.DB != nil),
			zap.Bool(logmsg.FieldAPIKey, oaSvc != nil),
		)
	}
	embedHTTP, err := ai.EmbeddingHTTPClient(cfg, oaSvc)
	if err != nil {
		lg.Fatal(logmsg.MsgKnowledgeEmbeddingClientFatal, zap.Error(err))
	}
	embedder := aiadapter.NewOpenAIKnowledgeEmbedder(embedHTTP, cfg.Openai, lg)
	chunker := aiadapter.NewOpenAIKnowledgeTextChunker(oaSvc.Client(), cfg, lg)
	redisstream.StartKnowledgeVectorizeConsumer(ctx, redisService.Client, mapper.NewKnowledgeBaseMapper(pg.DB), chunker, embedder, lg)
}
