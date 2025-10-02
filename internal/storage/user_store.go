package storage

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/models"
)

// UserStoreImpl implements UserStore interface
type UserStoreImpl struct {
    db          *sqlx.DB
    redisClient *RedisClient
    logger      *zap.Logger
}

// NewUserStore creates a new user store
func NewUserStore(db *sqlx.DB, redisClient *RedisClient, logger *zap.Logger) UserStore {
    return &UserStoreImpl{
        db:          db,
        redisClient: redisClient,
        logger:      logger.With(zap.String("store", "user")),
    }
}

// Create creates a new user
func (s *UserStoreImpl) Create(ctx context.Context, user *models.User) error {
    query := `
        INSERT INTO users (
            id, email, username, password_hash, full_name, bio, 
            profile_picture, is_active, is_verified, is_admin,
            created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
        )`

    _, err := s.db.ExecContext(ctx, query,
        user.ID, user.Email, user.Username, user.PasswordHash,
        user.FullName, user.Bio, user.ProfilePicture, user.IsActive,
        user.IsVerified, user.IsAdmin, user.CreatedAt, user.UpdatedAt,
    )

    if err != nil {
        s.logger.Error("Failed to create user", zap.Error(err))
        return fmt.Errorf("failed to create user: %w", err)
    }

    s.logger.Info("User created", zap.String("user_id", user.ID.String()))
    return nil
}

// GetByID gets a user by ID with caching
func (s *UserStoreImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
    cacheKey := fmt.Sprintf("user:%s", id.String())
    var user models.User

    if err := s.redisClient.Get(ctx, cacheKey, &user); err == nil {
        return &user, nil
    }

    query := `
        SELECT id, email, username, password_hash, full_name, bio,
               profile_picture, is_active, is_verified, is_admin,
               follower_count, following_count, story_count,
               created_at, updated_at, deleted_at
        FROM users 
        WHERE id = $1 AND deleted_at IS NULL`

    err := s.db.GetContext(ctx, &user, query, id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    // Cache the result
    s.redisClient.Set(ctx, cacheKey, &user, 300) // Cache for 5 minutes

    return &user, nil
}

// GetByEmail gets a user by email
func (s *UserStoreImpl) GetByEmail(ctx context.Context, email string) (*models.User, error) {
    var user models.User
    query := `
        SELECT id, email, username, password_hash, full_name, bio,
               profile_picture, is_active, is_verified, is_admin,
               follower_count, following_count, story_count,
               created_at, updated_at, deleted_at
        FROM users 
        WHERE email = $1 AND deleted_at IS NULL`

    err := s.db.GetContext(ctx, &user, query, email)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get user by email: %w", err)
    }

    return &user, nil
}

// GetByUsername gets a user by username
func (s *UserStoreImpl) GetByUsername(ctx context.Context, username string) (*models.User, error) {
    var user models.User
    query := `
        SELECT id, email, username, password_hash, full_name, bio,
               profile_picture, is_active, is_verified, is_admin,
               follower_count, following_count, story_count,
               created_at, updated_at, deleted_at
        FROM users 
        WHERE username = $1 AND deleted_at IS NULL`

    err := s.db.GetContext(ctx, &user, query, username)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get user by username: %w", err)
    }

    return &user, nil
}

// Update updates a user
func (s *UserStoreImpl) Update(ctx context.Context, user *models.User) error {
    user.UpdatedAt = time.Now()

    query := `
        UPDATE users SET 
            email = $2, username = $3, password_hash = $4, full_name = $5,
            bio = $6, profile_picture = $7, is_active = $8, is_verified = $9,
            updated_at = $10
        WHERE id = $1 AND deleted_at IS NULL`

    result, err := s.db.ExecContext(ctx, query,
        user.ID, user.Email, user.Username, user.PasswordHash,
        user.FullName, user.Bio, user.ProfilePicture, user.IsActive,
        user.IsVerified, user.UpdatedAt,
    )

    if err != nil {
        return fmt.Errorf("failed to update user: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    // Invalidate cache
    cacheKey := fmt.Sprintf("user:%s", user.ID.String())
    s.redisClient.Delete(ctx, cacheKey)

    return nil
}

// Delete soft deletes a user
func (s *UserStoreImpl) Delete(ctx context.Context, id uuid.UUID) error {
    now := time.Now()
    query := `
        UPDATE users SET 
            deleted_at = $2, updated_at = $2
        WHERE id = $1 AND deleted_at IS NULL`

    result, err := s.db.ExecContext(ctx, query, id, now)
    if err != nil {
        return fmt.Errorf("failed to delete user: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    // Invalidate cache
    cacheKey := fmt.Sprintf("user:%s", id.String())
    s.redisClient.Delete(ctx, cacheKey)

    return nil
}

// List gets a list of users
func (s *UserStoreImpl) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
    query := `
        SELECT id, email, username, password_hash, full_name, bio,
               profile_picture, is_active, is_verified, is_admin,
               follower_count, following_count, story_count,
               created_at, updated_at, deleted_at
        FROM users 
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2`

    var users []*models.User
    err := s.db.SelectContext(ctx, &users, query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to list users: %w", err)
    }

    return users, nil
}

// Search searches for users
func (s *UserStoreImpl) Search(ctx context.Context, query string, limit, offset int) ([]*models.User, error) {
    searchQuery := `
        SELECT id, email, username, password_hash, full_name, bio,
               profile_picture, is_active, is_verified, is_admin,
               follower_count, following_count, story_count,
               created_at, updated_at, deleted_at
        FROM users 
        WHERE deleted_at IS NULL 
        AND (
            username ILIKE '%' || $1 || '%' OR 
            full_name ILIKE '%' || $1 || '%' OR
            to_tsvector('english', username || ' ' || coalesce(full_name, '')) @@ plainto_tsquery('english', $1)
        )
        ORDER BY 
            CASE 
                WHEN username ILIKE $1 || '%' THEN 1
                WHEN full_name ILIKE $1 || '%' THEN 2
                ELSE 3
            END,
            follower_count DESC
        LIMIT $2 OFFSET $3`

    var users []*models.User
    err := s.db.SelectContext(ctx, &users, searchQuery, query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to search users: %w", err)
    }

    return users, nil
}

// GetStats gets user statistics
func (s *UserStoreImpl) GetStats(ctx context.Context, userID uuid.UUID) (*models.UserStats, error) {
    query := `
        SELECT follower_count, following_count, story_count
        FROM users 
        WHERE id = $1 AND deleted_at IS NULL`

    var stats models.UserStats
    err := s.db.GetContext(ctx, &stats, query, userID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get user stats: %w", err)
    }

    return &stats, nil
}

// UpdateStats updates user statistics
func (s *UserStoreImpl) UpdateStats(ctx context.Context, userID uuid.UUID, stats models.UserStats) error {
    query := `
        UPDATE users SET 
            follower_count = $2, following_count = $3, story_count = $4,
            updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL`

    result, err := s.db.ExecContext(ctx, query, userID, stats.FollowerCount, stats.FollowingCount, stats.StoryCount)
    if err != nil {
        return fmt.Errorf("failed to update user stats: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    // Invalidate cache
    cacheKey := fmt.Sprintf("user:%s", userID.String())
    s.redisClient.Delete(ctx, cacheKey)

    return nil
}
