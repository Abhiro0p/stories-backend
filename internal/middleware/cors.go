package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"

    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(cfg *config.Config) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        origin := c.Request.Header.Get("Origin")

        // Check if CORS is enabled
        if !cfg.CORS.Enabled {
            c.Next()
            return
        }

        // Set default headers
        c.Header("Access-Control-Allow-Credentials", "true")
        c.Header("Access-Control-Max-Age", "86400") // 24 hours

        // Handle allowed origins
        if isAllowedOrigin(origin, cfg.CORS.AllowedOrigins) {
            c.Header("Access-Control-Allow-Origin", origin)
        } else if len(cfg.CORS.AllowedOrigins) == 1 && cfg.CORS.AllowedOrigins[0] == "*" {
            c.Header("Access-Control-Allow-Origin", "*")
        }

        // Set allowed methods
        c.Header("Access-Control-Allow-Methods", strings.Join(cfg.CORS.AllowedMethods, ", "))

        // Set allowed headers
        c.Header("Access-Control-Allow-Headers", strings.Join(cfg.CORS.AllowedHeaders, ", "))

        // Set exposed headers
        if len(cfg.CORS.ExposedHeaders) > 0 {
            c.Header("Access-Control-Expose-Headers", strings.Join(cfg.CORS.ExposedHeaders, ", "))
        }

        // Handle preflight requests
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }

        c.Next()
    })
}

// isAllowedOrigin checks if the origin is in the allowed origins list
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
    for _, allowed := range allowedOrigins {
        if allowed == "*" || allowed == origin {
            return true
        }
        
        // Support for wildcard subdomains
        if strings.HasPrefix(allowed, "*.") {
            domain := strings.TrimPrefix(allowed, "*.")
            if strings.HasSuffix(origin, "."+domain) || origin == domain {
                return true
            }
        }
    }
    return false
}
