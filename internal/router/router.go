package router

import (
	"net/http"

	"github.com/gin-contrib/cors"
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
	// 虽然当前前后端同源部署无需 CORS，但为了兼容本地前后端分离开发和未来 API 开放，保留跨域中间件
	r.Use(cors.Default())

	// 1. 托管前端静态资源 (必须在 NoRoute 之前)
	setStaticFile(r)

	// 2.健康检查
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 3.API 路由组，路由注册
	api := r.Group("/api/v1", middleware.RateLimit(cfg.Sniplink.RateLimitRps, cfg.Sniplink.RateLimitBurst))
	{
		api.POST("/links", h.Create)
		api.GET("/links/:code", h.Get)
		api.DELETE("/links/:code", h.Delete)
	}

	// 4.兜底路由：短链跳转，所有未匹配的 GET 请求按短码处理
	r.NoRoute(h.HandleShort)

	return r
}

// 设置静态资源
func setStaticFile(r *gin.Engine) {
	// 将根路径 "/" 映射到 web 目录下的 index.html
	r.StaticFile("/", "./web/index.html")
	// 浏览器标签页展示的小图标
	r.StaticFile("/favicon.ico", "./web/favicon.ico")
}
