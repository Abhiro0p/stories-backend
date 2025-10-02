package handlers

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/storage"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
    db          *storage.PostgresDB
    redisClient *storage.RedisClient
    logger      *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *storage.PostgresDB, redisClient *storage.RedisClient, logger *zap.Logger) *HealthHandler {
    return &HealthHandler{
        db:          db,
        redisClient: redisClient,
        logger:      logger.With(zap.String("handler", "health")),
    }
}

// Health performs basic health check
func (h *HealthHandler) Health(c *gin.Context) {
    status := "ok"
    statusCode := http.StatusOK

    // Basic health response
    response := gin.H{
        "status":    status,
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "service":   "stories-backend",
        "version":   "1.0.0", // TODO: Get from build info
    }

    // If detailed query parameter is provided, include dependency checks
    if c.Query("detailed") == "true" {
        dependencies := h.checkDependencies(c.Request.Context())
        response["dependencies"] = dependencies

        // Check if any dependency is down
        for _, dep := range dependencies {
            if dep["status"] != "ok" {
                status = "degraded"
                statusCode = http.StatusServiceUnavailable
                response["status"] = status
                break
            }
        }
    }

    c.JSON(statusCode, response)
}

// Ready checks if the service is ready to accept traffic
func (h *HealthHandler) Ready(c *gin.Context) {
    dependencies := h.checkDependencies(c.Request.Context())
    
    // Service is ready only if all critical dependencies are healthy
    ready := true
    for _, dep := range dependencies {
        if dep["critical"].(bool) && dep["status"] != "ok" {
            ready = false
            break
        }
    }

    status := "ready"
    statusCode := http.StatusOK
    if !ready {
        status = "not_ready"
        statusCode = http.StatusServiceUnavailable
    }

    c.JSON(statusCode, gin.H{
        "status":       status,
        "timestamp":    time.Now().UTC().Format(time.RFC3339),
        "dependencies": dependencies,
    })
}

// Live checks if the service is alive (for liveness probes)
func (h *HealthHandler) Live(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":    "alive",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    })
}

// checkDependencies checks the health of all dependencies - FIXED: Added ctx parameter
func (h *HealthHandler) checkDependencies(ctx context.Context) []gin.H {
    dependencies := []gin.H{}

    // Check PostgreSQL
    dbStatus := h.checkDatabase(ctx)
    dependencies = append(dependencies, gin.H{
        "name":     "postgresql",
        "status":   dbStatus["status"],
        "critical": true,
        "details":  dbStatus,
    })

    // Check Redis
    redisStatus := h.checkRedis(ctx)
    dependencies = append(dependencies, gin.H{
        "name":     "redis",
        "status":   redisStatus["status"],
        "critical": true,
        "details":  redisStatus,
    })

    return dependencies
}

// checkDatabase checks PostgreSQL connection and basic functionality - FIXED: Added ctx parameter
func (h *HealthHandler) checkDatabase(ctx context.Context) gin.H {
    start := time.Now()
    
    // Check basic connectivity - FIXED: Use ctx parameter instead of c.Request.Context()
    err := h.db.Health(ctx)
    duration := time.Since(start)

    if err != nil {
        h.logger.Error("Database health check failed", zap.Error(err))
        return gin.H{
            "status":      "error",
            "error":       err.Error(),
            "duration_ms": duration.Milliseconds(),
        }
    }

    // Get connection stats
    stats := h.db.GetStats()

    return gin.H{
        "status":              "ok",
        "duration_ms":         duration.Milliseconds(),
        "open_connections":    stats.OpenConnections,
        "in_use":              stats.InUse,
        "idle":                stats.Idle,
        "wait_count":          stats.WaitCount,
        "wait_duration":       stats.WaitDuration.String(),
        "max_idle_closed":     stats.MaxIdleClosed,
        "max_idle_time_closed": stats.MaxIdleTimeClosed,
        "max_lifetime_closed": stats.MaxLifetimeClosed,
    }
}

// checkRedis checks Redis connection and basic functionality - FIXED
func (h *HealthHandler) checkRedis(ctx context.Context) gin.H {
    start := time.Now()
    
    // ✅ FIXED: Use GetClient() to access underlying Redis client methods
    client := h.redisClient.GetClient()
    
    // Ping Redis
    err := client.Ping(ctx).Err()
    duration := time.Since(start)

    if err != nil {
        h.logger.Error("Redis health check failed", zap.Error(err))
        return gin.H{
            "status":      "error",
            "error":       err.Error(),
            "duration_ms": duration.Milliseconds(),
        }
    }

    // Get Redis info - basic info only to avoid too much data
    infoCmd := client.Info(ctx, "server")
    infoResult := ""
    if infoCmd.Err() == nil {
        infoResult = infoCmd.Val()
    }
    
    return gin.H{
        "status":      "ok",
        "duration_ms": duration.Milliseconds(),
        "connected":   true,
        "info":        infoResult, // Basic server info
    }
}

// Metrics returns basic service metrics
func (h *HealthHandler) Metrics(c *gin.Context) {
    // Get database metrics
    dbStats := h.db.GetStats()
    
    // Get basic Redis status
    ctx := c.Request.Context()
    client := h.redisClient.GetClient()
    redisConnected := client.Ping(ctx).Err() == nil
    
    metrics := gin.H{
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "database": gin.H{
            "open_connections": dbStats.OpenConnections,
            "in_use":          dbStats.InUse,
            "idle":            dbStats.Idle,
            "wait_count":      dbStats.WaitCount,
            "wait_duration":   dbStats.WaitDuration.String(),
        },
        "redis": gin.H{
            "connected": redisConnected,
        },
        "uptime": gin.H{
            "started_at": time.Now().Add(-1 * time.Hour).Format(time.RFC3339), // Placeholder
        },
    }

    c.JSON(http.StatusOK, metrics)
}

// Version returns service version information
func (h *HealthHandler) Version(c *gin.Context) {
    // These would typically be set via build flags
    c.JSON(http.StatusOK, gin.H{
        "service":    "stories-backend",
        "version":    "1.0.0",
        "commit":     "unknown",
        "build_time": "unknown",
        "go_version": "unknown",
    })
}

// Status returns detailed service status (combines health + metrics)
func (h *HealthHandler) Status(c *gin.Context) {
    ctx := c.Request.Context()
    dependencies := h.checkDependencies(ctx)
    dbStats := h.db.GetStats()
    
    // Determine overall status
    status := "healthy"
    for _, dep := range dependencies {
        if dep["status"] != "ok" {
            if dep["critical"].(bool) {
                status = "unhealthy"
                break
            } else {
                status = "degraded"
            }
        }
    }
    
    statusCode := http.StatusOK
    if status == "unhealthy" {
        statusCode = http.StatusServiceUnavailable
    }
    
    c.JSON(statusCode, gin.H{
        "service":      "stories-backend",
        "version":      "1.0.0",
        "status":       status,
        "timestamp":    time.Now().UTC().Format(time.RFC3339),
        "dependencies": dependencies,
        "metrics": gin.H{
            "database": gin.H{
                "open_connections": dbStats.OpenConnections,
                "in_use":          dbStats.InUse,
                "idle":            dbStats.Idle,
            },
        },
    })
}
