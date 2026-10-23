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

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sniplink/internal/cache"
	"sniplink/internal/config"
	"sniplink/internal/handler"
	"sniplink/internal/idgen"
	"sniplink/internal/middleware"
	"sniplink/internal/model"
	"sniplink/internal/repository"
	"sniplink/internal/service"
)

func main() {
	cfg := config.Load()

	// MySQL
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true, // 开启后重复键会返回 gorm.ErrDuplicatedKey
	})
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	if err := db.AutoMigrate(&model.Link{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	pingCtx, pingCancel := context.WithTimeout(context.Background(), cfg.Redis.PingTimeout)
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	pingCancel()

	// 组装
	repo := repository.NewLinkRepo(db)
	linkCache := cache.NewLinkCache(rdb)
	gen := idgen.New(rdb, cfg.Redis.IdgenKey)
	svc := service.NewLinkService(repo, linkCache, gen, cfg)
	defer svc.Close()

	h := handler.NewLinkHandler(svc, cfg)

	// 路由
	gin.SetMode(cfg.Sniplink.LogMod)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1", middleware.RateLimit(cfg.Sniplink.RateLimitRps, cfg.Sniplink.RateLimitBurst))
	{
		api.POST("/links", h.Create)
		api.GET("/links/:code", h.Get)
		api.DELETE("/links/:code", h.Delete)
	}

	// 所有未匹配的 GET 请求按短码处理
	r.NoRoute(h.HandleShort)

	srv := &http.Server{
		Addr:              cfg.Sniplink.Addr,
		Handler:           r,
		ReadHeaderTimeout: cfg.Sniplink.ReadHeaderTimeout,
	}

	go func() {
		log.Printf("sniplink server listening on %s", cfg.Sniplink.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	_ = rdb.Close()
	log.Println("bye")
}
