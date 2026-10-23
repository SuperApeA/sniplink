package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sniplink/internal/config"
	"sniplink/internal/handler"
	"sniplink/internal/middleware"
)

// Setup 初始化并配置 Gin 引擎
func Setup(cfg *config.Config, h *handler.LinkHandler) *gin.Engine {
	gin.SetMode(cfg.Sniplink.LogMod)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 健康检查
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API 路由组
	api := r.Group("/api/v1", middleware.RateLimit(cfg.Sniplink.RateLimitRps, cfg.Sniplink.RateLimitBurst))
	{
		api.POST("/links", h.Create)
		api.GET("/links/:code", h.Get)
		api.DELETE("/links/:code", h.Delete)
	}

	// 兜底路由：短链跳转，所有未匹配的 GET 请求按短码处理
	r.NoRoute(h.HandleShort)

	return r
}
