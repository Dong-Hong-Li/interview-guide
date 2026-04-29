package main

import (
	"context"
	"fmt"
	"interview-guide-go/internal/config"
	httpserver "interview-guide-go/internal/interfaces/controller"
	"interview-guide-go/internal/logger"
	"interview-guide-go/shared/logmsg"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// 初始化 logger =====
	lg, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("%s: %v", logmsg.MsgLoggerInitFatal, err)
	}
	defer func() { _ = lg.Sync() }()
	// =========================

	// 加载环境变量（全量快照见 config.LogStartup；敏感字段已脱敏）
	cfg := config.LoadEnvironmentVariables()
	cfg.LogStartup(lg)
	// =========================

	// newDeps 装配所有适配器与应用服务，返回 HTTP API 控制器列表与 cleanup。
	apiControllers, cleanupDeps := StartDeps(context.Background(), lg, cfg)
	defer cleanupDeps()
	// 注册 HTTP 路由
	httpHandlers := httpserver.RegistrationRoutes(lg, cfg, apiControllers)
	// 创建 HTTP 服务器
	httpServer := newHttpServer(cfg, httpHandlers)

	// 启动 HTTP 服务器
	go func() {
		lg.Info(logmsg.MsgServerListening, zap.String(logmsg.FieldAddr, fmt.Sprintf("http://%s:%d", cfg.Server.ServerHost, cfg.Server.ServerPort)))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			lg.Fatal(logmsg.MsgListenFatal, zap.Error(err))
		}
	}()

	// 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	// 关闭 HTTP 服务器
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		lg.Warn(logmsg.MsgShutdownWarn, zap.Error(err))
	}
	lg.Info(logmsg.MsgServerStopped)
}

// startHttpServer 创建 HTTP 服务器
func newHttpServer(cfg *config.Config, httpHandlers http.Handler) *http.Server {
	addr := fmt.Sprintf("%s:%d", cfg.Server.ServerHost, cfg.Server.ServerPort)
	readTO := time.Duration(cfg.Server.ServerReadTimeoutSec) * time.Second
	return &http.Server{
		Addr:              addr,
		Handler:           httpHandlers,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       readTO,
		WriteTimeout:      15 * time.Minute,
		IdleTimeout:       120 * time.Second,
	}
}
