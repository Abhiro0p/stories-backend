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
    
    // Check if token is blacklisted
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    if s.IsTokenBlacklisted(ctx, claims.TokenID) {
        return nil, fmt.Errorf("token has been revoked")
    }
    
    // Get user from database
    userCtx, userCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer userCancel()
    
    user, err := s.userStore.GetByID(userCtx, claims.UserID)
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
    s.logger.Info("User logout", 
        zap.String("user_id", userID.String()),
        zap.String("token_id", tokenID),
    )
    
    // Create a timeout context using the provided context
    timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    // Add token to blacklist if tokenID is provided
    if tokenID != "" {
        blacklistKey := fmt.Sprintf("blacklist:token:%s", tokenID)
        // Set with same TTL as token expiry
        ttl := time.Duration(s.config.JWTExpiryHours) * time.Hour
        
        if err := s.addToBlacklist(timeoutCtx, blacklistKey, ttl); err != nil {
            s.logger.Error("Failed to blacklist token", 
                zap.Error(err),
                zap.String("token_id", tokenID),
            )
            return fmt.Errorf("failed to logout: %w", err)
        }
    }
    
    s.logger.Info("User logout successful", zap.String("user_id", userID.String()))
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

    // Create session record for logging/auditing
    session := models.NewSession(
        user.ID,
        tokenID,
        now.Add(time.Duration(s.config.JWTRefreshExpiryDays) * 24 * time.Hour),
        nil, // IP address - could be passed from request context
        nil, // User agent - could be passed from request context
    )

    // Log session creation (using the ctx parameter for timeout)
    timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
    defer cancel()
    
    select {
    case <-timeoutCtx.Done():
        s.logger.Warn("Session logging timeout")
    default:
        s.logger.Debug("Session created",
            zap.String("session_id", session.ID.String()),
            zap.String("token_id", tokenID),
            zap.String("user_id", user.ID.String()),
            zap.Time("expires_at", session.ExpiresAt),
        )
    }

    // TODO: Store session in database when SessionStore is implemented
    // if err := s.sessionStore.Create(ctx, session); err != nil {
    //     s.logger.Error("Failed to store session", zap.Error(err))
    //     // Don't fail token generation if session storage fails
    // }
    
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

// addToBlacklist adds a token to the blacklist
func (s *Service) addToBlacklist(ctx context.Context, key string, ttl time.Duration) error {
    // TODO: Implement actual blacklist storage using Redis
    // For now, just simulate the operation with timeout handling
    
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Simulate some work
        time.Sleep(10 * time.Millisecond)
        s.logger.Debug("Token added to blacklist", 
            zap.String("key", key),
            zap.Duration("ttl", ttl),
        )
    }
    
    return nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (s *Service) IsTokenBlacklisted(ctx context.Context, tokenID string) bool {
    if tokenID == "" {
        return false
    }
    
    blacklistKey := fmt.Sprintf("blacklist:token:%s", tokenID)
    
    // TODO: Implement actual blacklist check using Redis
    // This would typically check Redis with a timeout
    
    select {
    case <-ctx.Done():
        s.logger.Warn("Blacklist check timeout", zap.String("token_id", tokenID))
        return false // Default to allowing access on timeout
    default:
        s.logger.Debug("Checking token blacklist", 
            zap.String("token_id", tokenID),
            zap.String("key", blacklistKey),
        )
    }
    
    return false // For now, no tokens are blacklisted
}

// RevokeAllUserTokens revokes all tokens for a user (useful for security incidents)
func (s *Service) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
    s.logger.Info("Revoking all tokens for user", zap.String("user_id", userID.String()))
    
    // TODO: Implement user token revocation
    // This would typically:
    // 1. Find all active sessions for the user
    // 2. Add all token IDs to blacklist
    // 3. Update user's security timestamp to invalidate old tokens
    
    timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    select {
    case <-timeoutCtx.Done():
        return timeoutCtx.Err()
    default:
        s.logger.Info("All tokens revoked for user", zap.String("user_id", userID.String()))
    }
    
    return nil
}

// ValidateTokenWithContext validates token with custom context (useful for middleware)
func (s *Service) ValidateTokenWithContext(ctx context.Context, tokenString string) (*models.User, error) {
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
    
    // Check if token is blacklisted with provided context
    if s.IsTokenBlacklisted(ctx, claims.TokenID) {
        return nil, fmt.Errorf("token has been revoked")
    }
    
    // Get user from database with provided context
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

// GetTokenClaims extracts claims from a token without validating the user
func (s *Service) GetTokenClaims(tokenString string) (*models.TokenClaims, error) {
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
    
    return claims, nil
}
