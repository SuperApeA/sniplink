package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr          string
	BaseURL       string // 生成的短链前缀,如 https://s.example.com
	MySQLDSN      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	CacheTTL      time.Duration // 正常链接的缓存时长
	EmptyCacheTTL time.Duration // 不存在链接的空值缓存时长
}

func Load() *Config {
	return &Config{
		Addr:          getEnv("ADDR", ":8080"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		MySQLDSN:      getEnv("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/sniplink?charset=utf8mb4&parseTime=True&loc=Local"),
		RedisAddr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		CacheTTL:      getEnvDuration("CACHE_TTL", time.Hour),
		EmptyCacheTTL: 60 * time.Second,
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getEnvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
