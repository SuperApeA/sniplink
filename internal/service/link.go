package service

import (
	"context"
	"errors"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"sniplink/internal/cache"
	"sniplink/internal/config"
	"sniplink/internal/idgen"
	"sniplink/internal/model"
	"sniplink/internal/repository"
	"sniplink/internal/shortcode"
)

var (
	ErrNotFound    = errors.New("short link not found")
	ErrInvalidURL  = errors.New("invalid url")
	ErrInvalidCode = errors.New("invalid custom code")
	ErrCodeTaken   = errors.New("custom code already taken")
	ErrGenFailed   = errors.New("failed to generate unique code")
)

var codeRe = regexp.MustCompile(`^[0-9A-Za-z_-]{1,16}$`)

// 保留字,避免与 API / 静态资源路径冲突
var reserved = map[string]struct{}{
	"api": {}, "healthz": {}, "favicon.ico": {}, "robots.txt": {}, "static": {},
}

type LinkService struct {
	repo  *repository.LinkRepo
	cache *cache.LinkCache
	gen   *idgen.Generator
	cfg   *config.Config

	stats  chan string
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewLinkService(repo *repository.LinkRepo, c *cache.LinkCache, gen *idgen.Generator, cfg *config.Config) *LinkService {
	s := &LinkService{
		repo:   repo,
		cache:  c,
		gen:    gen,
		cfg:    cfg,
		stats:  make(chan string, 4096),
		stopCh: make(chan struct{}),
	}
	s.wg.Add(1)
	go s.statsWorker()
	return s
}

// Shorten 创建短链。customCode 为空则自动生成;ttl>0 表示过期时间。
func (s *LinkService) Shorten(ctx context.Context, rawURL, customCode string, ttl time.Duration) (*model.Link, error) {
	target, err := normalizeURL(rawURL)
	if err != nil {
		return nil, err
	}

	link := &model.Link{URL: target}
	if ttl > 0 {
		exp := time.Now().Add(ttl)
		link.ExpiresAt = &exp
	}

	if customCode != "" {
		if !validCustomCode(customCode) {
			return nil, ErrInvalidCode
		}
		link.Code = customCode
		if err := s.repo.Create(ctx, link); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, ErrCodeTaken
			}
			return nil, err
		}
		return link, nil
	}

	// 自动生成:Redis 发号 + Base62,DB 唯一索引兜底重试
	for i := 0; i < 5; i++ {
		id, err := s.gen.Next(ctx)
		if err != nil {
			return nil, err
		}
		link.Code = shortcode.Encode(id)
		link.ID = 0
		err = s.repo.Create(ctx, link)
		if err == nil {
			return link, nil
		}
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, err
		}
	}
	return nil, ErrGenFailed
}

// Resolve 解析短码,返回长链接。
func (s *LinkService) Resolve(ctx context.Context, code string) (string, error) {
	// 1. 查缓存
	if target, hit, err := s.cache.Get(ctx, code); err == nil && hit {
		if target == "" {
			return "", ErrNotFound
		}
		s.recordClick(code)
		return target, nil
	}

	// 2. 回源 DB
	link, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = s.cache.SetNotFound(ctx, code, s.cfg.Redis.EmptyCacheTTL)
			return "", ErrNotFound
		}
		return "", err
	}

	now := time.Now()
	if link.IsExpired(now) {
		_ = s.cache.SetNotFound(ctx, code, s.cfg.Redis.EmptyCacheTTL)
		return "", ErrNotFound
	}

	// 3. 回填缓存(过期时间早于缓存 TTL 时按剩余时间设置)
	ttl := s.cfg.Redis.CacheTTL
	if link.ExpiresAt != nil {
		if remain := link.ExpiresAt.Sub(now); remain < ttl {
			ttl = remain
		}
	}
	_ = s.cache.Set(ctx, code, link.URL, ttl)

	s.recordClick(code)
	return link.URL, nil
}

func (s *LinkService) Get(ctx context.Context, code string) (*model.Link, error) {
	l, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return l, nil
}

func (s *LinkService) Delete(ctx context.Context, code string) error {
	if err := s.repo.DeleteByCode(ctx, code); err != nil {
		return err
	}
	_ = s.cache.Del(ctx, code)
	return nil
}

func (s *LinkService) Close() {
	close(s.stopCh)
	s.wg.Wait()
}

// ---------- 访问统计(异步批量) ----------

func (s *LinkService) recordClick(code string) {
	select {
	case s.stats <- code:
	default:
		// 队列满则丢弃,跳转链路不能因此变慢
	}
}

func (s *LinkService) statsWorker() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	buf := make(map[string]int64, 256)

	flush := func() {
		if len(buf) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		for code, n := range buf {
			if err := s.repo.IncrClick(ctx, code, n); err != nil {
				log.Printf("incr click %s: %v", code, err)
			}
		}
		cancel()
		buf = make(map[string]int64, 256)
	}

	for {
		select {
		case code := <-s.stats:
			buf[code]++
			if len(buf) >= 500 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.stopCh:
			// 排空 channel 后刷库
			for {
				select {
				case code := <-s.stats:
					buf[code]++
				default:
					flush()
					return
				}
			}
		}
	}
}

// ---------- 工具函数 ----------

func normalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 2048 {
		return "", ErrInvalidURL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", ErrInvalidURL
	}
	if u.Host == "" {
		return "", ErrInvalidURL
	}
	return u.String(), nil
}

func validCustomCode(code string) bool {
	if !codeRe.MatchString(code) {
		return false
	}
	if _, ok := reserved[strings.ToLower(code)]; ok {
		return false
	}
	return true
}
