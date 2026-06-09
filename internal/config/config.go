package config

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
