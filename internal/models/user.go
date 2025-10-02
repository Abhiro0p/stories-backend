package models

import (
    "crypto/rand"
    "crypto/subtle"
    "encoding/base64"
    "strings"
    "time"

    "github.com/google/uuid"
    "golang.org/x/crypto/argon2"
)

// User represents a user in the system
type User struct {
    ID               uuid.UUID  `json:"id" db:"id"`
    Email            string     `json:"email" db:"email"`
    Username         string     `json:"username" db:"username"`
    PasswordHash     string     `json:"-" db:"password_hash"`
    FullName         *string    `json:"full_name" db:"full_name"`
    Bio              *string    `json:"bio" db:"bio"`
    ProfilePicture   *string    `json:"profile_picture" db:"profile_picture"`
    IsActive         bool       `json:"is_active" db:"is_active"`
    IsVerified       bool       `json:"is_verified" db:"is_verified"`
    IsAdmin          bool       `json:"is_admin" db:"is_admin"`
    FollowerCount    int        `json:"follower_count" db:"follower_count"`
    FollowingCount   int        `json:"following_count" db:"following_count"`
    StoryCount       int        `json:"story_count" db:"story_count"`
    CreatedAt        time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
    DeletedAt        *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
    
    // Additional fields not stored in DB
    IsFollowing      bool       `json:"is_following,omitempty" db:"-"`
    LastActiveAt     *time.Time `json:"last_active_at,omitempty" db:"-"`
}

// UserCreateRequest represents the request to create a new user
type UserCreateRequest struct {
    Email    string `json:"email" validate:"required,email,max=255"`
    Username string `json:"username" validate:"required,username,min=3,max=30"`
    Password string `json:"password" validate:"required,strong_password,min=8,max=100"`
    FullName string `json:"full_name" validate:"required,min=1,max=100"`
}

// UserUpdateRequest represents the request to update user information
type UserUpdateRequest struct {
    Username       *string `json:"username,omitempty" validate:"omitempty,username,min=3,max=30"`
    FullName       *string `json:"full_name,omitempty" validate:"omitempty,min=1,max=100"`
    Bio            *string `json:"bio,omitempty" validate:"omitempty,max=500"`
    ProfilePicture *string `json:"profile_picture,omitempty" validate:"omitempty,url,max=255"`
}

// UserResponse represents the user data returned in API responses
type UserResponse struct {
    ID             uuid.UUID  `json:"id"`
    Email          string     `json:"email"`
    Username       string     `json:"username"`
    FullName       *string    `json:"full_name"`
    Bio            *string    `json:"bio"`
    ProfilePicture *string    `json:"profile_picture"`
    IsVerified     bool       `json:"is_verified"`
    FollowerCount  int        `json:"follower_count"`
    FollowingCount int        `json:"following_count"`
    StoryCount     int        `json:"story_count"`
    CreatedAt      time.Time  `json:"created_at"`
    IsFollowing    bool       `json:"is_following,omitempty"`
    LastActiveAt   *time.Time `json:"last_active_at,omitempty"`
}

// UserStats represents user statistics
type UserStats struct {
    FollowerCount  int `json:"follower_count" db:"follower_count"`
    FollowingCount int `json:"following_count" db:"following_count"`
    StoryCount     int `json:"story_count" db:"story_count"`
}

// NewUser creates a new user with hashed password
func NewUser(email, username, password, fullName string) (*User, error) {
    id := uuid.New()
    
    // Hash password
    hashedPassword, err := hashPassword(password)
    if err != nil {
        return nil, err
    }
    
    now := time.Now()
    
    return &User{
        ID:           id,
        Email:        email,
        Username:     username,
        PasswordHash: hashedPassword,
        FullName:     &fullName,
        IsActive:     true,
        IsVerified:   false,
        IsAdmin:      false,
        CreatedAt:    now,
        UpdatedAt:    now,
    }, nil
}

// CheckPassword verifies if the provided password matches the stored hash
func (u *User) CheckPassword(password string) bool {
    return verifyPassword(password, u.PasswordHash)
}

// UpdatePassword updates the user's password
func (u *User) UpdatePassword(password string) error {
    hashedPassword, err := hashPassword(password)
    if err != nil {
        return err
    }
    
    u.PasswordHash = hashedPassword
    u.UpdatedAt = time.Now()
    return nil
}

// Update updates user fields from request
func (u *User) Update(req UserUpdateRequest) {
    if req.Username != nil {
        u.Username = *req.Username
    }
    if req.FullName != nil {
        u.FullName = req.FullName
    }
    if req.Bio != nil {
        u.Bio = req.Bio
    }
    if req.ProfilePicture != nil {
        u.ProfilePicture = req.ProfilePicture
    }
    u.UpdatedAt = time.Now()
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() *UserResponse {
    return &UserResponse{
        ID:             u.ID,
        Email:          u.Email,
        Username:       u.Username,
        FullName:       u.FullName,
        Bio:            u.Bio,
        ProfilePicture: u.ProfilePicture,
        IsVerified:     u.IsVerified,
        FollowerCount:  u.FollowerCount,
        FollowingCount: u.FollowingCount,
        StoryCount:     u.StoryCount,
        CreatedAt:      u.CreatedAt,
        IsFollowing:    u.IsFollowing,
        LastActiveAt:   u.LastActiveAt,
    }
}

// hashPassword creates a hash of the password using Argon2
func hashPassword(password string) (string, error) {
    // Generate a random salt
    salt := make([]byte, 16)
    if _, err := rand.Read(salt); err != nil {
        return "", err
    }
    
    // Hash the password
    hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    
    // Encode salt and hash
    encoded := base64.RawStdEncoding.EncodeToString(salt) + ":" + base64.RawStdEncoding.EncodeToString(hash)
    return encoded, nil
}

// verifyPassword verifies a password against its hash
func verifyPassword(password, encoded string) bool {
    parts := strings.Split(encoded, ":")
    if len(parts) != 2 {
        return false
    }
    
    salt, err := base64.RawStdEncoding.DecodeString(parts[0])
    if err != nil {
        return false
    }
    
    hash, err := base64.RawStdEncoding.DecodeString(parts[1])
    if err != nil {
        return false
    }
    
    // Hash the provided password with the same salt
    computedHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    
    // Compare hashes
    return subtle.ConstantTimeCompare(hash, computedHash) == 1
}
