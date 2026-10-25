package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type ApiConfig struct {
	Port              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxBodyBytes      int64
	// Requests per minute allowed per client IP on public routes.
	RateLimitPerMinute int
	// When true, X-Forwarded-For / X-Real-IP are trusted for client IP
	// (only enable behind a proxy you control, e.g. Fly.io's edge).
	TrustProxy bool
}

type DbConfig struct {
	DataSourceName  string
	MaxConns        int32
	MigrateOnBoot   bool
	ConnMaxLifetime time.Duration
}

type CorsConfig struct {
	AllowedOrigins []string
}

type AdminConfig struct {
	Email              string
	PasswordBcryptHash string
}

type Config struct {
	Env string
	ApiConfig
	DbConfig
	CorsConfig
	AdminConfig
}

func (c Config) IsProduction() bool {
	return c.Env == "production"
}

// NewConfig loads configuration from the environment. A local .env file is
// read if present (dev convenience); real deployments use host secrets.
func NewConfig() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		Env: getEnv("APP_ENV", "development"),
		ApiConfig: ApiConfig{
			Port:               getEnv("PORT", "8080"),
			ReadHeaderTimeout:  5 * time.Second,
			ReadTimeout:        15 * time.Second,
			WriteTimeout:       15 * time.Second,
			IdleTimeout:        60 * time.Second,
			ShutdownTimeout:    10 * time.Second,
			MaxBodyBytes:       int64(getEnvInt("MAX_BODY_BYTES", 1<<20)),
			RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 120),
			TrustProxy:         getEnvBool("TRUST_PROXY", false),
		},
		DbConfig: DbConfig{
			DataSourceName:  os.Getenv("DATABASE_URL"),
			MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 10)),
			MigrateOnBoot:   getEnvBool("MIGRATE_ON_BOOT", true),
			ConnMaxLifetime: time.Hour,
		},
		CorsConfig: CorsConfig{
			AllowedOrigins: splitCSV(os.Getenv("CORS_ORIGIN")),
		},
		AdminConfig: AdminConfig{
			Email:              strings.TrimSpace(os.Getenv("ADMIN_EMAIL")),
			PasswordBcryptHash: strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_BCRYPT_HASH")),
		},
	}

	return cfg, cfg.validate()
}

func (c Config) validate() error {
	var errs []error
	if c.DataSourceName == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.IsProduction() {
		if len(c.AllowedOrigins) == 0 {
			errs = append(errs, errors.New("CORS_ORIGIN is required in production"))
		}
		for _, o := range c.AllowedOrigins {
			if o == "*" {
				errs = append(errs, errors.New("CORS_ORIGIN must not be '*' in production"))
			}
		}
	}
	if (c.Email == "") != (c.PasswordBcryptHash == "") {
		errs = append(errs, errors.New("ADMIN_EMAIL and ADMIN_PASSWORD_BCRYPT_HASH must be set together"))
	}
	if c.PasswordBcryptHash != "" && !strings.HasPrefix(c.PasswordBcryptHash, "$2") {
		errs = append(errs, errors.New("ADMIN_PASSWORD_BCRYPT_HASH must be a bcrypt hash, not a plaintext password"))
	}
	if c.MaxBodyBytes <= 0 {
		errs = append(errs, fmt.Errorf("MAX_BODY_BYTES must be positive"))
	}
	return errors.Join(errs...)
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
