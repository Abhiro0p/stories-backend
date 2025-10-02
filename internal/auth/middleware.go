package auth

import (
    "context"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// RequireAuth middleware that validates JWT tokens
func RequireAuth(authService *Service) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        // Get token from Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Authorization header is required",
            })
            c.Abort()
            return
        }

        // Extract token from "Bearer <token>" format
        tokenParts := strings.Split(authHeader, " ")
        if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized", 
                "message": "Invalid authorization header format",
            })
            c.Abort()
            return
        }

        token := tokenParts[1]

        // Validate token and get user
        user, err := authService.ValidateToken(token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Invalid or expired token",
            })
            c.Abort()
            return
        }

        // Add user to context
        ctx := context.WithValue(c.Request.Context(), "user", user)
        c.Request = c.Request.WithContext(ctx)

        // Set user in Gin context as well for easier access
        c.Set("user", user)
        c.Set("user_id", user.ID)

        c.Next()
    })
}

// OptionalAuth middleware that adds user to context if token is present
func OptionalAuth(authService *Service) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        // Get token from Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.Next()
            return
        }

        // Extract token from "Bearer <token>" format  
        tokenParts := strings.Split(authHeader, " ")
        if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
            c.Next()
            return
        }

        token := tokenParts[1]

        // Validate token and get user
        user, err := authService.ValidateToken(token)
        if err != nil {
            // Invalid token, but don't abort - continue without user
            c.Next()
            return
        }

        // Add user to context
        ctx := context.WithValue(c.Request.Context(), "user", user)
        c.Request = c.Request.WithContext(ctx)

        // Set user in Gin context
        c.Set("user", user)
        c.Set("user_id", user.ID)

        c.Next()
    })
}

// AdminOnly middleware that requires admin role
func AdminOnly() gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        user, exists := c.Get("user")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Authentication required",
            })
            c.Abort()
            return
        }

        // Type assertion
        userModel, ok := user.(*models.User)
        if !ok {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error":   "internal_error",
                "message": "Invalid user context",
            })
            c.Abort()
            return
        }

        // Check if user has admin role
        // Note: This assumes you have a Role field in your User model
        // You may need to implement role-based access control based on your needs
        if !userModel.IsAdmin {
            c.JSON(http.StatusForbidden, gin.H{
                "error":   "forbidden",
                "message": "Admin access required",
            })
            c.Abort()
            return
        }

        c.Next()
    })
}

// GetCurrentUser helper function to get user from Gin context
func GetCurrentUser(c *gin.Context) (*models.User, bool) {
    user, exists := c.Get("user")
    if !exists {
        return nil, false
    }

    userModel, ok := user.(*models.User)
    return userModel, ok
}

// MustGetCurrentUser helper function that panics if user not found
func MustGetCurrentUser(c *gin.Context) *models.User {
    user, ok := GetCurrentUser(c)
    if !ok {
        panic("user not found in context")
    }
    return user
}
