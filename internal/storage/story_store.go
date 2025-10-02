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

// StoryStoreImpl implements StoryStore interface
type StoryStoreImpl struct {
    db          *sqlx.DB
    redisClient *RedisClient
    logger      *zap.Logger
}

// NewStoryStore creates a new story store
func NewStoryStore(db *sqlx.DB, redisClient *RedisClient, logger *zap.Logger) StoryStore {
    return &StoryStoreImpl{
        db:          db,
        redisClient: redisClient,
        logger:      logger.With(zap.String("store", "story")),
    }
}

// Create creates a new story
func (s *StoryStoreImpl) Create(ctx context.Context, story *models.Story) error {
    query := `
        INSERT INTO stories (
            id, author_id, type, text, media_url, media_key, 
            visibility, view_count, expires_at, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
        )`

    _, err := s.db.ExecContext(ctx, query,
        story.ID, story.AuthorID, story.Type, story.Text,
        story.MediaURL, story.MediaKey, story.Visibility,
        story.ViewCount, story.ExpiresAt, story.CreatedAt, story.UpdatedAt,
    )

    if err != nil {
        s.logger.Error("Failed to create story", zap.Error(err))
        return fmt.Errorf("failed to create story: %w", err)
    }

    // Invalidate related caches
    s.invalidateStoryCache(story.AuthorID)
    
    s.logger.Info("Story created", zap.String("story_id", story.ID.String()))
    return nil
}

// GetByID gets a story by ID with caching
func (s *StoryStoreImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.Story, error) {
    cacheKey := fmt.Sprintf("story:%s", id.String())
    var story models.Story

    if err := s.redisClient.Get(ctx, cacheKey, &story); err == nil {
        return &story, nil
    }

    query := `
        SELECT s.id, s.author_id, s.type, s.text, s.media_url, s.media_key,
               s.visibility, s.view_count, s.expires_at, s.created_at, 
               s.updated_at, s.deleted_at,
               u.username as author_username, u.full_name as author_full_name,
               u.profile_picture as author_profile_picture, u.is_verified as author_is_verified
        FROM stories s
        JOIN users u ON s.author_id = u.id
        WHERE s.id = $1 AND s.deleted_at IS NULL`

    var storyWithAuthor models.StoryWithAuthor
    err := s.db.GetContext(ctx, &storyWithAuthor, query, id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get story: %w", err)
    }

    // Convert to Story model
    story = storyWithAuthor.Story
    story.Author = storyWithAuthor.GetAuthorInfo()

    // Cache the result
    s.redisClient.Set(ctx, cacheKey, &story, 180) // Cache for 3 minutes

    return &story, nil
}

// GetFeed gets stories feed for a user
func (s *StoryStoreImpl) GetFeed(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Story, error) {
    query := `
        SELECT DISTINCT s.id, s.author_id, s.type, s.text, s.media_url, s.media_key,
               s.visibility, s.view_count, s.expires_at, s.created_at, s.updated_at,
               u.username as author_username, u.full_name as author_full_name,
               u.profile_picture as author_profile_picture, u.is_verified as author_is_verified
        FROM stories s
        JOIN users u ON s.author_id = u.id
        LEFT JOIN follows f ON s.author_id = f.followee_id
        WHERE s.deleted_at IS NULL 
        AND s.expires_at > NOW()
        AND (
            s.visibility = 'public' OR 
            (s.visibility = 'friends' AND f.follower_id = $1) OR
            s.author_id = $1
        )
        ORDER BY s.created_at DESC
        LIMIT $2 OFFSET $3`

    var storiesWithAuthor []models.StoryWithAuthor
    err := s.db.SelectContext(ctx, &storiesWithAuthor, query, userID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get stories feed: %w", err)
    }

    stories := make([]*models.Story, len(storiesWithAuthor))
    for i, storyWithAuthor := range storiesWithAuthor {
        story := storyWithAuthor.Story
        story.Author = storyWithAuthor.GetAuthorInfo()
        stories[i] = &story
    }

    return stories, nil
}

// GetPublic gets public stories - ADDED MISSING METHOD
func (s *StoryStoreImpl) GetPublic(ctx context.Context, limit, offset int) ([]*models.Story, error) {
    query := `
        SELECT DISTINCT s.id, s.author_id, s.type, s.text, s.media_url, s.media_key,
               s.visibility, s.view_count, s.expires_at, s.created_at, s.updated_at,
               u.username as author_username, u.full_name as author_full_name,
               u.profile_picture as author_profile_picture, u.is_verified as author_is_verified
        FROM stories s
        JOIN users u ON s.author_id = u.id
        WHERE s.deleted_at IS NULL 
        AND s.expires_at > NOW()
        AND s.visibility = 'public'
        ORDER BY s.created_at DESC
        LIMIT $1 OFFSET $2`

    var storiesWithAuthor []models.StoryWithAuthor
    err := s.db.SelectContext(ctx, &storiesWithAuthor, query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get public stories: %w", err)
    }

    stories := make([]*models.Story, len(storiesWithAuthor))
    for i, storyWithAuthor := range storiesWithAuthor {
        story := storyWithAuthor.Story
        story.Author = storyWithAuthor.GetAuthorInfo()
        stories[i] = &story
    }

    return stories, nil
}

// GetByAuthorID gets stories by author ID
func (s *StoryStoreImpl) GetByAuthorID(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]*models.Story, error) {
    query := `
        SELECT id, author_id, type, text, media_url, media_key,
               visibility, view_count, expires_at, created_at, updated_at, deleted_at
        FROM stories 
        WHERE author_id = $1 AND deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3`

    var stories []*models.Story
    err := s.db.SelectContext(ctx, &stories, query, authorID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get stories by author: %w", err)
    }

    return stories, nil
}

// Update updates a story
func (s *StoryStoreImpl) Update(ctx context.Context, story *models.Story) error {
    story.UpdatedAt = time.Now()

    query := `
        UPDATE stories SET 
            text = $2, visibility = $3, updated_at = $4
        WHERE id = $1 AND deleted_at IS NULL`

    result, err := s.db.ExecContext(ctx, query, story.ID, story.Text, story.Visibility, story.UpdatedAt)
    if err != nil {
        return fmt.Errorf("failed to update story: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    // Invalidate caches
    s.invalidateStoryCache(story.AuthorID)
    cacheKey := fmt.Sprintf("story:%s", story.ID.String())
    s.redisClient.Delete(ctx, cacheKey)

    return nil
}

// Delete soft deletes a story
func (s *StoryStoreImpl) Delete(ctx context.Context, id uuid.UUID) error {
    now := time.Now()
    query := `
        UPDATE stories SET 
            deleted_at = $2, updated_at = $2
        WHERE id = $1 AND deleted_at IS NULL`

    result, err := s.db.ExecContext(ctx, query, id, now)
    if err != nil {
        return fmt.Errorf("failed to delete story: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    // Invalidate cache
    cacheKey := fmt.Sprintf("story:%s", id.String())
    s.redisClient.Delete(ctx, cacheKey)

    return nil
}

// GetExpired gets expired stories
func (s *StoryStoreImpl) GetExpired(ctx context.Context, limit, offset int) ([]*models.Story, error) {
    query := `
        SELECT id, author_id, type, text, media_url, media_key,
               visibility, view_count, expires_at, created_at, updated_at, deleted_at
        FROM stories 
        WHERE expires_at <= NOW() AND deleted_at IS NULL
        ORDER BY expires_at ASC
        LIMIT $1 OFFSET $2`

    var stories []*models.Story
    err := s.db.SelectContext(ctx, &stories, query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get expired stories: %w", err)
    }

    return stories, nil
}

// IncrementViewCount increments view count for a story
func (s *StoryStoreImpl) IncrementViewCount(ctx context.Context, storyID uuid.UUID) error {
    query := `
        UPDATE stories SET 
            view_count = view_count + 1,
            updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL`

    _, err := s.db.ExecContext(ctx, query, storyID)
    if err != nil {
        return fmt.Errorf("failed to increment view count: %w", err)
    }

    // Invalidate cache
    cacheKey := fmt.Sprintf("story:%s", storyID.String())
    s.redisClient.Delete(ctx, cacheKey)

    return nil
}

// GetViewCount gets view count for a story
func (s *StoryStoreImpl) GetViewCount(ctx context.Context, storyID uuid.UUID) (int, error) {
    var count int
    query := `SELECT view_count FROM stories WHERE id = $1 AND deleted_at IS NULL`
    
    err := s.db.GetContext(ctx, &count, query, storyID)
    if err != nil {
        if err == sql.ErrNoRows {
            return 0, ErrNotFound
        }
        return 0, fmt.Errorf("failed to get view count: %w", err)
    }

    return count, nil
}

// Helper function to invalidate story-related caches
func (s *StoryStoreImpl) invalidateStoryCache(authorID uuid.UUID) {
    // This would invalidate feed caches, author story caches, etc.
    // Implementation depends on your caching strategy
    ctx := context.Background()
    s.redisClient.Delete(ctx, fmt.Sprintf("user_stories:%s", authorID.String()))
}
