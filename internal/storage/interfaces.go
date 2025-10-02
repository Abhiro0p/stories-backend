package storage

import (
    "context"
    "errors"

    "github.com/google/uuid"

    "github.com/Abhiro0p/stories-backend/internal/models"
)

// Common errors
var (
    ErrNotFound      = errors.New("record not found")
    ErrAlreadyExists = errors.New("record already exists")
    ErrInvalidInput  = errors.New("invalid input")
    ErrUnauthorized  = errors.New("unauthorized")
)

// UserStore defines the interface for user storage operations
type UserStore interface {
    Create(ctx context.Context, user *models.User) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
    GetByEmail(ctx context.Context, email string) (*models.User, error)
    GetByUsername(ctx context.Context, username string) (*models.User, error)
    Update(ctx context.Context, user *models.User) error
    Delete(ctx context.Context, id uuid.UUID) error
    Search(ctx context.Context, query string, limit, offset int) ([]*models.User, error)
    List(ctx context.Context, limit, offset int) ([]*models.User, error)
    UpdateStats(ctx context.Context, userID uuid.UUID, stats models.UserStats) error
    GetStats(ctx context.Context, userID uuid.UUID) (*models.UserStats, error)
}

// StoryStore defines the interface for story storage operations
type StoryStore interface {
    Create(ctx context.Context, story *models.Story) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.Story, error)
    GetByAuthorID(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]*models.Story, error)
    GetFeed(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Story, error)
    GetPublic(ctx context.Context, limit, offset int) ([]*models.Story, error)
    Update(ctx context.Context, story *models.Story) error
    Delete(ctx context.Context, id uuid.UUID) error
    GetExpired(ctx context.Context, limit, offset int) ([]*models.Story, error)
    IncrementViewCount(ctx context.Context, storyID uuid.UUID) error
    GetViewCount(ctx context.Context, storyID uuid.UUID) (int, error)
}

// FollowStore defines the interface for follow relationship storage operations
type FollowStore interface {
    Create(ctx context.Context, follow *models.Follow) error
    Delete(ctx context.Context, id uuid.UUID) error
    DeleteByUsers(ctx context.Context, followerID, followeeID uuid.UUID) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.Follow, error)
    GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.FollowWithUser, error)
    GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.FollowWithUser, error)
    IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error)
    GetFollowStats(ctx context.Context, userID uuid.UUID) (*models.FollowStats, error)
    GetMutualFollows(ctx context.Context, userID1, userID2 uuid.UUID) (*models.MutualFollowCheck, error)
    GetFollowSuggestions(ctx context.Context, userID uuid.UUID, limit int) ([]*models.FollowSuggestion, error)
}

// ViewStore defines the interface for story view storage operations
type ViewStore interface {
    Create(ctx context.Context, view *models.StoryView) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.StoryView, error)
    GetByStoryID(ctx context.Context, storyID uuid.UUID, limit, offset int) ([]*models.StoryViewWithUser, error)
    GetByViewerID(ctx context.Context, viewerID uuid.UUID, limit, offset int) ([]*models.StoryView, error)
    GetViewStats(ctx context.Context, storyID uuid.UUID) (*models.StoryViewStats, error)
    GetViewerStats(ctx context.Context, viewerID uuid.UUID) (*models.ViewerStats, error)
    HasViewed(ctx context.Context, storyID, viewerID uuid.UUID) (bool, error)
    GetViewAnalytics(ctx context.Context, storyID uuid.UUID, period string) (*models.ViewAnalytics, error)
    GetViewTrends(ctx context.Context, storyID uuid.UUID) ([]*models.ViewTrend, error)
}

// ReactionStore defines the interface for reaction storage operations
type ReactionStore interface {
    Create(ctx context.Context, reaction *models.Reaction) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.Reaction, error)
    GetByStoryID(ctx context.Context, storyID uuid.UUID, limit, offset int) ([]*models.ReactionWithUser, error)
    GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Reaction, error)
    Update(ctx context.Context, reaction *models.Reaction) error
    Delete(ctx context.Context, id uuid.UUID) error
    GetUserReactionForStory(ctx context.Context, storyID, userID uuid.UUID) (*models.Reaction, error)
    GetReactionSummary(ctx context.Context, storyID uuid.UUID) (*models.ReactionSummary, error)
    GetReactionStats(ctx context.Context, storyID uuid.UUID) (map[models.ReactionType]int, error)
}

// CacheStore defines the interface for caching operations
type CacheStore interface {
    Set(ctx context.Context, key string, value interface{}, expiration int) error
    Get(ctx context.Context, key string, dest interface{}) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    SetMany(ctx context.Context, items map[string]interface{}, expiration int) error
    GetMany(ctx context.Context, keys []string) (map[string]interface{}, error)
    DeleteMany(ctx context.Context, keys []string) error
    Increment(ctx context.Context, key string, value int64) (int64, error)
    Decrement(ctx context.Context, key string, value int64) (int64, error)
    Expire(ctx context.Context, key string, expiration int) error
    TTL(ctx context.Context, key string) (int, error)
    FlushAll(ctx context.Context) error
}

// SessionStore defines the interface for session storage operations
type SessionStore interface {
    Create(ctx context.Context, session *models.Session) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.Session, error)
    GetByTokenID(ctx context.Context, tokenID string) (*models.Session, error)
    GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Session, error)
    Update(ctx context.Context, session *models.Session) error
    Delete(ctx context.Context, id uuid.UUID) error
    DeleteExpired(ctx context.Context) error
    DeleteByUserID(ctx context.Context, userID uuid.UUID) error
    IsValidSession(ctx context.Context, tokenID string) (bool, error)
}

// AnalyticsStore defines the interface for analytics storage operations
type AnalyticsStore interface {
    RecordEvent(ctx context.Context, event AnalyticsEvent) error
    GetUserMetrics(ctx context.Context, userID uuid.UUID, period string) (*UserMetrics, error)
    GetStoryMetrics(ctx context.Context, storyID uuid.UUID, period string) (*StoryMetrics, error)
    GetPlatformMetrics(ctx context.Context, period string) (*PlatformMetrics, error)
    GetTopStories(ctx context.Context, period string, limit int) ([]*TopStoryMetric, error)
    GetTopUsers(ctx context.Context, period string, limit int) ([]*TopUserMetric, error)
    GetEngagementMetrics(ctx context.Context, period string) (*EngagementMetrics, error)
}

// Analytics types
type AnalyticsEvent struct {
    Type      string                 `json:"type"`
    UserID    *uuid.UUID             `json:"user_id,omitempty"`
    StoryID   *uuid.UUID             `json:"story_id,omitempty"`
    Timestamp int64                  `json:"timestamp"`
    Data      map[string]interface{} `json:"data,omitempty"`
}

type UserMetrics struct {
    UserID        uuid.UUID `json:"user_id"`
    StoriesPosted int       `json:"stories_posted"`
    ViewsReceived int       `json:"views_received"`
    ReactionsReceived int   `json:"reactions_received"`
    FollowersGained int     `json:"followers_gained"`
}

type StoryMetrics struct {
    StoryID     uuid.UUID `json:"story_id"`
    Views       int       `json:"views"`
    UniqueViews int       `json:"unique_views"`
    Reactions   int       `json:"reactions"`
    Shares      int       `json:"shares"`
}

type PlatformMetrics struct {
    ActiveUsers    int `json:"active_users"`
    StoriesPosted  int `json:"stories_posted"`
    TotalViews     int `json:"total_views"`
    TotalReactions int `json:"total_reactions"`
}

type TopStoryMetric struct {
    StoryID   uuid.UUID `json:"story_id"`
    AuthorID  uuid.UUID `json:"author_id"`
    Views     int       `json:"views"`
    Reactions int       `json:"reactions"`
    Score     float64   `json:"score"`
}

type TopUserMetric struct {
    UserID          uuid.UUID `json:"user_id"`
    StoriesPosted   int       `json:"stories_posted"`
    TotalViews      int       `json:"total_views"`
    TotalReactions  int       `json:"total_reactions"`
    FollowerCount   int       `json:"follower_count"`
    Score           float64   `json:"score"`
}

type EngagementMetrics struct {
    AverageViewsPerStory     float64 `json:"average_views_per_story"`
    AverageReactionsPerStory float64 `json:"average_reactions_per_story"`
    ViewToReactionRatio      float64 `json:"view_to_reaction_ratio"`
    DailyActiveUsers         int     `json:"daily_active_users"`
    RetentionRate            float64 `json:"retention_rate"`
}

// Health check interface
type HealthChecker interface {
    HealthCheck(ctx context.Context) error
}

// Transaction interface for database operations
type Transactional interface {
    WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// Paginated results
type PaginatedResult struct {
    Data       interface{} `json:"data"`
    Total      int         `json:"total"`
    Limit      int         `json:"limit"`
    Offset     int         `json:"offset"`
    HasMore    bool        `json:"has_more"`
    NextOffset *int        `json:"next_offset,omitempty"`
}

// Search filters
type SearchFilters struct {
    Query      string            `json:"query"`
    Filters    map[string]string `json:"filters"`
    SortBy     string            `json:"sort_by"`
    SortOrder  string            `json:"sort_order"`
    Limit      int               `json:"limit"`
    Offset     int               `json:"offset"`
}

// Batch operations
type BatchOperation struct {
    Operation string      `json:"operation"`
    Data      interface{} `json:"data"`
}

type BatchResult struct {
    Success []uuid.UUID `json:"success"`
    Failed  []uuid.UUID `json:"failed"`
    Errors  []string    `json:"errors"`
}

// Store factory interface
type StoreFactory interface {
    NewUserStore() UserStore
    NewStoryStore() StoryStore
    NewFollowStore() FollowStore
    NewViewStore() ViewStore
    NewReactionStore() ReactionStore
    NewSessionStore() SessionStore
    NewAnalyticsStore() AnalyticsStore
    Close() error
}
