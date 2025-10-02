package models

import (
    "time"

    "github.com/google/uuid"
)

// ReactionType represents the type of reaction
type ReactionType string

const (
    ReactionLike    ReactionType = "like"
    ReactionLove    ReactionType = "love"
    ReactionLaugh   ReactionType = "laugh"
    ReactionWow     ReactionType = "wow"
    ReactionSad     ReactionType = "sad"
    ReactionAngry   ReactionType = "angry"
    ReactionFire    ReactionType = "fire"
    ReactionHundred ReactionType = "hundred"
)

// Reaction represents a user's reaction to a story
type Reaction struct {
    ID        uuid.UUID    `json:"id" db:"id"`
    StoryID   uuid.UUID    `json:"story_id" db:"story_id"`
    UserID    uuid.UUID    `json:"user_id" db:"user_id"`
    Type      ReactionType `json:"type" db:"type"`
    CreatedAt time.Time    `json:"created_at" db:"created_at"`
    UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
}

// ReactionWithUser represents a reaction with user information
type ReactionWithUser struct {
    Reaction
    Username       string  `json:"username" db:"username"`
    FullName       *string `json:"full_name" db:"full_name"`
    ProfilePicture *string `json:"profile_picture" db:"profile_picture"`
    IsVerified     bool    `json:"is_verified" db:"is_verified"`
}

// ReactionCreateRequest represents the request to create a reaction
type ReactionCreateRequest struct {
    Type ReactionType `json:"type" validate:"required,reaction_type"`
}

// ReactionUpdateRequest represents the request to update a reaction
type ReactionUpdateRequest struct {
    Type ReactionType `json:"type" validate:"required,reaction_type"`
}

// ReactionSummary represents a summary of reactions for a story
type ReactionSummary struct {
    StoryID        uuid.UUID                `json:"story_id"`
    TotalReactions int                      `json:"total_reactions"`
    ReactionCounts map[ReactionType]int     `json:"reaction_counts"`
    RecentReactions []ReactionWithUser      `json:"recent_reactions"`
    UserReaction   *ReactionType            `json:"user_reaction,omitempty"`
}

// NewReaction creates a new reaction
func NewReaction(storyID, userID uuid.UUID, reactionType ReactionType) *Reaction {
    now := time.Now()
    
    return &Reaction{
        ID:        uuid.New(),
        StoryID:   storyID,
        UserID:    userID,
        Type:      reactionType,
        CreatedAt: now,
        UpdatedAt: now,
    }
}

// Update updates the reaction type
func (r *Reaction) Update(reactionType ReactionType) {
    r.Type = reactionType
    r.UpdatedAt = time.Now()
}

// GetUserInfo extracts user information from ReactionWithUser
func (rw *ReactionWithUser) GetUserInfo() *UserResponse {
    return &UserResponse{
        ID:             rw.UserID,
        Username:       rw.Username,
        FullName:       rw.FullName,
        ProfilePicture: rw.ProfilePicture,
        IsVerified:     rw.IsVerified,
    }
}

// ValidateReactionType validates if the reaction type is valid
func ValidateReactionType(reactionType string) bool {
    validTypes := []ReactionType{
        ReactionLike,
        ReactionLove,
        ReactionLaugh,
        ReactionWow,
        ReactionSad,
        ReactionAngry,
        ReactionFire,
        ReactionHundred,
    }
    
    for _, valid := range validTypes {
        if reactionType == string(valid) {
            return true
        }
    }
    return false
}

// GetReactionEmoji returns the emoji representation of the reaction
func (rt ReactionType) GetEmoji() string {
    switch rt {
    case ReactionLike:
        return "👍"
    case ReactionLove:
        return "❤️"
    case ReactionLaugh:
        return "😂"
    case ReactionWow:
        return "😮"
    case ReactionSad:
        return "😢"
    case ReactionAngry:
        return "😡"
    case ReactionFire:
        return "🔥"
    case ReactionHundred:
        return "💯"
    default:
        return "👍"
    }
}

// GetReactionDisplayName returns the display name of the reaction
func (rt ReactionType) GetDisplayName() string {
    switch rt {
    case ReactionLike:
        return "Like"
    case ReactionLove:
        return "Love"
    case ReactionLaugh:
        return "Laugh"
    case ReactionWow:
        return "Wow"
    case ReactionSad:
        return "Sad"
    case ReactionAngry:
        return "Angry"
    case ReactionFire:
        return "Fire"
    case ReactionHundred:
        return "100"
    default:
        return "Like"
    }
}

// IsPositive returns true if the reaction is generally positive
func (rt ReactionType) IsPositive() bool {
    positiveReactions := []ReactionType{
        ReactionLike,
        ReactionLove,
        ReactionLaugh,
        ReactionWow,
        ReactionFire,
        ReactionHundred,
    }
    
    for _, positive := range positiveReactions {
        if rt == positive {
            return true
        }
    }
    return false
}
