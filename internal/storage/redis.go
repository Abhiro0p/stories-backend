package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// RedisClient wraps redis client with additional functionality
type RedisClient struct {
    client *redis.Client
    logger *zap.Logger
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.Config, logger *zap.Logger) (*RedisClient, error) {
    options, err := redis.ParseURL(cfg.RedisURL)
    if err != nil {
        return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
    }

    // Override with config values if provided
    if cfg.Redis.Password != "" {
        options.Password = cfg.Redis.Password
    }
    if cfg.Redis.DB != 0 {
        options.DB = cfg.Redis.DB
    }
    if cfg.Redis.PoolSize > 0 {
        options.PoolSize = cfg.Redis.PoolSize
    }
    if cfg.Redis.MinIdleConns > 0 {
        options.MinIdleConns = cfg.Redis.MinIdleConns
    }

    client := redis.NewClient(options)

    // Test connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }

    logger.Info("Connected to Redis successfully",
        zap.String("addr", options.Addr),
        zap.Int("db", options.DB),
    )

    return &RedisClient{
        client: client,
        logger: logger.With(zap.String("component", "redis")),
    }, nil
}

// Set stores a value in Redis with expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration int) error {
    data, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("failed to marshal value: %w", err)
    }

    duration := time.Duration(expiration) * time.Second
    return r.client.Set(ctx, key, data, duration).Err()
}

// Get retrieves a value from Redis
func (r *RedisClient) Get(ctx context.Context, key string, dest interface{}) error {
    data, err := r.client.Get(ctx, key).Result()
    if err != nil {
        if err == redis.Nil {
            return ErrNotFound
        }
        return fmt.Errorf("failed to get value: %w", err)
    }

    return json.Unmarshal([]byte(data), dest)
}

// Delete removes a key from Redis
func (r *RedisClient) Delete(ctx context.Context, key string) error {
    return r.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in Redis
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
    count, err := r.client.Exists(ctx, key).Result()
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

// Increment increments a key's value
func (r *RedisClient) Increment(ctx context.Context, key string, value int64) (int64, error) {
    return r.client.IncrBy(ctx, key, value).Result()
}

// Expire sets expiration for a key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration int) error {
    duration := time.Duration(expiration) * time.Second
    return r.client.Expire(ctx, key, duration).Err()
}

// Health checks Redis connection health
func (r *RedisClient) Health(ctx context.Context) error {
    return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
    return r.client.Close()
}

// Pipeline operations for batch processing
func (r *RedisClient) Pipeline() redis.Pipeliner {
    return r.client.Pipeline()
}

// GetClient returns the underlying Redis client
func (r *RedisClient) GetClient() *redis.Client {
    return r.client
}

// SetMany sets multiple key-value pairs
func (r *RedisClient) SetMany(ctx context.Context, items map[string]interface{}, expiration int) error {
    pipe := r.client.Pipeline()
    duration := time.Duration(expiration) * time.Second
    
    for key, value := range items {
        data, err := json.Marshal(value)
        if err != nil {
            return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
        }
        pipe.Set(ctx, key, data, duration)
    }
    
    _, err := pipe.Exec(ctx)
    return err
}

// GetMany gets multiple keys
func (r *RedisClient) GetMany(ctx context.Context, keys []string) (map[string]interface{}, error) {
    pipe := r.client.Pipeline()
    cmds := make([]*redis.StringCmd, len(keys))
    
    for i, key := range keys {
        cmds[i] = pipe.Get(ctx, key)
    }
    
    _, err := pipe.Exec(ctx)
    if err != nil {
        return nil, err
    }
    
    result := make(map[string]interface{})
    for i, cmd := range cmds {
        val, err := cmd.Result()
        if err == nil {
            var data interface{}
            if json.Unmarshal([]byte(val), &data) == nil {
                result[keys[i]] = data
            }
        }
    }
    
    return result, nil
}

// DeleteMany deletes multiple keys
func (r *RedisClient) DeleteMany(ctx context.Context, keys []string) error {
    if len(keys) == 0 {
        return nil
    }
    return r.client.Del(ctx, keys...).Err()
}

// Decrement decrements a key's value
func (r *RedisClient) Decrement(ctx context.Context, key string, value int64) (int64, error) {
    return r.client.DecrBy(ctx, key, value).Result()
}

// TTL gets the time to live for a key
func (r *RedisClient) TTL(ctx context.Context, key string) (int, error) {
    duration, err := r.client.TTL(ctx, key).Result()
    if err != nil {
        return 0, err
    }
    return int(duration.Seconds()), nil
}

// FlushAll flushes all keys
func (r *RedisClient) FlushAll(ctx context.Context) error {
    return r.client.FlushAll(ctx).Err()
}
// Add this method to your existing RedisClient struct in redis.go

// Eval executes a Lua script
func (r *RedisClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
    return r.client.Eval(ctx, script, keys, args...)
}

// EvalSha executes a Lua script by SHA
func (r *RedisClient) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
    return r.client.EvalSha(ctx, sha1, keys, args...)
}

// ScriptExists checks if scripts exist
func (r *RedisClient) ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd {
    return r.client.ScriptExists(ctx, hashes...)
}

// ScriptFlush flushes all scripts
func (r *RedisClient) ScriptFlush(ctx context.Context) *redis.StatusCmd {
    return r.client.ScriptFlush(ctx)
}

// ScriptKill kills a running script
func (r *RedisClient) ScriptKill(ctx context.Context) *redis.StatusCmd {
    return r.client.ScriptKill(ctx)
}

// ScriptLoad loads a script
func (r *RedisClient) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
    return r.client.ScriptLoad(ctx, script)
}

// Ping tests connectivity to Redis
func (r *RedisClient) Ping(ctx context.Context) *redis.StatusCmd {
    return r.client.Ping(ctx)
}

// Info gets Redis server information
func (r *RedisClient) Info(ctx context.Context, section ...string) *redis.StringCmd {
    return r.client.Info(ctx, section...)
}
