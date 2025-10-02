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

// ReactionStoreImpl implements ReactionStore interface
type ReactionStoreImpl struct {
    db          *sqlx.DB
    redisClient *RedisClient
    logger      *zap.Logger
}

// NewReactionStore creates a new reaction store
func NewReactionStore(db *sqlx.DB, redisClient *RedisClient, logger *zap.Logger) ReactionStore {
    return &ReactionStoreImpl{
        db:          db,
        redisClient: redisClient,
        logger:      logger.With(zap.String("store", "reaction")),
    }
}

// Create creates a new reaction
func (s *ReactionStoreImpl) Create(ctx context.Context, reaction *models.Reaction) error {
    query := `
        INSERT INTO reactions (id, story_id, user_id, type, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (story_id, user_id) DO UPDATE SET
            type = EXCLUDED.type,
            updated_at = EXCLUDED.updated_at`

    _, err := s.db.ExecContext(ctx, query,
        reaction.ID, reaction.StoryID, reaction.UserID,
        reaction.Type, reaction.CreatedAt, reaction.UpdatedAt,
    )

    if err != nil {
        s.logger.Error("Failed to create reaction", zap.Error(err))
        return fmt.Errorf("failed to create reaction: %w", err)
    }

    s.logger.Debug("Reaction created",
        zap.String("reaction_id", reaction.ID.String()),
        zap.String("story_id", reaction.StoryID.String()),
        zap.String("user_id", reaction.UserID.String()),
        zap.String("type", string(reaction.Type)),
    )

    return nil
}

// GetByID gets a reaction by ID
func (s *ReactionStoreImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.Reaction, error) {
    var reaction models.Reaction
    query := `
        SELECT id, story_id, user_id, type, created_at, updated_at
        FROM reactions 
        WHERE id = $1`

    err := s.db.GetContext(ctx, &reaction, query, id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get reaction: %w", err)
    }

    return &reaction, nil
}

// GetByStoryID gets reactions for a specific story
func (s *ReactionStoreImpl) GetByStoryID(ctx context.Context, storyID uuid.UUID, limit, offset int) ([]*models.ReactionWithUser, error) {
    query := `
        SELECT r.id, r.story_id, r.user_id, r.type, r.created_at, r.updated_at,
               u.username, u.full_name, u.profile_picture, u.is_verified
        FROM reactions r
        JOIN users u ON r.user_id = u.id
        WHERE r.story_id = $1 AND u.deleted_at IS NULL
        ORDER BY r.created_at DESC
        LIMIT $2 OFFSET $3`

    var reactions []*models.ReactionWithUser
    err := s.db.SelectContext(ctx, &reactions, query, storyID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get story reactions: %w", err)
    }

    return reactions, nil
}

// GetByUserID gets reactions by a specific user
func (s *ReactionStoreImpl) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Reaction, error) {
    query := `
        SELECT id, story_id, user_id, type, created_at, updated_at
        FROM reactions
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3`

    var reactions []*models.Reaction
    err := s.db.SelectContext(ctx, &reactions, query, userID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get user reactions: %w", err)
    }

    return reactions, nil
}

// Update updates a reaction
func (s *ReactionStoreImpl) Update(ctx context.Context, reaction *models.Reaction) error {
    query := `
        UPDATE reactions SET 
            type = $2, updated_at = $3
        WHERE id = $1`

    result, err := s.db.ExecContext(ctx, query, reaction.ID, reaction.Type, reaction.UpdatedAt)
    if err != nil {
        return fmt.Errorf("failed to update reaction: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    s.logger.Debug("Reaction updated",
        zap.String("reaction_id", reaction.ID.String()),
        zap.String("type", string(reaction.Type)),
    )

    return nil
}

// Delete deletes a reaction
func (s *ReactionStoreImpl) Delete(ctx context.Context, id uuid.UUID) error {
    query := `DELETE FROM reactions WHERE id = $1`

    result, err := s.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to delete reaction: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    s.logger.Debug("Reaction deleted", zap.String("reaction_id", id.String()))
    return nil
}

// GetUserReactionForStory gets a user's reaction for a specific story
func (s *ReactionStoreImpl) GetUserReactionForStory(ctx context.Context, storyID, userID uuid.UUID) (*models.Reaction, error) {
    var reaction models.Reaction
    query := `
        SELECT id, story_id, user_id, type, created_at, updated_at
        FROM reactions 
        WHERE story_id = $1 AND user_id = $2`

    err := s.db.GetContext(ctx, &reaction, query, storyID, userID)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get user reaction: %w", err)
    }

    return &reaction, nil
}

// GetReactionSummary gets a summary of reactions for a story
func (s *ReactionStoreImpl) GetReactionSummary(ctx context.Context, storyID uuid.UUID) (*models.ReactionSummary, error) {
    // Get total count and type breakdown
    statsQuery := `
        SELECT 
            type,
            COUNT(*) as count
        FROM reactions
        WHERE story_id = $1
        GROUP BY type`

    type reactionStat struct {
        Type  models.ReactionType `db:"type"`
        Count int                 `db:"count"`
    }

    var stats []reactionStat
    err := s.db.SelectContext(ctx, &stats, statsQuery, storyID)
    if err != nil {
        return nil, fmt.Errorf("failed to get reaction stats: %w", err)
    }

    summary := &models.ReactionSummary{
        StoryID:        storyID,
        ReactionCounts: make(map[models.ReactionType]int),
    }

    totalReactions := 0
    for _, stat := range stats {
        summary.ReactionCounts[stat.Type] = stat.Count
        totalReactions += stat.Count
    }
    summary.TotalReactions = totalReactions

    // Get recent reactions with user info
    recentQuery := `
        SELECT r.id, r.story_id, r.user_id, r.type, r.created_at, r.updated_at,
               u.username, u.full_name, u.profile_picture, u.is_verified
        FROM reactions r
        JOIN users u ON r.user_id = u.id
        WHERE r.story_id = $1 AND u.deleted_at IS NULL
        ORDER BY r.created_at DESC
        LIMIT 10`

    var recentReactions []models.ReactionWithUser
    err = s.db.SelectContext(ctx, &recentReactions, recentQuery, storyID)
    if err != nil {
        s.logger.Warn("Failed to get recent reactions", zap.Error(err))
    } else {
        summary.RecentReactions = recentReactions
    }

    return summary, nil
}

// GetReactionStats gets reaction statistics for a story
func (s *ReactionStoreImpl) GetReactionStats(ctx context.Context, storyID uuid.UUID) (map[models.ReactionType]int, error) {
    query := `
        SELECT type, COUNT(*) as count
        FROM reactions
        WHERE story_id = $1
        GROUP BY type`

    type reactionStat struct {
        Type  models.ReactionType `db:"type"`
        Count int                 `db:"count"`
    }

    var stats []reactionStat
    err := s.db.SelectContext(ctx, &stats, query, storyID)
    if err != nil {
        return nil, fmt.Errorf("failed to get reaction stats: %w", err)
    }

    result := make(map[models.ReactionType]int)
    for _, stat := range stats {
        result[stat.Type] = stat.Count
    }

    return result, nil
}
