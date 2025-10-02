package models

import (
    "fmt"
    "net"
    "time"

    "github.com/google/uuid"
)

// StoryView represents a view of a story by a user
type StoryView struct {
    ID        uuid.UUID  `json:"id" db:"id"`
    StoryID   uuid.UUID  `json:"story_id" db:"story_id"`
    ViewerID  uuid.UUID  `json:"viewer_id" db:"viewer_id"`
    ViewedAt  time.Time  `json:"viewed_at" db:"viewed_at"`
    IPAddress *net.IP    `json:"ip_address,omitempty" db:"ip_address"`
    UserAgent *string    `json:"user_agent,omitempty" db:"user_agent"`
}

// StoryViewWithUser represents a story view with user information
type StoryViewWithUser struct {
    StoryView
    ViewerUsername       string  `json:"viewer_username" db:"viewer_username"`
    ViewerFullName       *string `json:"viewer_full_name" db:"viewer_full_name"`
    ViewerProfilePicture *string `json:"viewer_profile_picture" db:"viewer_profile_picture"`
    ViewerIsVerified     bool    `json:"viewer_is_verified" db:"viewer_is_verified"`
}

// StoryViewStats represents viewing statistics for a story
type StoryViewStats struct {
    StoryID      uuid.UUID `json:"story_id" db:"story_id"`
    TotalViews   int       `json:"total_views" db:"total_views"`
    UniqueViews  int       `json:"unique_views" db:"unique_views"`
    LastViewedAt time.Time `json:"last_viewed_at" db:"last_viewed_at"`
}

// ViewerStats represents viewing statistics for a user
type ViewerStats struct {
    ViewerID     uuid.UUID `json:"viewer_id" db:"viewer_id"`
    StoriesViewed int      `json:"stories_viewed" db:"stories_viewed"`
    LastViewedAt time.Time `json:"last_viewed_at" db:"last_viewed_at"`
}

// ViewAnalytics represents analytics data for story views
type ViewAnalytics struct {
    StoryID       uuid.UUID            `json:"story_id"`
    Period        string               `json:"period"`
    TotalViews    int                  `json:"total_views"`
    UniqueViews   int                  `json:"unique_views"`
    ViewsByHour   map[string]int       `json:"views_by_hour"`
    ViewsByDevice map[string]int       `json:"views_by_device"`
    TopViewers    []ViewerSummary      `json:"top_viewers"`
}

// ViewTrend represents view trend data over time
type ViewTrend struct {
    Hour        time.Time `json:"hour" db:"hour"`
    ViewCount   int       `json:"view_count" db:"view_count"`
    UniqueCount int       `json:"unique_count" db:"unique_count"`
}

// ViewerSummary represents a summary of a viewer
type ViewerSummary struct {
    ViewerID     uuid.UUID `json:"viewer_id"`
    Username     string    `json:"username"`
    FullName     *string   `json:"full_name"`
    ViewCount    int       `json:"view_count"`
    LastViewedAt time.Time `json:"last_viewed_at"`
}

// NewStoryView creates a new story view
func NewStoryView(storyID, viewerID uuid.UUID, ipAddress *string, userAgent *string) *StoryView {
    view := &StoryView{
        ID:       uuid.New(),
        StoryID:  storyID,
        ViewerID: viewerID,
        ViewedAt: time.Now(),
    }
    
    // Parse IP address if provided
    if ipAddress != nil {
        if ip := net.ParseIP(*ipAddress); ip != nil {
            view.IPAddress = &ip
        }
    }
    
    view.UserAgent = userAgent
    
    return view
}

// GetViewerInfo extracts viewer information
func (svw *StoryViewWithUser) GetViewerInfo() *UserResponse {
    return &UserResponse{
        ID:             svw.ViewerID,
        Username:       svw.ViewerUsername,
        FullName:       svw.ViewerFullName,
        ProfilePicture: svw.ViewerProfilePicture,
        IsVerified:     svw.ViewerIsVerified,
    }
}

// IsRecent checks if the view was made recently (within last hour)
func (sv *StoryView) IsRecent() bool {
    return time.Since(sv.ViewedAt) < time.Hour
}

// GetTimeAgo returns a human-readable time since the view
func (sv *StoryView) GetTimeAgo() string {
    duration := time.Since(sv.ViewedAt)
    
    if duration < time.Minute {
        return "just now"
    } else if duration < time.Hour {
        minutes := int(duration.Minutes())
        if minutes == 1 {
            return "1 minute ago"
        }
        return fmt.Sprintf("%d minutes ago", minutes)
    } else if duration < 24*time.Hour {
        hours := int(duration.Hours())
        if hours == 1 {
            return "1 hour ago"
        }
        return fmt.Sprintf("%d hours ago", hours)
    } else {
        days := int(duration.Hours() / 24)
        if days == 1 {
            return "1 day ago"
        }
        return fmt.Sprintf("%d days ago", days)
    }
}
