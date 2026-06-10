package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// 项目配置
type Config struct {
	// === 应用 ===
	Port      string // 监听端口，默认 "8080"
	Mode      string // 运行模式："debug" 或 "release"
	SecretKey string // JWT 签名密钥
	AdminUID  string // 管理员 UID

	// === 数据库 ===
	DB    DatabaseConfig
	Redis RedisConfig

	// === 并发 ===
	GlobalRateLimitCalls  int // 全局限流：每 IP 每窗口允许请求数
	GlobalRateLimitPeriod int // 限流窗口：秒

	// === AI ===
	OpenRouterKey     string
	OpenRouterBaseURL string
	GeminiKey         string

	// === 存储 ===
	DataDir       string
	MaxUploadSize int64 // 字节

	// === 邮件 ===
	Mail MailConfig

	// === 第三方 API ===
	TwelveDataKey string
}

// 数据库配置
type DatabaseConfig struct {
	DSN          string // PostgreSQL 连接字符串
	MaxOpenConns int    // 最大连接数
	MaxIdleConns int    // 最大空闲连接数
}

type RedisConfig struct {
	Addr     string // host:port
	Password string
	DB       int
}

type MailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatalln("未找到 .env 文件，使用系统环境变量")
	}
	return &Config{
		Port:      getEnv("PORT", "8080"),
		Mode:      getEnv("MODE", "dev"),
		SecretKey: getEnv("SECRET_KEY", ""),
		AdminUID:  getEnv("ADMIN_UID", ""),

		// 数据库
		DB: DatabaseConfig{
			DSN:          getEnv("DATABASE_DSN", ""),
			MaxOpenConns: getEnvInt("DATABASE_MAX_OPEN_CONNS", 20),
			MaxIdleConns: getEnvInt("DATABASE_MAX_IDLE_CONNS", 10),
		},

		// redis
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},

		// 限流
		GlobalRateLimitCalls:  getEnvInt("GLOBAL_RATE_LIMIT_CALLS", 1200),
		GlobalRateLimitPeriod: getEnvInt("GLOBAL_RATE_LIMIT_PERIOD", 60),

		// 邮件
		Mail: MailConfig{
			Host:     getEnv("MAIL_HOST", "localhost"),
			Port:     getEnvInt("MAIL_PORT", 25),
			Username: getEnv("MAIL_USERNAME", ""),
			Password: getEnv("MAIL_PASSWORD", ""),
			From:     getEnv("MAIL_FROM", ""),
		},

		// AI相关
		OpenRouterKey:     getEnv("OPEN_ROUTER_KEY", ""),
		OpenRouterBaseURL: getEnv("OPEN_ROUTER_BASE_URL", ""),
		GeminiKey:         getEnv("GEMINI_KEY", ""),

		DataDir:       getEnv("DATA_DIR", ""),
		MaxUploadSize: int64(getEnvInt("MAX_UPLOAD_SIZE", 5<<20)), // 默认 5MB

		TwelveDataKey: getEnv("TwelveDataKey", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}
