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

// ViewStoreImpl implements ViewStore interface
type ViewStoreImpl struct {
    db          *sqlx.DB
    redisClient *RedisClient
    logger      *zap.Logger
}

// NewViewStore creates a new view store
func NewViewStore(db *sqlx.DB, redisClient *RedisClient, logger *zap.Logger) ViewStore {
    return &ViewStoreImpl{
        db:          db,
        redisClient: redisClient,
        logger:      logger.With(zap.String("store", "view")),
    }
}

// Create creates a new story view
func (s *ViewStoreImpl) Create(ctx context.Context, view *models.StoryView) error {
    query := `
        INSERT INTO story_views (id, story_id, viewer_id, viewed_at, ip_address, user_agent)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (story_id, viewer_id) DO UPDATE SET
            viewed_at = EXCLUDED.viewed_at,
            ip_address = EXCLUDED.ip_address,
            user_agent = EXCLUDED.user_agent`

    _, err := s.db.ExecContext(ctx, query,
        view.ID, view.StoryID, view.ViewerID, view.ViewedAt,
        view.IPAddress, view.UserAgent,
    )

    if err != nil {
        s.logger.Error("Failed to create story view", zap.Error(err))
        return fmt.Errorf("failed to create story view: %w", err)
    }

    s.logger.Debug("Story view created",
        zap.String("story_id", view.StoryID.String()),
        zap.String("viewer_id", view.ViewerID.String()),
    )

    return nil
}

// GetByID gets a story view by ID
func (s *ViewStoreImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.StoryView, error) {
    var view models.StoryView
    query := `
        SELECT id, story_id, viewer_id, viewed_at, ip_address, user_agent
        FROM story_views 
        WHERE id = $1`

    err := s.db.GetContext(ctx, &view, query, id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get story view: %w", err)
    }

    return &view, nil
}

// GetByStoryID gets views for a specific story
func (s *ViewStoreImpl) GetByStoryID(ctx context.Context, storyID uuid.UUID, limit, offset int) ([]*models.StoryViewWithUser, error) {
    query := `
        SELECT sv.id, sv.story_id, sv.viewer_id, sv.viewed_at, sv.ip_address, sv.user_agent,
               u.username as viewer_username, u.full_name as viewer_full_name,
               u.profile_picture as viewer_profile_picture, u.is_verified as viewer_is_verified
        FROM story_views sv
        JOIN users u ON sv.viewer_id = u.id
        WHERE sv.story_id = $1 AND u.deleted_at IS NULL
        ORDER BY sv.viewed_at DESC
        LIMIT $2 OFFSET $3`

    var views []*models.StoryViewWithUser
    err := s.db.SelectContext(ctx, &views, query, storyID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get story views: %w", err)
    }

    return views, nil
}

// GetByViewerID gets views by a specific viewer
func (s *ViewStoreImpl) GetByViewerID(ctx context.Context, viewerID uuid.UUID, limit, offset int) ([]*models.StoryView, error) {
    query := `
        SELECT id, story_id, viewer_id, viewed_at, ip_address, user_agent
        FROM story_views
        WHERE viewer_id = $1
        ORDER BY viewed_at DESC
        LIMIT $2 OFFSET $3`

    var views []*models.StoryView
    err := s.db.SelectContext(ctx, &views, query, viewerID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get viewer views: %w", err)
    }

    return views, nil
}

// GetViewStats gets view statistics for a story
func (s *ViewStoreImpl) GetViewStats(ctx context.Context, storyID uuid.UUID) (*models.StoryViewStats, error) {
    query := `
        SELECT 
            $1 as story_id,
            COUNT(*) as total_views,
            COUNT(DISTINCT viewer_id) as unique_views,
            MAX(viewed_at) as last_viewed_at
        FROM story_views
        WHERE story_id = $1`

    var stats models.StoryViewStats
    err := s.db.GetContext(ctx, &stats, query, storyID)
    if err != nil {
        return nil, fmt.Errorf("failed to get view stats: %w", err)
    }

    return &stats, nil
}

// GetViewerStats gets viewing statistics for a user
func (s *ViewStoreImpl) GetViewerStats(ctx context.Context, viewerID uuid.UUID) (*models.ViewerStats, error) {
    query := `
        SELECT 
            $1 as viewer_id,
            COUNT(DISTINCT story_id) as stories_viewed,
            MAX(viewed_at) as last_viewed_at
        FROM story_views
        WHERE viewer_id = $1`

    var stats models.ViewerStats
    err := s.db.GetContext(ctx, &stats, query, viewerID)
    if err != nil {
        return nil, fmt.Errorf("failed to get viewer stats: %w", err)
    }

    return &stats, nil
}

// HasViewed checks if a user has viewed a specific story
func (s *ViewStoreImpl) HasViewed(ctx context.Context, storyID, viewerID uuid.UUID) (bool, error) {
    var count int
    query := `SELECT COUNT(*) FROM story_views WHERE story_id = $1 AND viewer_id = $2`
    
    err := s.db.GetContext(ctx, &count, query, storyID, viewerID)
    if err != nil {
        return false, fmt.Errorf("failed to check view status: %w", err)
    }

    return count > 0, nil
}

// GetViewAnalytics gets analytics data for story views
func (s *ViewStoreImpl) GetViewAnalytics(ctx context.Context, storyID uuid.UUID, period string) (*models.ViewAnalytics, error) {
    // This is a simplified implementation
    // In a real-world scenario, you'd want more sophisticated analytics
    
    stats, err := s.GetViewStats(ctx, storyID)
    if err != nil {
        return nil, err
    }
    
    analytics := &models.ViewAnalytics{
        StoryID:     storyID,
        Period:      period,
        TotalViews:  stats.TotalViews,
        UniqueViews: stats.UniqueViews,
        ViewsByHour: make(map[string]int),
        ViewsByDevice: make(map[string]int),
    }
    
    // Get hourly breakdown
    hourlyQuery := `
        SELECT 
            DATE_TRUNC('hour', viewed_at) as hour,
            COUNT(*) as view_count
        FROM story_views
        WHERE story_id = $1 
        AND viewed_at >= NOW() - INTERVAL '24 hours'
        GROUP BY hour
        ORDER BY hour`
    
    type hourlyResult struct {
        Hour      string `db:"hour"`
        ViewCount int    `db:"view_count"`
    }
    
    var hourlyResults []hourlyResult
    err = s.db.SelectContext(ctx, &hourlyResults, hourlyQuery, storyID)
    if err != nil {
        s.logger.Warn("Failed to get hourly analytics", zap.Error(err))
    } else {
        for _, result := range hourlyResults {
            analytics.ViewsByHour[result.Hour] = result.ViewCount
        }
    }
    
    return analytics, nil
}

// GetViewTrends gets view trend data over time - ADDED MISSING METHOD
func (s *ViewStoreImpl) GetViewTrends(ctx context.Context, storyID uuid.UUID) ([]*models.ViewTrend, error) {
    query := `
        SELECT 
            DATE_TRUNC('hour', viewed_at) as hour,
            COUNT(*) as view_count,
            COUNT(DISTINCT viewer_id) as unique_count
        FROM story_views
        WHERE story_id = $1 
        AND viewed_at >= NOW() - INTERVAL '24 hours'
        GROUP BY hour
        ORDER BY hour`

    var trends []*models.ViewTrend
    err := s.db.SelectContext(ctx, &trends, query, storyID)
    if err != nil {
        return nil, fmt.Errorf("failed to get view trends: %w", err)
    }

    return trends, nil
}
