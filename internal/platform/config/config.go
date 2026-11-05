package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment  string
	HTTPAddr     string
	UploadDir    string
	MediaBaseURL string
	// WebRevalidateURL/Token 指向 SvelteKit 的页面缓存失效端点；留空则跳过。
	WebRevalidateURL   string
	WebRevalidateToken string
	// SMTP 留空时使用日志发送器，仅在本地开发接受。
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string
	SMTPMode     string
	// EmailCodePepper 参与验证码哈希，生产必须设置。
	EmailCodePepper    string
	DatabaseDSN        string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	AllowedOrigins     []string
	TrustedProxies     []string
	SessionIdleTTL     time.Duration
	SessionAbsoluteTTL time.Duration
	CookieSecure       bool
	ShutdownTimeout    time.Duration
}

func Load() (Config, error) {
	environment := env("VOIDXO_ENV", "development")
	shutdownTimeout, err := durationEnv("VOIDXO_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	sessionIdleTTL, err := durationEnv("VOIDXO_SESSION_IDLE_TTL", 7*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	sessionAbsoluteTTL, err := durationEnv("VOIDXO_SESSION_ABSOLUTE_TTL", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	redisDB, err := intEnv("VOIDXO_REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}
	cookieSecure, err := boolEnv("VOIDXO_COOKIE_SECURE", environment != "development" && environment != "test")
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		Environment:        environment,
		HTTPAddr:           env("VOIDXO_HTTP_ADDR", "127.0.0.1:8080"),
		UploadDir:          env("VOIDXO_UPLOAD_DIR", "./uploads"),
		MediaBaseURL:       env("VOIDXO_MEDIA_BASE_URL", "/media"),
		SMTPHost:           strings.TrimSpace(os.Getenv("VOIDXO_SMTP_HOST")),
		SMTPPort:           envInt("VOIDXO_SMTP_PORT", 587),
		SMTPUsername:       os.Getenv("VOIDXO_SMTP_USERNAME"),
		SMTPPassword:       os.Getenv("VOIDXO_SMTP_PASSWORD"),
		SMTPFrom:           strings.TrimSpace(os.Getenv("VOIDXO_SMTP_FROM")),
		SMTPFromName:       env("VOIDXO_SMTP_FROM_NAME", "VOIDXO"),
		SMTPMode:           env("VOIDXO_SMTP_MODE", "starttls"),
		EmailCodePepper:    os.Getenv("VOIDXO_EMAIL_CODE_PEPPER"),
		WebRevalidateURL:   strings.TrimSpace(os.Getenv("VOIDXO_WEB_REVALIDATE_URL")),
		WebRevalidateToken: strings.TrimSpace(os.Getenv("VOIDXO_WEB_REVALIDATE_TOKEN")),
		DatabaseDSN:        strings.TrimSpace(os.Getenv("VOIDXO_DATABASE_DSN")),
		RedisAddr:          env("VOIDXO_REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:      os.Getenv("VOIDXO_REDIS_PASSWORD"),
		RedisDB:            redisDB,
		AllowedOrigins:     splitCSV(os.Getenv("VOIDXO_ALLOWED_ORIGINS")),
		TrustedProxies:     splitCSV(env("VOIDXO_TRUSTED_PROXIES", "127.0.0.1/32,::1/128")),
		SessionIdleTTL:     sessionIdleTTL,
		SessionAbsoluteTTL: sessionAbsoluteTTL,
		CookieSecure:       cookieSecure,
		ShutdownTimeout:    shutdownTimeout,
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.DatabaseDSN == "" {
		return fmt.Errorf("VOIDXO_DATABASE_DSN is required")
	}
	if strings.TrimSpace(c.RedisAddr) == "" {
		return fmt.Errorf("VOIDXO_REDIS_ADDR is required")
	}
	if c.RedisDB < 0 {
		return fmt.Errorf("VOIDXO_REDIS_DB must be non-negative")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("VOIDXO_SHUTDOWN_TIMEOUT must be positive")
	}
	if c.SessionIdleTTL <= 0 || c.SessionAbsoluteTTL <= 0 {
		return fmt.Errorf("session TTLs must be positive")
	}
	if c.SessionIdleTTL > c.SessionAbsoluteTTL {
		return fmt.Errorf("VOIDXO_SESSION_IDLE_TTL cannot exceed VOIDXO_SESSION_ABSOLUTE_TTL")
	}
	if c.Environment != "development" && c.Environment != "test" && !c.CookieSecure {
		return fmt.Errorf("VOIDXO_COOKIE_SECURE must be true outside development and test")
	}
	for _, origin := range c.AllowedOrigins {
		if err := validateOrigin(origin); err != nil {
			return err
		}
	}
	for _, proxy := range c.TrustedProxies {
		if _, _, err := net.ParseCIDR(proxy); err != nil {
			return fmt.Errorf("invalid VOIDXO_TRUSTED_PROXIES entry %q", proxy)
		}
	}
	return nil
}

func validateOrigin(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return fmt.Errorf("invalid VOIDXO_ALLOWED_ORIGINS entry %q", value)
	}
	return nil
}

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		if seconds, secondsErr := strconv.Atoi(value); secondsErr == nil {
			if seconds <= 0 {
				return 0, fmt.Errorf("%s must be positive", key)
			}
			return time.Duration(seconds) * time.Second, nil
		}
		return 0, fmt.Errorf("%s must be a duration such as 10s", key)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return parsed, nil
}

func intEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return parsed, nil
}

func boolEnv(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return parsed, nil
}
