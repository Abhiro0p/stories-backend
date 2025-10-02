package middleware

import (
    "time"
    "fmt"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// Logger middleware for structured request logging
func Logger(logger *zap.Logger) gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        // Use zap for structured logging instead of gin's default formatter
        fields := []zap.Field{
            zap.String("method", param.Method),
            zap.String("path", param.Path),
            zap.String("protocol", param.Request.Proto),
            zap.Int("status_code", param.StatusCode),
            zap.Duration("latency", param.Latency),
            zap.String("client_ip", param.ClientIP),
            zap.String("user_agent", param.Request.UserAgent()),
            zap.Int("body_size", param.BodySize),
        }

        // Add error if present
        if param.ErrorMessage != "" {
            fields = append(fields, zap.String("error", param.ErrorMessage))
        }

        // Add request ID if present
        if requestID := param.Request.Header.Get("X-Request-ID"); requestID != "" {
            fields = append(fields, zap.String("request_id", requestID))
        }

        // Log based on status code
        if param.StatusCode >= 500 {
            logger.Error("HTTP Request", fields...)
        } else if param.StatusCode >= 400 {
            logger.Warn("HTTP Request", fields...)
        } else {
            logger.Info("HTTP Request", fields...)
        }

        return ""
    })
}

// RequestID middleware adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = generateRequestID()
        }

        c.Header("X-Request-ID", requestID)
        c.Set("request_id", requestID)
        c.Next()
    }
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
