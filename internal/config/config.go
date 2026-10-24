package config

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// 从项目根目录开始
	configFile = "./configs/config.yaml"
)

// Config 采用嵌套结构体，完全对齐 YAML 的层级
type Config struct {
	Sniplink SniplinkConfig `mapstructure:"sniplink"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

// SniplinkConfig Sniplink的配置
type SniplinkConfig struct {
	Addr              string        `mapstructure:"addr"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	RateLimitRps      float64       `mapstructure:"rate_limit_rps"`
	RateLimitBurst    int           `mapstructure:"rate_limit_burst"`
	LogMod            string        `mapstructure:"log_mod"`
}

// MySQLConfig MySQL配置
type MySQLConfig struct {
	DSN string `mapstructure:"dsn"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr          string        `mapstructure:"addr"`
	Password      string        `mapstructure:"password"`
	DB            int           `mapstructure:"db"`
	PingTimeout   time.Duration `mapstructure:"ping_timeout"`
	IdgenKey      string        `mapstructure:"idgen_key"`
	CacheTTL      time.Duration `mapstructure:"cache_ttl"`
	EmptyCacheTTL time.Duration `mapstructure:"empty_cache_ttl"`

	// --- 以下是高阶连接池与超时配置 ---
	PoolSize     int           `mapstructure:"pool_size"`      // 最大连接数
	MinIdleConns int           `mapstructure:"min_idle_conns"` // 最小空闲连接数
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`   // 建立连接超时
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`   // 读超时
	WriteTimeout time.Duration `mapstructure:"write_timeout"`  // 写超时
	PoolTimeout  time.Duration `mapstructure:"pool_timeout"`   // 连接池满时的等待超时
}

func Load() *Config {
	v := viper.New()
	v.SetConfigFile(configFile)

	// 1. 直接读取 config.yaml
	if err := v.ReadInConfig(); err != nil {
		// 失败打日志后退出程序
		log.Fatalf("解析 config.yaml 失败，请检查文件是否存在: %v", err)
	}
	log.Printf("success load config: %s", v.ConfigFileUsed())

	// 2. 开启环境变量覆盖（支持 Docker/K8s 部署时动态修改参数）
	v.AutomaticEnv()
	// 将 . 替换为 _ (例如 sniplink.addr 对应 SNIPLINK_ADDR)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 3. 映射到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		// 失败打日志后退出程序
		log.Fatalf("解析配置失败: %v", err)
	}
	log.Printf("config loaded successfully:")
	log.Printf("  - Server Addr : %s", cfg.Sniplink.Addr)
	log.Printf("  - MySQL DSN   : %s", regexp.MustCompile(`:([^@]+)@`).ReplaceAllString(cfg.MySQL.DSN, ":***@"))
	log.Printf("  - Redis Addr  : %s", cfg.Redis.Addr)

	return &cfg
}
