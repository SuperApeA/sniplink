package config

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// 从项目根目录开始
	configDir  = "./configs"
	configName = "config.yaml"
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
	BaseURL           string        `mapstructure:"base_url"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	RateLimitRps      float64       `mapstructure:"rate_limit_rps"`
	RateLimitBurst    int           `mapstructure:"rate_limit_burst"`
	LogMod            string        `mapstructure:"log_mod"`
}

// MySQLConfig MySQL配置
type MySQLConfig struct {
	DSN string `mapstructure:"mysql_dsn"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr          string        `mapstructure:"redis_addr"`
	Password      string        `mapstructure:"redis_password"`
	DB            int           `mapstructure:"redis_db"`
	PingTimeout   time.Duration `mapstructure:"redis_ping_timeout"`
	IdgenKey      string        `mapstructure:"redis_idgen_key"`
	CacheTTL      time.Duration `mapstructure:"cache_ttl"`       // 正常链接的缓存时长
	EmptyCacheTTL time.Duration `mapstructure:"empty_cache_ttl"` // 不存在链接的空值缓存时长
}

func Load() *Config {
	v := viper.New()

	v.SetConfigName(strings.TrimSuffix(configName, filepath.Ext(configName)))
	v.SetConfigType(strings.TrimPrefix(filepath.Ext(configName), "."))
	// TODO: 部署环境这里需要对应
	v.AddConfigPath(configDir)

	// 1. 直接读取 config.yaml
	if err := v.ReadInConfig(); err != nil {
		// 失败打日志后退出程序
		log.Fatalf("解析 config.yaml 失败，请检查文件是否存在: %v", err)
	}
	log.Printf("成功加载配置文件: %s", v.ConfigFileUsed())

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
	log.Printf("配置文件:\n %+v\n", cfg)

	return &cfg
}
