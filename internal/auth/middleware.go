package auth

import (
    "context"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/Abhiro0p/stories-backend/internal/models"
)

// Define custom types for context keys to avoid collisions - ADDED
type contextKey string

const (
    UserContextKey contextKey = "user"
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

        // Add user to context - FIXED: Use custom context key type
        ctx := context.WithValue(c.Request.Context(), UserContextKey, user)
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

        // Add user to context - FIXED: Use custom context key type
        ctx := context.WithValue(c.Request.Context(), UserContextKey, user)
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

// ActiveUserOnly middleware that requires active user status
func ActiveUserOnly() gin.HandlerFunc {
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

        userModel, ok := user.(*models.User)
        if !ok {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error":   "internal_error",
                "message": "Invalid user context",
            })
            c.Abort()
            return
        }

        if !userModel.IsActive {
            c.JSON(http.StatusForbidden, gin.H{
                "error":   "account_inactive",
                "message": "Account is not active",
            })
            c.Abort()
            return
        }

        c.Next()
    })
}

// VerifiedUserOnly middleware that requires verified user status
func VerifiedUserOnly() gin.HandlerFunc {
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

        userModel, ok := user.(*models.User)
        if !ok {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error":   "internal_error",
                "message": "Invalid user context",
            })
            c.Abort()
            return
        }

        if !userModel.IsVerified {
            c.JSON(http.StatusForbidden, gin.H{
                "error":   "account_not_verified",
                "message": "Account verification required",
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

// GetCurrentUserID helper function to get user ID from Gin context
func GetCurrentUserID(c *gin.Context) (string, bool) {
    userID, exists := c.Get("user_id")
    if !exists {
        return "", false
    }

    id, ok := userID.(string)
    return id, ok
}

// GetUserFromContext gets user from standard context (not Gin context)
func GetUserFromContext(ctx context.Context) (*models.User, bool) {
    user := ctx.Value(UserContextKey)
    if user == nil {
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

// RequireRoles middleware that requires specific roles
func RequireRoles(roles ...string) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        user, ok := GetCurrentUser(c)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Authentication required",
            })
            c.Abort()
            return
        }

        // Check if user has any of the required roles
        hasRole := false
        
        // Check admin role
        if user.IsAdmin {
            for _, role := range roles {
                if role == "admin" {
                    hasRole = true
                    break
                }
            }
        }

        // TODO: Implement more sophisticated role checking if you have a roles system
        // For now, we only support admin role checking
        
        if !hasRole {
            c.JSON(http.StatusForbidden, gin.H{
                "error":   "forbidden",
                "message": "Insufficient privileges",
            })
            c.Abort()
            return
        }

        c.Next()
    })
}

// RequireOwnership middleware that ensures user can only access their own resources - FIXED
func RequireOwnership(resourceUserIDKey string) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        currentUser, ok := GetCurrentUser(c)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Authentication required",
            })
            c.Abort()
            return
        }

        // Get resource user ID from URL params or context - FIXED
        resourceUserID := c.Param(resourceUserIDKey)
        if resourceUserID == "" {
            // ✅ FIXED: Handle multiple return values from c.Get()
            value, exists := c.Get(resourceUserIDKey)
            if exists {
                resourceUserID, _ = value.(string)
            }
        }

        // Allow if user is admin or owns the resource
        if !currentUser.IsAdmin && currentUser.ID.String() != resourceUserID {
            c.JSON(http.StatusForbidden, gin.H{
                "error":   "forbidden",
                "message": "Access denied",
            })
            c.Abort()
            return
        }

        c.Next()
    })
}

// RateLimitByUser middleware that applies rate limiting per user
func RateLimitByUser(limit int, window int) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        user, ok := GetCurrentUser(c)
        if !ok {
            // If no user, continue without rate limiting or apply global rate limiting
            c.Next()
            return
        }

        // Set user-specific identifier for rate limiting
        c.Set("rate_limit_key", "user:"+user.ID.String())

        c.Next()
    })
}

// LogUserActivity middleware that logs user activities
func LogUserActivity(action string) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        user, ok := GetCurrentUser(c)
        if ok {
            // TODO: Implement activity logging
            _ = user
            _ = action
            // log.Info("User activity", "user_id", user.ID, "action", action, "ip", c.ClientIP())
        }

        c.Next()
    })
}
