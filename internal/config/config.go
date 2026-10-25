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
	// When true, the client IP is read from ClientIPHeader, which the edge
	// proxy must overwrite on every request (Fly.io: Fly-Client-IP).
	TrustProxy     bool
	ClientIPHeader string
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

// WebConfig lets the API ask the Next.js frontend to drop cached pages after
// content changes. Optional: without it, pages refresh within the ISR window.
type WebConfig struct {
	WebRevalidateURL string
	RevalidateSecret string
}

type AdminConfig struct {
	Email              string
	PasswordBcryptHash string
}

type AuthConfig struct {
	JWTSecret []byte
	JWTTTL    time.Duration
	JWTIssuer string
	// Login attempts allowed per client IP per minute.
	LoginRateLimitPerMinute int
}

type StorageConfig struct {
	// "local" (dev: files on disk, served by this API) or "r2".
	Driver         string
	MaxUploadBytes int64
	// Local driver.
	UploadDir     string
	PublicBaseURL string // e.g. http://localhost:8080, used to build file URLs
	// R2 driver (S3-compatible).
	R2Endpoint        string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
	R2PublicBaseURL   string // public bucket URL or custom domain
}

type Config struct {
	Env string
	ApiConfig
	DbConfig
	CorsConfig
	WebConfig
	AdminConfig
	AuthConfig
	StorageConfig
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
			ReadTimeout:        30 * time.Second, // image uploads on slow links
			WriteTimeout:       30 * time.Second,
			IdleTimeout:        60 * time.Second,
			ShutdownTimeout:    10 * time.Second,
			MaxBodyBytes:       int64(getEnvInt("MAX_BODY_BYTES", 1<<20)),
			RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 120),
			TrustProxy:         getEnvBool("TRUST_PROXY", false),
			ClientIPHeader:     getEnv("CLIENT_IP_HEADER", "Fly-Client-IP"),
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
		WebConfig: WebConfig{
			WebRevalidateURL: strings.TrimSpace(os.Getenv("WEB_REVALIDATE_URL")),
			RevalidateSecret: os.Getenv("REVALIDATE_SECRET"),
		},
		AdminConfig: AdminConfig{
			Email:              strings.TrimSpace(os.Getenv("ADMIN_EMAIL")),
			PasswordBcryptHash: strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_BCRYPT_HASH")),
		},
		AuthConfig: AuthConfig{
			JWTSecret:               []byte(os.Getenv("JWT_SECRET")),
			JWTTTL:                  time.Duration(getEnvInt("JWT_TTL_HOURS", 24)) * time.Hour,
			JWTIssuer:               getEnv("JWT_ISSUER", "portfolio-api"),
			LoginRateLimitPerMinute: getEnvInt("LOGIN_RATE_LIMIT_PER_MINUTE", 5),
		},
		StorageConfig: StorageConfig{
			Driver:            getEnv("STORAGE_DRIVER", "local"),
			MaxUploadBytes:    int64(getEnvInt("MAX_UPLOAD_BYTES", 15<<20)),
			UploadDir:         getEnv("UPLOAD_DIR", "./uploads"),
			PublicBaseURL:     strings.TrimRight(getEnv("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
			R2Endpoint:        os.Getenv("R2_ENDPOINT"),
			R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
			R2SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
			R2Bucket:          os.Getenv("R2_BUCKET"),
			R2PublicBaseURL:   strings.TrimRight(os.Getenv("R2_PUBLIC_BASE_URL"), "/"),
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
	// HS256 key must be at least as long as the hash output.
	if len(c.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET must be at least 32 bytes (generate with: openssl rand -base64 48)"))
	}
	if c.JWTTTL <= 0 || c.JWTTTL > 7*24*time.Hour {
		errs = append(errs, errors.New("JWT_TTL_HOURS must be between 1 and 168"))
	}
	if c.WebRevalidateURL != "" && len(c.RevalidateSecret) < 32 {
		errs = append(errs, errors.New("REVALIDATE_SECRET must be at least 32 bytes when WEB_REVALIDATE_URL is set"))
	}
	switch c.Driver {
	case "local":
		if c.IsProduction() {
			errs = append(errs, errors.New("STORAGE_DRIVER=local is for development; use r2 in production"))
		}
	case "r2":
		if c.R2Endpoint == "" || c.R2AccessKeyID == "" || c.R2SecretAccessKey == "" || c.R2Bucket == "" || c.R2PublicBaseURL == "" {
			errs = append(errs, errors.New("STORAGE_DRIVER=r2 requires R2_ENDPOINT, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET and R2_PUBLIC_BASE_URL"))
		}
	default:
		errs = append(errs, fmt.Errorf("STORAGE_DRIVER must be local or r2, got %q", c.Driver))
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
