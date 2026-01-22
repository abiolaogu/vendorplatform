// =============================================================================
// CONFIGURATION PACKAGE
// Centralized configuration management with environment variables and defaults
// =============================================================================

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Application
	App AppConfig
	
	// Server
	Server ServerConfig
	
	// Database
	Database DatabaseConfig
	
	// Redis
	Redis RedisConfig
	
	// Authentication
	Auth AuthConfig
	
	// Payment
	Payment PaymentConfig
	
	// Storage
	Storage StorageConfig
	
	// Search
	Search SearchConfig
	
	// Notification
	Notification NotificationConfig
	
	// Features
	Features FeatureFlags
}

// AppConfig for application settings
type AppConfig struct {
	Name        string
	Environment string // development, staging, production
	Version     string
	Debug       bool
	LogLevel    string
}

// ServerConfig for HTTP server
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	TrustedProxies  []string
	CORSOrigins     []string
}

// DatabaseConfig for PostgreSQL
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// URL returns the database connection URL
func (c DatabaseConfig) URL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

// RedisConfig for Redis
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// URL returns the Redis connection URL
func (c RedisConfig) URL() string {
	if c.Password != "" {
		return fmt.Sprintf("redis://:%s@%s:%d/%d", c.Password, c.Host, c.Port, c.DB)
	}
	return fmt.Sprintf("redis://%s:%d/%d", c.Host, c.Port, c.DB)
}

// AuthConfig for authentication
type AuthConfig struct {
	JWTSecret           string
	AccessTokenExpiry   time.Duration
	RefreshTokenExpiry  time.Duration
	BCryptCost          int
	MaxSessionsPerUser  int
	VerificationExpiry  time.Duration
}

// PaymentConfig for payment providers
type PaymentConfig struct {
	PaystackSecretKey    string
	PaystackPublicKey    string
	FlutterwaveSecretKey string
	FlutterwavePublicKey string
	StripeSecretKey      string
	StripePublicKey      string
	WebhookSecret        string
	DefaultCurrency      string
	PlatformFeePercent   float64
	EscrowExpiryDays     int
}

// StorageConfig for file storage
type StorageConfig struct {
	Provider     string // "s3", "local"
	S3Bucket     string
	S3Region     string
	S3Endpoint   string
	LocalPath    string
	LocalBaseURL string
	CDNBaseURL   string
	MaxFileSize  int64
}

// SearchConfig for Elasticsearch
type SearchConfig struct {
	URL         string
	IndexPrefix string
	CacheTTL    time.Duration
}

// NotificationConfig for notifications
type NotificationConfig struct {
	// Email
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
	
	// SMS
	TermiiAPIKey string
	TermiiSender string
	
	// Push
	OneSignalAppID  string
	OneSignalAPIKey string
}

// FeatureFlags for feature toggles
type FeatureFlags struct {
	EnablePayments      bool
	EnableNotifications bool
	EnableSearch        bool
	EnableAnalytics     bool
	EnableHomeRescue    bool
	EnableVendorNet     bool
	EnableEventGPT      bool
	EnableLifeOS        bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "vendorplatform"),
			Environment: getEnv("ENV", "development"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Debug:       getEnvBool("DEBUG", true),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
		},
		Server: ServerConfig{
			Host:            getEnv("HOST", ""),
			Port:            getEnvInt("PORT", 8080),
			ReadTimeout:     getEnvDuration("READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
			TrustedProxies:  getEnvSlice("TRUSTED_PROXIES", []string{}),
			CORSOrigins:     getEnvSlice("CORS_ORIGINS", []string{"*"}),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "vendorplatform"),
			Password:        getEnv("DB_PASSWORD", "vendorplatform"),
			Database:        getEnv("DB_NAME", "vendorplatform"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 25)),
			MinConns:        int32(getEnvInt("DB_MIN_CONNS", 5)),
			MaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime: getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Auth: AuthConfig{
			JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
			AccessTokenExpiry:  getEnvDuration("ACCESS_TOKEN_EXPIRY", 15*time.Minute),
			RefreshTokenExpiry: getEnvDuration("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour),
			BCryptCost:         getEnvInt("BCRYPT_COST", 12),
			MaxSessionsPerUser: getEnvInt("MAX_SESSIONS_PER_USER", 5),
			VerificationExpiry: getEnvDuration("VERIFICATION_EXPIRY", 24*time.Hour),
		},
		Payment: PaymentConfig{
			PaystackSecretKey:    getEnv("PAYSTACK_SECRET_KEY", ""),
			PaystackPublicKey:    getEnv("PAYSTACK_PUBLIC_KEY", ""),
			FlutterwaveSecretKey: getEnv("FLUTTERWAVE_SECRET_KEY", ""),
			FlutterwavePublicKey: getEnv("FLUTTERWAVE_PUBLIC_KEY", ""),
			StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			StripePublicKey:      getEnv("STRIPE_PUBLIC_KEY", ""),
			WebhookSecret:        getEnv("PAYMENT_WEBHOOK_SECRET", ""),
			DefaultCurrency:      getEnv("DEFAULT_CURRENCY", "NGN"),
			PlatformFeePercent:   getEnvFloat("PLATFORM_FEE_PERCENT", 10.0),
			EscrowExpiryDays:     getEnvInt("ESCROW_EXPIRY_DAYS", 14),
		},
		Storage: StorageConfig{
			Provider:     getEnv("STORAGE_PROVIDER", "local"),
			S3Bucket:     getEnv("S3_BUCKET", ""),
			S3Region:     getEnv("S3_REGION", "eu-west-1"),
			S3Endpoint:   getEnv("S3_ENDPOINT", ""),
			LocalPath:    getEnv("LOCAL_STORAGE_PATH", "./uploads"),
			LocalBaseURL: getEnv("LOCAL_STORAGE_URL", "http://localhost:8080/uploads"),
			CDNBaseURL:   getEnv("CDN_BASE_URL", ""),
			MaxFileSize:  int64(getEnvInt("MAX_FILE_SIZE", 50*1024*1024)),
		},
		Search: SearchConfig{
			URL:         getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
			IndexPrefix: getEnv("ELASTICSEARCH_INDEX_PREFIX", "vendorplatform_"),
			CacheTTL:    getEnvDuration("SEARCH_CACHE_TTL", 5*time.Minute),
		},
		Notification: NotificationConfig{
			SMTPHost:        getEnv("SMTP_HOST", ""),
			SMTPPort:        getEnvInt("SMTP_PORT", 587),
			SMTPUser:        getEnv("SMTP_USER", ""),
			SMTPPassword:    getEnv("SMTP_PASSWORD", ""),
			FromEmail:       getEnv("FROM_EMAIL", "noreply@vendorplatform.com"),
			FromName:        getEnv("FROM_NAME", "VendorPlatform"),
			TermiiAPIKey:    getEnv("TERMII_API_KEY", ""),
			TermiiSender:    getEnv("TERMII_SENDER", "VendorPlatform"),
			OneSignalAppID:  getEnv("ONESIGNAL_APP_ID", ""),
			OneSignalAPIKey: getEnv("ONESIGNAL_API_KEY", ""),
		},
		Features: FeatureFlags{
			EnablePayments:      getEnvBool("FEATURE_PAYMENTS", true),
			EnableNotifications: getEnvBool("FEATURE_NOTIFICATIONS", true),
			EnableSearch:        getEnvBool("FEATURE_SEARCH", true),
			EnableAnalytics:     getEnvBool("FEATURE_ANALYTICS", true),
			EnableHomeRescue:    getEnvBool("FEATURE_HOMERESCUE", true),
			EnableVendorNet:     getEnvBool("FEATURE_VENDORNET", true),
			EnableEventGPT:      getEnvBool("FEATURE_EVENTGPT", true),
			EnableLifeOS:        getEnvBool("FEATURE_LIFEOS", true),
		},
	}
	
	// Validate required settings for production
	if cfg.App.Environment == "production" {
		if cfg.Auth.JWTSecret == "change-me-in-production" {
			return nil, fmt.Errorf("JWT_SECRET must be set in production")
		}
	}
	
	return cfg, nil
}

// Helper functions

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return strings.ToLower(val) == "true" || val == "1"
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvSlice(key string, defaultVal []string) []string {
	if val := os.Getenv(key); val != "" {
		return strings.Split(val, ",")
	}
	return defaultVal
}
