package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env     string
	Port    int
	BaseURL string

	DB DBConfig
	JWT JWTConfig
	Redis RedisConfig
	S3    S3Config
	RateLimit RateLimitConfig
	Idempotency IdempotencyConfig
}

type DBConfig struct {
	Driver string
	DSN    string
}

type JWTConfig struct {
	Secret          string
	AccessTTL       time.Duration
	RefreshTTL      time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
}

type S3Config struct {
	Endpoint       string
	Bucket         string
	AccessKey      string
	SecretKey      string
	Region         string
	PresignTTL     time.Duration
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

type IdempotencyConfig struct {
	TTL time.Duration
}

func Load() *Config {
	port := 8080
	if p := os.Getenv("APP_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	accessTTL := 15 * time.Minute
	if v := os.Getenv("JWT_ACCESS_TTL_MINUTES"); v != "" {
		if mins, err := strconv.Atoi(v); err == nil {
			accessTTL = time.Duration(mins) * time.Minute
		}
	}

	refreshTTL := 7 * 24 * time.Hour
	if v := os.Getenv("JWT_REFRESH_TTL_DAYS"); v != "" {
		if days, err := strconv.Atoi(v); err == nil {
			refreshTTL = time.Duration(days) * 24 * time.Hour
		}
	}

	rateLimitReqs := 100
	if v := os.Getenv("RATE_LIMIT_REQUESTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rateLimitReqs = n
		}
	}
	rateLimitWindow := 60 * time.Second
	if v := os.Getenv("RATE_LIMIT_WINDOW_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rateLimitWindow = time.Duration(n) * time.Second
		}
	}

	idempotencyTTL := 24 * time.Hour
	if v := os.Getenv("IDEMPOTENCY_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			idempotencyTTL = time.Duration(n) * time.Hour
		}
	}

	presignTTL := 15 * time.Minute
	if v := os.Getenv("S3_PRESIGN_TTL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			presignTTL = time.Duration(n) * time.Minute
		}
	}

	cfg := &Config{
		Env:     getEnv("APP_ENV", "development"),
		Port:    port,
		BaseURL: getEnv("APP_BASE_URL", "http://localhost:8080"),
		DB: DBConfig{
			Driver: getEnv("DB_DRIVER", ""),
			DSN:    getEnv("DB_DSN", ""),
		},
		JWT: JWTConfig{
			Secret:     mustEnv("JWT_SECRET"),
			AccessTTL:  accessTTL,
			RefreshTTL: refreshTTL,
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
		S3: S3Config{
			Endpoint:   getEnv("S3_ENDPOINT", ""),
			Bucket:     getEnv("S3_BUCKET", "clarity-documents"),
			AccessKey:  getEnv("S3_ACCESS_KEY", ""),
			SecretKey:  getEnv("S3_SECRET_KEY", ""),
			Region:     getEnv("S3_REGION", "eu-west-1"),
			PresignTTL: presignTTL,
		},
		RateLimit: RateLimitConfig{
			Requests: rateLimitReqs,
			Window:   rateLimitWindow,
		},
		Idempotency: IdempotencyConfig{
			TTL: idempotencyTTL,
		},
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}
