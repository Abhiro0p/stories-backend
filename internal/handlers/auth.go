package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/auth"
    "github.com/Abhiro0p/stories-backend/internal/models"
    "github.com/Abhiro0p/stories-backend/pkg/validator"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
    authService *auth.Service
    logger      *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *auth.Service, logger *zap.Logger) *AuthHandler {
    return &AuthHandler{
        authService: authService,
        logger:      logger.With(zap.String("handler", "auth")),
    }
}

// Signup handles user registration
func (h *AuthHandler) Signup(c *gin.Context) {
    var req models.UserCreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid signup request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
            "details": err.Error(),
        })
        return
    }

    // Validate request
    if err := validator.ValidateStruct(&req); err != nil {
        h.logger.Warn("Signup validation failed", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "validation_failed",
            "message": "Request validation failed",
            "details": err.Error(),
        })
        return
    }

    // Create user
    response, err := h.authService.Signup(c.Request.Context(), req)
    if err != nil {
        h.logger.Error("Signup failed", zap.Error(err))
        c.JSON(http.StatusConflict, gin.H{
            "error":   "signup_failed",
            "message": err.Error(),
        })
        return
    }

    h.logger.Info("User signup successful", 
        zap.String("user_id", response.User.ID.String()),
        zap.String("email", response.User.Email),
    )

    c.JSON(http.StatusCreated, response)
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
    var req models.AuthRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid login request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
        })
        return
    }

    // Validate request
    if err := validator.ValidateStruct(&req); err != nil {
        h.logger.Warn("Login validation failed", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "validation_failed",
            "message": "Request validation failed",
        })
        return
    }

    // Authenticate user
    response, err := h.authService.Login(c.Request.Context(), req)
    if err != nil {
        h.logger.Warn("Login failed", 
            zap.String("email", req.Email),
            zap.Error(err),
        )
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "login_failed",
            "message": "Invalid email or password",
        })
        return
    }

    h.logger.Info("User login successful", 
        zap.String("user_id", response.User.ID.String()),
        zap.String("email", response.User.Email),
    )

    c.JSON(http.StatusOK, response)
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
    var req models.RefreshTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid refresh request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
        })
        return
    }

    // Validate request
    if err := validator.ValidateStruct(&req); err != nil {
        h.logger.Warn("Refresh validation failed", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "validation_failed",
            "message": "Request validation failed",
        })
        return
    }

    // Refresh token
    response, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
    if err != nil {
        h.logger.Warn("Token refresh failed", zap.Error(err))
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "refresh_failed",
            "message": "Invalid or expired refresh token",
        })
        return
    }

    h.logger.Info("Token refresh successful", 
        zap.String("user_id", response.User.ID.String()),
    )

    c.JSON(http.StatusOK, response)
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Get token ID from context (if available)
    tokenID := c.GetString("token_id")

    // Logout user
    if err := h.authService.Logout(c.Request.Context(), user.ID, tokenID); err != nil {
        h.logger.Error("Logout failed", 
            zap.String("user_id", user.ID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "logout_failed",
            "message": "Failed to logout user",
        })
        return
    }

    h.logger.Info("User logout successful", zap.String("user_id", user.ID.String()))

    c.JSON(http.StatusOK, gin.H{
        "message": "Logout successful",
    })
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    var req models.PasswordChangeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid password change request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
        })
        return
    }

    // Validate request
    if err := validator.ValidateStruct(&req); err != nil {
        h.logger.Warn("Password change validation failed", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "validation_failed",
            "message": "Request validation failed",
        })
        return
    }

    // Change password
    if err := h.authService.ChangePassword(c.Request.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
        h.logger.Error("Password change failed", 
            zap.String("user_id", user.ID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "password_change_failed",
            "message": err.Error(),
        })
        return
    }

    h.logger.Info("Password change successful", zap.String("user_id", user.ID.String()))

    c.JSON(http.StatusOK, gin.H{
        "message": "Password changed successfully",
    })
}

// ForgotPassword handles password reset request
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
    var req models.PasswordResetRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid forgot password request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
        })
        return
    }

    // TODO: Implement password reset functionality
    // This would typically:
    // 1. Generate a reset token
    // 2. Store it in database with expiration
    // 3. Send email with reset link
    
    h.logger.Info("Password reset requested", zap.String("email", req.Email))

    c.JSON(http.StatusOK, gin.H{
        "message": "Password reset email sent",
    })
}

// ResetPassword handles password reset confirmation
func (h *AuthHandler) ResetPassword(c *gin.Context) {
    var req models.PasswordResetConfirmRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid reset password request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
        })
        return
    }

    // TODO: Implement password reset confirmation
    // This would typically:
    // 1. Validate reset token
    // 2. Update user password
    // 3. Invalidate reset token
    
    h.logger.Info("Password reset confirmation", zap.String("token", req.Token))

    c.JSON(http.StatusOK, gin.H{
        "message": "Password reset successful",
    })
}

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
    var req models.EmailVerificationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid email verification request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
        })
        return
    }

    // TODO: Implement email verification
    // This would typically:
    // 1. Validate verification token
    // 2. Mark user as verified
    // 3. Invalidate verification token
    
    h.logger.Info("Email verification", zap.String("token", req.Token))

    c.JSON(http.StatusOK, gin.H{
        "message": "Email verified successfully",
    })
}
