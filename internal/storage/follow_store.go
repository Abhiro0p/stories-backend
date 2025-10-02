package storage

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/models"
)

// FollowStoreImpl implements FollowStore interface
type FollowStoreImpl struct {
    db          *sqlx.DB
    redisClient *RedisClient
    logger      *zap.Logger
}

// NewFollowStore creates a new follow store
func NewFollowStore(db *sqlx.DB, redisClient *RedisClient, logger *zap.Logger) FollowStore {
    return &FollowStoreImpl{
        db:          db,
        redisClient: redisClient,
        logger:      logger.With(zap.String("store", "follow")),
    }
}

// Create creates a new follow relationship
func (s *FollowStoreImpl) Create(ctx context.Context, follow *models.Follow) error {
    tx, err := s.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Insert follow relationship
    query := `
        INSERT INTO follows (id, follower_id, followee_id, created_at)
        VALUES ($1, $2, $3, $4)`

    _, err = tx.ExecContext(ctx, query, follow.ID, follow.FollowerID, follow.FolloweeID, follow.CreatedAt)
    if err != nil {
        return fmt.Errorf("failed to create follow: %w", err)
    }

    // Update follower count for followee
    _, err = tx.ExecContext(ctx, 
        "UPDATE users SET follower_count = follower_count + 1 WHERE id = $1", 
        follow.FolloweeID)
    if err != nil {
        return fmt.Errorf("failed to update follower count: %w", err)
    }

    // Update following count for follower
    _, err = tx.ExecContext(ctx, 
        "UPDATE users SET following_count = following_count + 1 WHERE id = $1", 
        follow.FollowerID)
    if err != nil {
        return fmt.Errorf("failed to update following count: %w", err)
    }

    if err = tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    // Invalidate relevant caches
    s.invalidateFollowCaches(follow.FollowerID, follow.FolloweeID)

    s.logger.Info("Follow relationship created",
        zap.String("follower_id", follow.FollowerID.String()),
        zap.String("followee_id", follow.FolloweeID.String()),
    )

    return nil
}

// Delete deletes a follow relationship by ID - ADDED MISSING METHOD
func (s *FollowStoreImpl) Delete(ctx context.Context, id uuid.UUID) error {
    query := `DELETE FROM follows WHERE id = $1`

    result, err := s.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to delete follow: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    s.logger.Info("Follow relationship deleted", zap.String("follow_id", id.String()))
    return nil
}

// DeleteByUsers deletes follow relationship by user IDs
func (s *FollowStoreImpl) DeleteByUsers(ctx context.Context, followerID, followeeID uuid.UUID) error {
    tx, err := s.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Delete follow relationship
    result, err := tx.ExecContext(ctx,
        "DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2",
        followerID, followeeID)
    if err != nil {
        return fmt.Errorf("failed to delete follow: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    // Update follower count for followee
    _, err = tx.ExecContext(ctx,
        "UPDATE users SET follower_count = GREATEST(follower_count - 1, 0) WHERE id = $1",
        followeeID)
    if err != nil {
        return fmt.Errorf("failed to update follower count: %w", err)
    }

    // Update following count for follower
    _, err = tx.ExecContext(ctx,
        "UPDATE users SET following_count = GREATEST(following_count - 1, 0) WHERE id = $1",
        followerID)
    if err != nil {
        return fmt.Errorf("failed to update following count: %w", err)
    }

    if err = tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    // Invalidate relevant caches
    s.invalidateFollowCaches(followerID, followeeID)

    s.logger.Info("Follow relationship deleted",
        zap.String("follower_id", followerID.String()),
        zap.String("followee_id", followeeID.String()),
    )

    return nil
}

// GetByID gets a follow relationship by ID - ADDED MISSING METHOD
func (s *FollowStoreImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.Follow, error) {
    var follow models.Follow
    query := `
        SELECT id, follower_id, followee_id, created_at
        FROM follows 
        WHERE id = $1`

    err := s.db.GetContext(ctx, &follow, query, id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get follow: %w", err)
    }

    return &follow, nil
}

// GetFollowers gets followers for a user
func (s *FollowStoreImpl) GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.FollowWithUser, error) {
    query := `
        SELECT f.id, f.follower_id, f.followee_id, f.created_at,
               u.username as follower_username, u.full_name as follower_full_name,
               u.profile_picture as follower_profile_picture, u.is_verified as follower_is_verified
        FROM follows f
        JOIN users u ON f.follower_id = u.id
        WHERE f.followee_id = $1 AND u.deleted_at IS NULL
        ORDER BY f.created_at DESC
        LIMIT $2 OFFSET $3`

    var follows []*models.FollowWithUser
    err := s.db.SelectContext(ctx, &follows, query, userID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get followers: %w", err)
    }

    return follows, nil
}

// GetFollowing gets users that a user is following
func (s *FollowStoreImpl) GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.FollowWithUser, error) {
    query := `
        SELECT f.id, f.follower_id, f.followee_id, f.created_at,
               u.username as followee_username, u.full_name as followee_full_name,
               u.profile_picture as followee_profile_picture, u.is_verified as followee_is_verified
        FROM follows f
        JOIN users u ON f.followee_id = u.id
        WHERE f.follower_id = $1 AND u.deleted_at IS NULL
        ORDER BY f.created_at DESC
        LIMIT $2 OFFSET $3`

    var follows []*models.FollowWithUser
    err := s.db.SelectContext(ctx, &follows, query, userID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get following: %w", err)
    }

    return follows, nil
}

// IsFollowing checks if user A follows user B
func (s *FollowStoreImpl) IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error) {
    cacheKey := fmt.Sprintf("follow:%s:%s", followerID.String(), followeeID.String())
    
    // Check cache first
    var result bool
    if err := s.redisClient.Get(ctx, cacheKey, &result); err == nil {
        return result, nil
    }

    var count int
    query := `SELECT COUNT(*) FROM follows WHERE follower_id = $1 AND followee_id = $2`
    err := s.db.GetContext(ctx, &count, query, followerID, followeeID)
    if err != nil {
        return false, fmt.Errorf("failed to check follow status: %w", err)
    }

    isFollowing := count > 0

    // Cache the result
    s.redisClient.Set(ctx, cacheKey, isFollowing, 300) // Cache for 5 minutes

    return isFollowing, nil
}

// GetFollowStats gets follow statistics for a user - ADDED MISSING METHOD
func (s *FollowStoreImpl) GetFollowStats(ctx context.Context, userID uuid.UUID) (*models.FollowStats, error) {
    query := `
        SELECT 
            $1 as user_id,
            (SELECT COUNT(*) FROM follows WHERE followee_id = $1) as follower_count,
            (SELECT COUNT(*) FROM follows WHERE follower_id = $1) as following_count`

    var stats models.FollowStats
    err := s.db.GetContext(ctx, &stats, query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get follow stats: %w", err)
    }

    return &stats, nil
}

// GetMutualFollows gets mutual follow status between two users - ADDED MISSING METHOD
func (s *FollowStoreImpl) GetMutualFollows(ctx context.Context, userID1, userID2 uuid.UUID) (*models.MutualFollowCheck, error) {
    query := `
        SELECT 
            $1 as user_id_1,
            $2 as user_id_2,
            EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND followee_id = $2) as user1_follows_user2,
            EXISTS(SELECT 1 FROM follows WHERE follower_id = $2 AND followee_id = $1) as user2_follows_user1`

    var mutual models.MutualFollowCheck
    err := s.db.GetContext(ctx, &mutual, query, userID1, userID2)
    if err != nil {
        return nil, fmt.Errorf("failed to get mutual follows: %w", err)
    }

    mutual.CheckMutual()

    return &mutual, nil
}

// GetFollowSuggestions gets follow suggestions for a user - ADDED MISSING METHOD
func (s *FollowStoreImpl) GetFollowSuggestions(ctx context.Context, userID uuid.UUID, limit int) ([]*models.FollowSuggestion, error) {
    query := `
        SELECT u.id as user_id, u.username, u.full_name, u.profile_picture, u.is_verified,
               COUNT(f.follower_id) as mutual_followers,
               'mutual_connections' as suggestion_reason
        FROM users u
        LEFT JOIN follows f ON u.id = f.followee_id
        WHERE u.id != $1 
        AND u.deleted_at IS NULL
        AND u.id NOT IN (
            SELECT followee_id FROM follows WHERE follower_id = $1
        )
        AND EXISTS (
            SELECT 1 FROM follows f1 
            JOIN follows f2 ON f1.followee_id = f2.follower_id 
            WHERE f1.follower_id = $1 AND f2.followee_id = u.id
        )
        GROUP BY u.id, u.username, u.full_name, u.profile_picture, u.is_verified
        ORDER BY mutual_followers DESC, u.follower_count DESC
        LIMIT $2`

    var suggestions []*models.FollowSuggestion
    err := s.db.SelectContext(ctx, &suggestions, query, userID, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to get follow suggestions: %w", err)
    }

    return suggestions, nil
}

// Helper function to invalidate follow-related caches
func (s *FollowStoreImpl) invalidateFollowCaches(followerID, followeeID uuid.UUID) {
    ctx := context.Background()
    s.redisClient.Delete(ctx, fmt.Sprintf("follow:%s:%s", followerID.String(), followeeID.String()))
    s.redisClient.Delete(ctx, fmt.Sprintf("user:%s", followerID.String()))
    s.redisClient.Delete(ctx, fmt.Sprintf("user:%s", followeeID.String()))
}
