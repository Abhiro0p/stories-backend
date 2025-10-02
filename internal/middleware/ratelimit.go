package middleware

import (
    "context"
    "fmt"
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/storage"
    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// RateLimit middleware implements rate limiting using Redis
func RateLimit(redisClient *storage.RedisClient, cfg config.RateLimitConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !cfg.Enabled {
            c.Next()
            return
        }

        // Get client identifier (IP address by default)
        clientIP := c.ClientIP()
        
        // Use user ID if authenticated
        if userID, exists := c.Get("user_id"); exists {
            clientIP = fmt.Sprintf("user:%v", userID)
        }

        // Create rate limit key
        key := fmt.Sprintf("rate_limit:%s:%s", clientIP, c.FullPath())

        // Check rate limit
        allowed, resetTime, err := checkRateLimit(c.Request.Context(), redisClient, key, cfg)
        if err != nil {
            // Log error but don't block request if Redis is down
            zap.L().Error("Rate limit check failed", zap.Error(err))
            c.Next()
            return
        }

        // Set rate limit headers
        c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerMinute))
        c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))

        if !allowed {
            c.Header("Retry-After", strconv.FormatInt(resetTime-time.Now().Unix(), 10))
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error":   "rate_limit_exceeded",
                "message": "Too many requests. Please try again later.",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

// checkRateLimit implements sliding window rate limiting
func checkRateLimit(ctx context.Context, client *storage.RedisClient, key string, cfg config.RateLimitConfig) (bool, int64, error) {
    now := time.Now().Unix()
    window := int64(60) // 1 minute window
    
    // Use Lua script for atomic operations
    luaScript := `
        local key = KEYS[1]
        local window = tonumber(ARGV[1])
        local limit = tonumber(ARGV[2])
        local now = tonumber(ARGV[3])
        local burst = tonumber(ARGV[4])
        
        -- Remove expired entries
        redis.call('zremrangebyscore', key, 0, now - window)
        
        -- Count current requests
        local current = redis.call('zcard', key)
        
        -- Check if within limit (including burst)
        if current < (limit + burst) then
            -- Add current request
            redis.call('zadd', key, now, now)
            redis.call('expire', key, window)
            return {1, now + window}
        else
            -- Rate limit exceeded
            local oldest = redis.call('zrange', key, 0, 0, 'WITHSCORES')
            local reset_time = now + window
            if #oldest > 0 then
                reset_time = tonumber(oldest[2]) + window
            end
            return {0, reset_time}
        end
    `

    result, err := client.Eval(ctx, luaScript, []string{key}, window, cfg.RequestsPerMinute, now, cfg.Burst).Result()
    if err != nil {
        return false, 0, err
    }

    resultSlice := result.([]interface{})
    allowed := resultSlice[0].(int64) == 1
    resetTime := resultSlice[1].(int64)

    return allowed, resetTime, nil
}

// EndpointRateLimit creates endpoint-specific rate limiting
func EndpointRateLimit(redisClient *storage.RedisClient, requestsPerMinute int, burst int) gin.HandlerFunc {
    cfg := config.RateLimitConfig{
        Enabled:           true,
        RequestsPerMinute: requestsPerMinute,
        Burst:             burst,
    }

    return RateLimit(redisClient, cfg)
}

// AuthRateLimit applies specific rate limiting for auth endpoints
func AuthRateLimit(redisClient *storage.RedisClient) gin.HandlerFunc {
    return EndpointRateLimit(redisClient, 5, 2) // 5 requests per minute with 2 burst
}

// MediaRateLimit applies specific rate limiting for media endpoints
func MediaRateLimit(redisClient *storage.RedisClient) gin.HandlerFunc {
    return EndpointRateLimit(redisClient, 10, 5) // 10 requests per minute with 5 burst
}
