package auth

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/models"
    "github.com/Abhiro0p/stories-backend/internal/storage"
    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// Service handles authentication operations
type Service struct {
    config    *config.Config
    userStore storage.UserStore
    logger    *zap.Logger
}

// NewService creates a new auth service
func NewService(cfg *config.Config, userStore storage.UserStore, logger *zap.Logger) *Service {
    return &Service{
        config:    cfg,
        userStore: userStore,
        logger:    logger.With(zap.String("component", "auth_service")),
    }
}

// Signup creates a new user account
func (s *Service) Signup(ctx context.Context, req models.UserCreateRequest) (*models.AuthResponse, error) {
    s.logger.Info("Creating new user account", zap.String("email", req.Email))
    
    // Check if user already exists
    existingUser, err := s.userStore.GetByEmail(ctx, req.Email)
    if err == nil && existingUser != nil {
        return nil, fmt.Errorf("user with email %s already exists", req.Email)
    }
    
    // Check username availability
    existingUser, err = s.userStore.GetByUsername(ctx, req.Username)
    if err == nil && existingUser != nil {
        return nil, fmt.Errorf("username %s is already taken", req.Username)
    }
    
    // Create new user
    user, err := models.NewUser(req.Email, req.Username, req.Password, req.FullName)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    // Save user to database
    if err := s.userStore.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("failed to save user: %w", err)
    }
    
    s.logger.Info("User created successfully", 
        zap.String("user_id", user.ID.String()),
        zap.String("username", user.Username),
    )
    
    // Generate tokens
    return s.generateTokens(ctx, user)
}

// Login authenticates a user with email/password
func (s *Service) Login(ctx context.Context, req models.AuthRequest) (*models.AuthResponse, error) {
    s.logger.Info("User login attempt", zap.String("email", req.Email))
    
    // Get user by email
    user, err := s.userStore.GetByEmail(ctx, req.Email)
    if err != nil {
        s.logger.Warn("Login failed - user not found", zap.String("email", req.Email))
        return nil, fmt.Errorf("invalid email or password")
    }
    
    // Check if user is active
    if !user.IsActive {
        s.logger.Warn("Login failed - user inactive", zap.String("user_id", user.ID.String()))
        return nil, fmt.Errorf("account is disabled")
    }
    
    // Verify password
    if !user.CheckPassword(req.Password) {
        s.logger.Warn("Login failed - invalid password", zap.String("user_id", user.ID.String()))
        return nil, fmt.Errorf("invalid email or password")
    }
    
    s.logger.Info("User login successful", 
        zap.String("user_id", user.ID.String()),
        zap.String("username", user.Username),
    )
    
    // Generate tokens
    return s.generateTokens(ctx, user)
}

// RefreshToken generates new tokens using refresh token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
    s.logger.Debug("Token refresh attempt")
    
    // Parse and validate refresh token
    claims, err := s.validateRefreshToken(refreshToken)
    if err != nil {
        s.logger.Warn("Invalid refresh token", zap.Error(err))
        return nil, fmt.Errorf("invalid refresh token")
    }
    
    // Get user
    user, err := s.userStore.GetByID(ctx, claims.UserID)
    if err != nil {
        s.logger.Warn("User not found for refresh token", zap.String("user_id", claims.UserID.String()))
        return nil, fmt.Errorf("user not found")
    }
    
    // Check if user is still active
    if !user.IsActive {
        s.logger.Warn("Refresh failed - user inactive", zap.String("user_id", user.ID.String()))
        return nil, fmt.Errorf("account is disabled")
    }
    
    s.logger.Info("Token refresh successful", zap.String("user_id", user.ID.String()))
    
    // Generate new tokens
    return s.generateTokens(ctx, user)
}

// ValidateToken validates and returns user from JWT token
func (s *Service) ValidateToken(tokenString string) (*models.User, error) {
    // Parse token
    token, err := jwt.ParseWithClaims(tokenString, &models.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(s.config.JWTSecret), nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }
    
    claims, ok := token.Claims.(*models.TokenClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token claims")
    }
    
    // Get user from database
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    user, err := s.userStore.GetByID(ctx, claims.UserID)
    if err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    
    // Check if user is active
    if !user.IsActive {
        return nil, fmt.Errorf("account is disabled")
    }
    
    return user, nil
}

// Logout invalidates user session
func (s *Service) Logout(ctx context.Context, userID uuid.UUID, tokenID string) error {
    s.logger.Info("User logout", zap.String("user_id", userID.String()))
    
    // TODO: Add token blacklisting logic here
    // For now, we rely on token expiration
    
    return nil
}

// generateTokens creates access and refresh tokens for user
func (s *Service) generateTokens(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
    tokenID := generateTokenID()
    now := time.Now()
    
    // Create access token claims
    accessClaims := &models.TokenClaims{
        UserID:   user.ID,
        Email:    user.Email,
        Username: user.Username,
        TokenID:  tokenID,
        RegisteredClaims: jwt.RegisteredClaims{
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.config.JWTExpiryHours) * time.Hour)),
            NotBefore: jwt.NewNumericDate(now),
            Issuer:    s.config.JWTIssuer,
            Audience:  jwt.ClaimStrings{s.config.JWTAudience},
        },
    }
    
    // Generate access token
    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessTokenString, err := accessToken.SignedString([]byte(s.config.JWTSecret))
    if err != nil {
        return nil, fmt.Errorf("failed to sign access token: %w", err)
    }
    
    // Create refresh token claims
    refreshClaims := &models.RefreshTokenClaims{
        UserID:  user.ID,
        TokenID: tokenID,
        RegisteredClaims: jwt.RegisteredClaims{
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.config.JWTRefreshExpiryDays) * 24 * time.Hour)),
            NotBefore: jwt.NewNumericDate(now),
            Issuer:    s.config.JWTIssuer,
            Audience:  jwt.ClaimStrings{s.config.JWTAudience},
        },
    }
    
    // Generate refresh token
    refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshTokenString, err := refreshToken.SignedString([]byte(s.config.JWTRefreshSecret))
    if err != nil {
        return nil, fmt.Errorf("failed to sign refresh token: %w", err)
    }
    
    return &models.AuthResponse{
        AccessToken:  accessTokenString,
        RefreshToken: refreshTokenString,
        TokenType:    "Bearer",
        ExpiresIn:    int64(time.Duration(s.config.JWTExpiryHours) * time.Hour / time.Second),
        User:         user.ToResponse(),
    }, nil
}

// validateRefreshToken validates refresh token and returns claims
func (s *Service) validateRefreshToken(tokenString string) (*models.RefreshTokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &models.RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(s.config.JWTRefreshSecret), nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("invalid refresh token: %w", err)
    }
    
    claims, ok := token.Claims.(*models.RefreshTokenClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid refresh token claims")
    }
    
    return claims, nil
}

// generateTokenID generates a unique token ID
func generateTokenID() string {
    bytes := make([]byte, 16)
    if _, err := rand.Read(bytes); err != nil {
        // Fallback to UUID if random fails
        return uuid.New().String()
    }
    return hex.EncodeToString(bytes)
}

// ChangePassword changes user password
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
    s.logger.Info("Password change request", zap.String("user_id", userID.String()))
    
    // Get user
    user, err := s.userStore.GetByID(ctx, userID)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }
    
    // Verify current password
    if !user.CheckPassword(currentPassword) {
        s.logger.Warn("Password change failed - invalid current password", 
            zap.String("user_id", userID.String()))
        return fmt.Errorf("current password is incorrect")
    }
    
    // Update password
    user.UpdatePassword(newPassword)
    
    // Save to database
    if err := s.userStore.Update(ctx, user); err != nil {
        return fmt.Errorf("failed to update password: %w", err)
    }
    
    s.logger.Info("Password changed successfully", zap.String("user_id", userID.String()))
    return nil
}

// GetUserFromContext extracts user from request context
func GetUserFromContext(ctx context.Context) (*models.User, bool) {
    user, ok := ctx.Value("user").(*models.User)
    return user, ok
}
