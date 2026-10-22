package idgen

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Generator 基于 Redis INCR 的全局发号器。
// 注意:生产环境务必给 Redis 开启 AOF/RDB 持久化,
// 否则重启后 ID 回退,可能撞上已存在的短码(DB 唯一索引会兜底报错并重试)。
type Generator struct {
	rdb *redis.Client
	key string
}

func New(rdb *redis.Client, key string) *Generator {
	return &Generator{rdb: rdb, key: key}
}

func (g *Generator) Next(ctx context.Context) (uint64, error) {
	v, err := g.rdb.Incr(ctx, g.key).Result()
	if err != nil {
		return 0, err
	}
	return uint64(v), nil
}
