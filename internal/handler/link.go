package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"sniplink/internal/config"
	"sniplink/internal/service"
)

type LinkHandler struct {
	svc *service.LinkService
	cfg *config.Config
}

func NewLinkHandler(svc *service.LinkService, cfg *config.Config) *LinkHandler {
	return &LinkHandler{svc: svc, cfg: cfg}
}

type CreateLinkRequest struct {
	URL        string `json:"url" binding:"required"`
	CustomCode string `json:"custom_code"`
	ExpireIn   int64  `json:"expire_in"` // 秒,0 表示永久
}

type CreateLinkResponse struct {
	Code      string     `json:"code"`
	ShortURL  string     `json:"short_url"`
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (h *LinkHandler) Create(c *gin.Context) {
	var req CreateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var ttl time.Duration
	if req.ExpireIn > 0 {
		if req.ExpireIn > int64((365*24*time.Hour)/time.Second) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expire_in too large"})
			return
		}
		ttl = time.Duration(req.ExpireIn) * time.Second
	}

	link, err := h.svc.Shorten(c.Request.Context(), req.URL, req.CustomCode, ttl)
	if err != nil {
		status, msg := mapErr(err)
		c.JSON(status, gin.H{"error": msg})
		return
	}

	c.JSON(http.StatusCreated, CreateLinkResponse{
		Code:      link.Code,
		ShortURL:  strings.TrimRight(h.cfg.Sniplink.BaseURL, "/") + "/" + link.Code,
		URL:       link.URL,
		ExpiresAt: link.ExpiresAt,
	})
}

func (h *LinkHandler) Get(c *gin.Context) {
	link, err := h.svc.Get(c.Request.Context(), c.Param("code"))
	if err != nil {
		status, msg := mapErr(err)
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":        link.Code,
		"url":         link.URL,
		"click_count": link.ClickCount,
		"created_at":  link.CreatedAt,
		"expires_at":  link.ExpiresAt,
	})
}

func (h *LinkHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("code")); err != nil {
		status, msg := mapErr(err)
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.Status(http.StatusNoContent)
}

// HandleShort 作为 NoRoute 兜底处理短链跳转。
// 好处:API 路由全部静态注册,不会与 "/:code" 通配符产生 Gin 路由冲突。
func (h *LinkHandler) HandleShort(c *gin.Context) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		c.String(http.StatusNotFound, "not found")
		return
	}

	path := strings.Trim(c.Request.URL.Path, "/")
	if path == "" || strings.Contains(path, "/") || len(path) > 16 {
		c.String(http.StatusNotFound, "not found")
		return
	}

	target, err := h.svc.Resolve(c.Request.Context(), path)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.String(http.StatusNotFound, "link not found or expired")
			return
		}
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	c.Redirect(http.StatusFound, target) // 302
}

// TODO: 错误返回不直观，后续有前端页面后，这里要返回错误码
func mapErr(err error) (int, string) {
	switch {
	case errors.Is(err, service.ErrInvalidURL):
		return http.StatusBadRequest, "invalid url, must be http/https"
	case errors.Is(err, service.ErrInvalidCode):
		return http.StatusBadRequest, "invalid custom_code, must match [0-9A-Za-z_-]{1,16}"
	case errors.Is(err, service.ErrCodeTaken):
		return http.StatusConflict, "custom code already taken"
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound, "link not found"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}
