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
	// 1.加载配置文件，包括MySQL，Redis，sniplink服务
	cfg := config.Load()

	// 2.设置MySQL
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true, // 开启后重复键会返回 gorm.ErrDuplicatedKey
	})
	if err != nil {
		log.Fatalf("connect mysql failed: %v", err)
	}
	// TODO:建表操作，生产环境这里需要删掉，应该在构建环境时提前建表
	if err := db.AutoMigrate(&model.Link{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// 3.设置Redis
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

	// 4.组装数据库
	repo := repository.NewLinkRepo(db)
	linkCache := cache.NewLinkCache(rdb)
	gen := idgen.New(rdb, cfg.Redis.IdgenKey)
	svc := service.NewLinkService(repo, linkCache, gen, cfg)
	defer svc.Close()

	handle := handler.NewLinkHandler(svc, cfg)

	// 5.设置路由，注册接口
	ginRouter := router.Setup(cfg, handle)

	srv := &http.Server{
		Addr:              cfg.Sniplink.Addr,
		Handler:           ginRouter,
		ReadHeaderTimeout: cfg.Sniplink.ReadHeaderTimeout,
	}

	// 6.开启监听
	go func() {
		log.Printf("sniplink server listening on %s", cfg.Sniplink.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	// 7.优雅关闭，收到关闭信号后执行关闭，主程序会一直阻塞在这里
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	// 关闭http服务、断开MySQL和Redis连接
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		// 判断是否是因为超时导致的强制关闭
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("wait shutdown timeout, close forced")
		}
		// 无论是不是超时，都打印一下真实错误，方便排查
		log.Printf("shutdown: %v", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	_ = rdb.Close()
	log.Println("bye")
}
