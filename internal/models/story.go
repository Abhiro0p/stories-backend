package models

import (
    "time"

    "github.com/google/uuid"
)

// StoryType represents the type of story content
type StoryType string

const (
    StoryTypeText  StoryType = "text"
    StoryTypeImage StoryType = "image"
    StoryTypeVideo StoryType = "video"
)

// StoryVisibility represents who can see the story
type StoryVisibility string

const (
    VisibilityPublic  StoryVisibility = "public"
    VisibilityPrivate StoryVisibility = "private"
    VisibilityFriends StoryVisibility = "friends"
)

// Story represents a story in the system
type Story struct {
    ID            uuid.UUID       `json:"id" db:"id"`
    AuthorID      uuid.UUID       `json:"author_id" db:"author_id"`
    Type          StoryType       `json:"type" db:"type"`
    Text          *string         `json:"text,omitempty" db:"text"`
    MediaURL      *string         `json:"media_url,omitempty" db:"media_url"`
    MediaKey      *uuid.UUID      `json:"media_key,omitempty" db:"media_key"`
    Visibility    StoryVisibility `json:"visibility" db:"visibility"`
    ViewCount     int             `json:"view_count" db:"view_count"`
    ReactionCount int             `json:"reaction_count" db:"reaction_count"`
    ExpiresAt     time.Time       `json:"expires_at" db:"expires_at"`
    CreatedAt     time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
    DeletedAt     *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
    
    // Additional fields not stored in DB
    Author        *UserResponse   `json:"author,omitempty" db:"-"`
    IsViewed      bool            `json:"is_viewed,omitempty" db:"-"`
    UserReaction  *ReactionType   `json:"user_reaction,omitempty" db:"-"`
    TimeRemaining *time.Duration  `json:"time_remaining,omitempty" db:"-"`
}

// StoryCreateRequest represents the request to create a new story
type StoryCreateRequest struct {
    Type       StoryType       `json:"type" validate:"required,oneof=text image video"`
    Text       *string         `json:"text,omitempty" validate:"omitempty,story_text"`
    MediaKey   *uuid.UUID      `json:"media_key,omitempty"`
    Visibility StoryVisibility `json:"visibility" validate:"required,visibility"`
    ExpiresIn  *int            `json:"expires_in,omitempty" validate:"omitempty,min=3600,max=604800"` // 1 hour to 7 days in seconds
}

// StoryUpdateRequest represents the request to update a story
type StoryUpdateRequest struct {
    Text       *string         `json:"text,omitempty" validate:"omitempty,story_text"`
    Visibility StoryVisibility `json:"visibility,omitempty" validate:"omitempty,visibility"`
}

// StoryWithAuthor represents a story with embedded author information
type StoryWithAuthor struct {
    Story
    AuthorUsername       string  `db:"author_username"`
    AuthorFullName       *string `db:"author_full_name"`
    AuthorProfilePicture *string `db:"author_profile_picture"`
    AuthorIsVerified     bool    `db:"author_is_verified"`
}

// NewStory creates a new story
func NewStory(authorID uuid.UUID, req StoryCreateRequest) *Story {
    id := uuid.New()
    now := time.Now()
    
    // Default expiry is 24 hours
    expiresIn := 24 * 60 * 60 // 24 hours in seconds
    if req.ExpiresIn != nil {
        expiresIn = *req.ExpiresIn
    }
    
    story := &Story{
        ID:         id,
        AuthorID:   authorID,
        Type:       req.Type,
        Text:       req.Text,
        MediaKey:   req.MediaKey,
        Visibility: req.Visibility,
        ExpiresAt:  now.Add(time.Duration(expiresIn) * time.Second),
        CreatedAt:  now,
        UpdatedAt:  now,
    }
    
    return story
}

// Update updates story fields from request
func (s *Story) Update(req StoryUpdateRequest) {
    if req.Text != nil {
        s.Text = req.Text
    }
    if req.Visibility != "" {
        s.Visibility = req.Visibility
    }
    s.UpdatedAt = time.Now()
}

// IsExpired checks if the story has expired
func (s *Story) IsExpired() bool {
    return time.Now().After(s.ExpiresAt)
}

// CanView checks if a user can view this story
func (s *Story) CanView(userID *uuid.UUID) bool {
    // Public stories can be viewed by anyone
    if s.Visibility == VisibilityPublic {
        return true
    }
    
    // Only the author can view private stories
    if s.Visibility == VisibilityPrivate {
        return userID != nil && *userID == s.AuthorID
    }
    
    // For friends-only stories, we need to check follow relationship
    // This would typically require a database query, so we'll assume
    // the caller has already verified the relationship
    if s.Visibility == VisibilityFriends {
        return userID != nil
    }
    
    return false
}

// CanEdit checks if a user can edit this story
func (s *Story) CanEdit(userID uuid.UUID) bool {
    return userID == s.AuthorID
}

// CanDelete checks if a user can delete this story
func (s *Story) CanDelete(userID uuid.UUID) bool {
    return userID == s.AuthorID
}

// GetTimeRemaining returns the time remaining before expiry
func (s *Story) GetTimeRemaining() time.Duration {
    remaining := time.Until(s.ExpiresAt)
    if remaining < 0 {
        return 0
    }
    return remaining
}

// GetAuthorInfo extracts author information from StoryWithAuthor
func (swa *StoryWithAuthor) GetAuthorInfo() *UserResponse {
    return &UserResponse{
        ID:             swa.AuthorID,
        Username:       swa.AuthorUsername,
        FullName:       swa.AuthorFullName,
        ProfilePicture: swa.AuthorProfilePicture,
        IsVerified:     swa.AuthorIsVerified,
    }
}

// ValidateStoryType validates if the story type is valid
func ValidateStoryType(storyType string) bool {
    validTypes := []string{string(StoryTypeText), string(StoryTypeImage), string(StoryTypeVideo)}
    for _, valid := range validTypes {
        if storyType == valid {
            return true
        }
    }
    return false
}

// ValidateVisibility validates if the visibility is valid
func ValidateVisibility(visibility string) bool {
    validVisibilities := []string{string(VisibilityPublic), string(VisibilityPrivate), string(VisibilityFriends)}
    for _, valid := range validVisibilities {
        if visibility == valid {
            return true
        }
    }
    return false
}
