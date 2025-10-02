package config

import (
    "fmt"
    "time"

    "github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
    // Server configuration
    Port         string `mapstructure:"PORT"`
    Environment  string `mapstructure:"ENVIRONMENT"`
    APIPrefix    string `mapstructure:"API_PREFIX"`
    
    // Database configuration
    DatabaseURL string         `mapstructure:"DATABASE_URL"`
    Database    DatabaseConfig `mapstructure:",squash"`
    
    // Redis configuration
    RedisURL string      `mapstructure:"REDIS_URL"`
    Redis    RedisConfig `mapstructure:",squash"`
    
    // JWT configuration
    JWTSecret              string `mapstructure:"JWT_SECRET"`
    JWTRefreshSecret       string `mapstructure:"JWT_REFRESH_SECRET"`
    JWTExpiryHours         int    `mapstructure:"JWT_EXPIRY_HOURS"`
    JWTRefreshExpiryDays   int    `mapstructure:"JWT_REFRESH_EXPIRY_DAYS"`
    JWTIssuer              string `mapstructure:"JWT_ISSUER"`
    JWTAudience            string `mapstructure:"JWT_AUDIENCE"`
    
    // MinIO configuration
    MinIOEndpoint    string `mapstructure:"MINIO_ENDPOINT"`
    MinIOAccessKey   string `mapstructure:"MINIO_ACCESS_KEY"`
    MinIOSecretKey   string `mapstructure:"MINIO_SECRET_KEY"`
    MinIOBucket      string `mapstructure:"MINIO_BUCKET"`
    MinIOUseSSL      bool   `mapstructure:"MINIO_USE_SSL"`
    MinIORegion      string `mapstructure:"MINIO_REGION"`
    
    // Media configuration
    PresignedURLExpiryMinutes int `mapstructure:"PRESIGNED_URL_EXPIRY_MINUTES"`
    
    // Rate limiting
    RateLimit RateLimitConfig `mapstructure:",squash"`
    
    // CORS configuration
    CORS CORSConfig `mapstructure:",squash"`
    
    // Logging configuration
    LogLevel  string `mapstructure:"LOG_LEVEL"`
    LogFormat string `mapstructure:"LOG_FORMAT"`
    
    // Worker configuration
    Worker WorkerConfig `mapstructure:",squash"`
    
    // Prometheus configuration
    PrometheusEnabled bool   `mapstructure:"PROMETHEUS_ENABLED"`
    PrometheusPort    string `mapstructure:"PROMETHEUS_PORT"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
    MaxOpenConns int           `mapstructure:"DB_MAX_OPEN_CONNS"`
    MaxIdleConns int           `mapstructure:"DB_MAX_IDLE_CONNS"`
    MaxLifetime  time.Duration `mapstructure:"DB_MAX_LIFETIME"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
    Password     string        `mapstructure:"REDIS_PASSWORD"`
    DB           int           `mapstructure:"REDIS_DB"`
    PoolSize     int           `mapstructure:"REDIS_POOL_SIZE"`
    MinIdleConns int           `mapstructure:"REDIS_MIN_IDLE_CONNS"`
    IdleTimeout  time.Duration `mapstructure:"REDIS_IDLE_TIMEOUT"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
    Enabled           bool `mapstructure:"RATE_LIMIT_ENABLED"`
    RequestsPerMinute int  `mapstructure:"RATE_LIMIT_REQUESTS_PER_MINUTE"`
    Burst             int  `mapstructure:"RATE_LIMIT_BURST"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
    Enabled         bool     `mapstructure:"CORS_ENABLED"`
    AllowedOrigins  []string `mapstructure:"CORS_ALLOWED_ORIGINS"`
    AllowedMethods  []string `mapstructure:"CORS_ALLOWED_METHODS"`
    AllowedHeaders  []string `mapstructure:"CORS_ALLOWED_HEADERS"`
    ExposedHeaders  []string `mapstructure:"CORS_EXPOSE_HEADERS"`
    AllowCredentials bool    `mapstructure:"CORS_ALLOW_CREDENTIALS"`
    MaxAge          int      `mapstructure:"CORS_MAX_AGE"`
}

// WorkerConfig holds worker configuration
type WorkerConfig struct {
    PollInterval time.Duration `mapstructure:"WORKER_POLL_INTERVAL"`
    BatchSize    int           `mapstructure:"WORKER_BATCH_SIZE"`
    MaxRetries   int           `mapstructure:"WORKER_MAX_RETRIES"`
    Timeout      time.Duration `mapstructure:"WORKER_TIMEOUT"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./config")
    
    // Set defaults
    setDefaults()
    
    // Read config file (optional)
    viper.ReadInConfig()
    
    // Override with environment variables
    viper.AutomaticEnv()
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    // Validate required fields
    if err := validateConfig(&config); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
    // Server defaults
    viper.SetDefault("PORT", "8080")
    viper.SetDefault("ENVIRONMENT", "development")
    viper.SetDefault("API_PREFIX", "/api/v1")
    
    // Database defaults
    viper.SetDefault("DB_MAX_OPEN_CONNS", 50)
    viper.SetDefault("DB_MAX_IDLE_CONNS", 10)
    viper.SetDefault("DB_MAX_LIFETIME", "1800s")
    
    // Redis defaults
    viper.SetDefault("REDIS_DB", 0)
    viper.SetDefault("REDIS_POOL_SIZE", 10)
    viper.SetDefault("REDIS_MIN_IDLE_CONNS", 5)
    viper.SetDefault("REDIS_IDLE_TIMEOUT", "300s")
    
    // JWT defaults
    viper.SetDefault("JWT_EXPIRY_HOURS", 24)
    viper.SetDefault("JWT_REFRESH_EXPIRY_DAYS", 7)
    viper.SetDefault("JWT_ISSUER", "stories-backend")
    viper.SetDefault("JWT_AUDIENCE", "stories-app")
    
    // MinIO defaults
    viper.SetDefault("MINIO_BUCKET", "stories-media")
    viper.SetDefault("MINIO_USE_SSL", false)
    viper.SetDefault("MINIO_REGION", "us-east-1")
    viper.SetDefault("PRESIGNED_URL_EXPIRY_MINUTES", 15)
    
    // Rate limiting defaults
    viper.SetDefault("RATE_LIMIT_ENABLED", true)
    viper.SetDefault("RATE_LIMIT_REQUESTS_PER_MINUTE", 60)
    viper.SetDefault("RATE_LIMIT_BURST", 10)
    
    // CORS defaults
    viper.SetDefault("CORS_ENABLED", true)
    viper.SetDefault("CORS_ALLOWED_ORIGINS", []string{"*"})
    viper.SetDefault("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
    viper.SetDefault("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization"})
    viper.SetDefault("CORS_ALLOW_CREDENTIALS", true)
    viper.SetDefault("CORS_MAX_AGE", 86400)
    
    // Logging defaults
    viper.SetDefault("LOG_LEVEL", "info")
    viper.SetDefault("LOG_FORMAT", "json")
    
    // Worker defaults
    viper.SetDefault("WORKER_POLL_INTERVAL", "30s")
    viper.SetDefault("WORKER_BATCH_SIZE", 100)
    viper.SetDefault("WORKER_MAX_RETRIES", 3)
    viper.SetDefault("WORKER_TIMEOUT", "300s")
    
    // Prometheus defaults
    viper.SetDefault("PROMETHEUS_ENABLED", true)
    viper.SetDefault("PROMETHEUS_PORT", "9090")
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
    if config.DatabaseURL == "" {
        return fmt.Errorf("DATABASE_URL is required")
    }
    
    if config.RedisURL == "" {
        return fmt.Errorf("REDIS_URL is required")
    }
    
    if config.JWTSecret == "" {
        return fmt.Errorf("JWT_SECRET is required")
    }
    
    if config.JWTRefreshSecret == "" {
        return fmt.Errorf("JWT_REFRESH_SECRET is required")
    }
    
    if config.MinIOEndpoint == "" {
        return fmt.Errorf("MINIO_ENDPOINT is required")
    }
    
    if config.MinIOAccessKey == "" {
        return fmt.Errorf("MINIO_ACCESS_KEY is required")
    }
    
    if config.MinIOSecretKey == "" {
        return fmt.Errorf("MINIO_SECRET_KEY is required")
    }
    
    return nil
}
