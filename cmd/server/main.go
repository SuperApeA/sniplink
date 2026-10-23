package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sniplink/internal/cache"
	"sniplink/internal/config"
	"sniplink/internal/handler"
	"sniplink/internal/idgen"
	"sniplink/internal/model"
	"sniplink/internal/repository"
	"sniplink/internal/router"
	"sniplink/internal/service"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化基础设施
	db := initMySQL(cfg)
	rdb := initRedis(cfg)

	// 3. 组装依赖
	repo := repository.NewLinkRepo(db)
	linkCache := cache.NewLinkCache(rdb)
	gen := idgen.New(rdb, cfg.Redis.IdgenKey)
	svc := service.NewLinkService(repo, linkCache, gen, cfg)
	defer svc.Close() // 确保异步统计的 worker 能安全退出

	handle := handler.NewLinkHandler(svc, cfg)
	ginRouter := router.Setup(cfg, handle)

	// 4. 构造 HTTP Server
	srv := &http.Server{
		Addr:              cfg.Sniplink.Addr,
		Handler:           ginRouter,
		ReadHeaderTimeout: cfg.Sniplink.ReadHeaderTimeout,
	}

	// 5. 后台启动监听
	go runServer(srv, cfg.Sniplink.Addr)

	// 6. 阻塞等待退出信号，并执行优雅关闭
	gracefulShutdown(srv, db, rdb)

	log.Println("bye")
}

// initMySQL 初始化数据库连接
func initMySQL(cfg *config.Config) *gorm.DB {
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true,
	})
	if err != nil {
		log.Fatalf("connect mysql failed: %v", err)
	}

	// TODO: 生产环境应删掉 AutoMigrate，改用 golang-migrate 等工具
	if err := db.AutoMigrate(&model.Link{}); err != nil {
		log.Fatalf("migrate failed: %v", err)
	}
	return db
}

// initRedis 初始化 Redis 客户端
func initRedis(cfg *config.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
		PoolTimeout:  cfg.Redis.PoolTimeout,
	})

	pingCtx, pingCancel := context.WithTimeout(context.Background(), cfg.Redis.PingTimeout)
	defer pingCancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		log.Fatalf("connect redis failed: %v", err)
	}
	return rdb
}

// runServer 在独立协程中启动 HTTP 服务
func runServer(srv *http.Server, addr string) {
	log.Printf("sniplink server listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
}

// gracefulShutdown 处理系统信号并优雅关闭资源
func gracefulShutdown(srv *http.Server, db *gorm.DB, rdb *redis.Client) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	// 1. 优雅关闭 HTTP 服务 (给正在处理的请求 10 秒钟)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("wait shutdown timeout, close forced")
		}
		log.Printf("shutdown error: %v", err)
	}

	// 2. 关闭底层连接池 (注意顺序：先关 HTTP 再关 DB/Redis)
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	_ = rdb.Close()
}
