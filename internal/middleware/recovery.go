package middleware

import (
    "net/http"
    "runtime/debug"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// Recovery middleware handles panics and recovers gracefully
func Recovery(logger *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                // Log the panic with stack trace
                logger.Error("Panic recovered",
                    zap.Any("error", err),
                    zap.String("path", c.Request.URL.Path),
                    zap.String("method", c.Request.Method),
                    zap.String("client_ip", c.ClientIP()),
                    zap.String("user_agent", c.Request.UserAgent()),
                    zap.String("stack", string(debug.Stack())),
                )

                // Return error response
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error":   "internal_server_error",
                    "message": "An unexpected error occurred",
                })

                c.Abort()
            }
        }()

        c.Next()
    }
}

// RecoveryWithWriter returns a Recovery middleware that writes to a custom writer
func RecoveryWithWriter(logger *zap.Logger) gin.HandlerFunc {
    return Recovery(logger)
}
