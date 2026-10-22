package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const emptyMark = "\x00" // 空值标记,防止缓存穿透

type LinkCache struct {
	rdb    *redis.Client
	prefix string
}

func NewLinkCache(rdb *redis.Client) *LinkCache {
	return &LinkCache{rdb: rdb, prefix: "sl:link:"}
}

func (c *LinkCache) key(code string) string { return c.prefix + code }

// Get 返回 (url, hit, err)。hit=true 且 url=="" 表示命中了"不存在"的缓存。
func (c *LinkCache) Get(ctx context.Context, code string) (string, bool, error) {
	v, err := c.rdb.Get(ctx, c.key(code)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if v == emptyMark {
		return "", true, nil
	}
	return v, true, nil
}

func (c *LinkCache) Set(ctx context.Context, code, url string, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.key(code), url, ttl).Err()
}

func (c *LinkCache) SetNotFound(ctx context.Context, code string, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.key(code), emptyMark, ttl).Err()
}

func (c *LinkCache) Del(ctx context.Context, code string) error {
	return c.rdb.Del(ctx, c.key(code)).Err()
}
