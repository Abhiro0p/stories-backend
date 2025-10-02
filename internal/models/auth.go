package models

import (
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

// TokenClaims represents the claims in a JWT access token
type TokenClaims struct {
    UserID   uuid.UUID `json:"user_id"`
    Email    string    `json:"email"`
    Username string    `json:"username"`
    TokenID  string    `json:"token_id"`
    jwt.RegisteredClaims
}

// RefreshTokenClaims represents the claims in a JWT refresh token
type RefreshTokenClaims struct {
    UserID  uuid.UUID `json:"user_id"`
    TokenID string    `json:"token_id"`
    jwt.RegisteredClaims
}

// Session represents a user session
type Session struct {
    ID        uuid.UUID  `json:"id" db:"id"`
    UserID    uuid.UUID  `json:"user_id" db:"user_id"`
    TokenID   string     `json:"token_id" db:"token_id"`
    ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
    CreatedAt time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
    RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
    IPAddress *string    `json:"ip_address,omitempty" db:"ip_address"`
    UserAgent *string    `json:"user_agent,omitempty" db:"user_agent"`
}

// AuthRequest represents login request
type AuthRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
    AccessToken  string       `json:"access_token"`
    RefreshToken string       `json:"refresh_token"`
    TokenType    string       `json:"token_type"`
    ExpiresIn    int64        `json:"expires_in"`
    User         *UserResponse `json:"user"`
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token" validate:"required"`
}

// ForgotPasswordRequest represents a forgot password request
type ForgotPasswordRequest struct {
    Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents a reset password request
type ResetPasswordRequest struct {
    Token       string `json:"token" validate:"required"`
    NewPassword string `json:"new_password" validate:"required,strong_password,min=8,max=100"`
}

// ChangePasswordRequest represents a change password request
type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password" validate:"required"`
    NewPassword     string `json:"new_password" validate:"required,strong_password,min=8,max=100"`
}

// EmailVerificationRequest represents an email verification request
type EmailVerificationRequest struct {
    Token string `json:"token" validate:"required"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
    RefreshToken string `json:"refresh_token,omitempty"`
}

// NewSession creates a new session
func NewSession(userID uuid.UUID, tokenID string, expiresAt time.Time, ipAddress, userAgent *string) *Session {
    now := time.Now()
    
    return &Session{
        ID:        uuid.New(),
        UserID:    userID,
        TokenID:   tokenID,
        ExpiresAt: expiresAt,
        CreatedAt: now,
        UpdatedAt: now,
        IPAddress: ipAddress,
        UserAgent: userAgent,
    }
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
    return time.Now().After(s.ExpiresAt)
}

// IsRevoked checks if the session has been revoked
func (s *Session) IsRevoked() bool {
    return s.RevokedAt != nil
}

// IsValid checks if the session is valid (not expired and not revoked)
func (s *Session) IsValid() bool {
    return !s.IsExpired() && !s.IsRevoked()
}

// Revoke revokes the session
func (s *Session) Revoke() {
    now := time.Now()
    s.RevokedAt = &now
    s.UpdatedAt = now
}

// Refresh updates the session's updated timestamp
func (s *Session) Refresh() {
    s.UpdatedAt = time.Now()
}

// GetTimeRemaining returns the time remaining before expiry
func (s *Session) GetTimeRemaining() time.Duration {
    if s.IsExpired() {
        return 0
    }
    return time.Until(s.ExpiresAt)
}
