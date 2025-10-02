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
    
    // Worker configuration - UPDATED to match your env vars
    Workers WorkersConfig `mapstructure:",squash"`
    
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

// WorkerConfig represents configuration for a single worker
type WorkerConfig struct {
    Enabled     bool          `mapstructure:"enabled" json:"enabled"`
    Interval    time.Duration `mapstructure:"interval" json:"interval"`
    BatchSize   int           `mapstructure:"batch_size" json:"batch_size"`
    Workers     int           `mapstructure:"workers" json:"workers"`        // For queue concurrency
    Concurrency int           `mapstructure:"concurrency" json:"concurrency"` // Alternative name for workers
    MaxRetries  int           `mapstructure:"max_retries" json:"max_retries"`
    Timeout     time.Duration `mapstructure:"timeout" json:"timeout"`
}

// WorkersConfig represents all worker configurations - UPDATED to match your env vars
type WorkersConfig struct {
    // Expiration worker (matches EXPIRATION_WORKER_* env vars)
    StoryExpiration WorkerConfig `mapstructure:"expiration_worker" json:"expiration_worker"`
    
    // Queue worker (matches QUEUE_WORKER_* env vars)
    Queue WorkerConfig `mapstructure:"queue_worker" json:"queue_worker"`
    
    // Additional workers with default names
    CacheCleanup  WorkerConfig `mapstructure:"cache_cleanup" json:"cache_cleanup"`
    UserStats     WorkerConfig `mapstructure:"user_stats" json:"user_stats"`
    Notifications WorkerConfig `mapstructure:"notifications" json:"notifications"`
    Analytics     WorkerConfig `mapstructure:"analytics" json:"analytics"`
    HealthCheck   WorkerConfig `mapstructure:"health_check" json:"health_check"`
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
    
    // Post-process worker configs to handle different field names
    normalizeWorkerConfigs(&config)
    
    // Validate required fields
    if err := validateConfig(&config); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return &config, nil
}

// normalizeWorkerConfigs handles different naming conventions
func normalizeWorkerConfigs(config *Config) {
    // For queue worker, if Concurrency is set but Workers is not, use Concurrency as Workers
    if config.Workers.Queue.Concurrency > 0 && config.Workers.Queue.Workers == 0 {
        config.Workers.Queue.Workers = config.Workers.Queue.Concurrency
    }
    
    // Set defaults for unset values
    if config.Workers.StoryExpiration.Interval == 0 {
        config.Workers.StoryExpiration.Interval = 5 * time.Minute
    }
    if config.Workers.Queue.Interval == 0 {
        config.Workers.Queue.Interval = 5 * time.Second
    }
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
    
    // Worker defaults - UPDATED to match your env var names
    setWorkerDefaults()
    
    // Prometheus defaults
    viper.SetDefault("PROMETHEUS_ENABLED", true)
    viper.SetDefault("PROMETHEUS_PORT", "9090")
}

// setWorkerDefaults sets worker-specific defaults - UPDATED
func setWorkerDefaults() {
    // Expiration worker (matches your EXPIRATION_WORKER_* env vars)
    viper.SetDefault("EXPIRATION_WORKER_ENABLED", true)
    viper.SetDefault("EXPIRATION_WORKER_INTERVAL", "60s")
    viper.SetDefault("EXPIRATION_WORKER_BATCH_SIZE", 1000)
    viper.SetDefault("EXPIRATION_WORKER_TIMEOUT", "30s")
    viper.SetDefault("EXPIRATION_WORKER_MAX_RETRIES", 3)
    
    // Queue worker (matches your QUEUE_WORKER_* env vars)
    viper.SetDefault("QUEUE_WORKER_ENABLED", true)
    viper.SetDefault("QUEUE_WORKER_INTERVAL", "5s")
    viper.SetDefault("QUEUE_WORKER_CONCURRENCY", 5)
    viper.SetDefault("QUEUE_WORKER_TIMEOUT", "5m")
    viper.SetDefault("QUEUE_WORKER_MAX_RETRIES", 3)
    
    // Additional workers with default values
    viper.SetDefault("CACHE_CLEANUP_ENABLED", true)
    viper.SetDefault("CACHE_CLEANUP_INTERVAL", "1h")
    viper.SetDefault("CACHE_CLEANUP_BATCH_SIZE", 1000)
    viper.SetDefault("CACHE_CLEANUP_TIMEOUT", "10m")
    
    viper.SetDefault("USER_STATS_ENABLED", true)
    viper.SetDefault("USER_STATS_INTERVAL", "10m")
    viper.SetDefault("USER_STATS_BATCH_SIZE", 50)
    viper.SetDefault("USER_STATS_TIMEOUT", "5m")
    
    viper.SetDefault("NOTIFICATIONS_ENABLED", true)
    viper.SetDefault("NOTIFICATIONS_INTERVAL", "1s")
    viper.SetDefault("NOTIFICATIONS_CONCURRENCY", 3)
    viper.SetDefault("NOTIFICATIONS_MAX_RETRIES", 5)
    viper.SetDefault("NOTIFICATIONS_TIMEOUT", "30s")
    
    viper.SetDefault("ANALYTICS_ENABLED", true)
    viper.SetDefault("ANALYTICS_INTERVAL", "15m")
    viper.SetDefault("ANALYTICS_BATCH_SIZE", 200)
    viper.SetDefault("ANALYTICS_TIMEOUT", "10m")
    
    viper.SetDefault("HEALTH_CHECK_ENABLED", true)
    viper.SetDefault("HEALTH_CHECK_INTERVAL", "30s")
    viper.SetDefault("HEALTH_CHECK_TIMEOUT", "10s")
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
