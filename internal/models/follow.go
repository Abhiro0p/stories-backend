package models

import (
    "time"

    "github.com/google/uuid"
)

// Follow represents a follow relationship between users
type Follow struct {
    ID         uuid.UUID `json:"id" db:"id"`
    FollowerID uuid.UUID `json:"follower_id" db:"follower_id"`
    FolloweeID uuid.UUID `json:"followee_id" db:"followee_id"`
    CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// FollowWithUser represents a follow relationship with user information
type FollowWithUser struct {
    Follow
    // For followers
    FollowerUsername       *string `json:"follower_username,omitempty" db:"follower_username"`
    FollowerFullName       *string `json:"follower_full_name,omitempty" db:"follower_full_name"`
    FollowerProfilePicture *string `json:"follower_profile_picture,omitempty" db:"follower_profile_picture"`
    FollowerIsVerified     *bool   `json:"follower_is_verified,omitempty" db:"follower_is_verified"`
    
    // For following
    FolloweeUsername       *string `json:"followee_username,omitempty" db:"followee_username"`
    FolloweeFullName       *string `json:"followee_full_name,omitempty" db:"followee_full_name"`
    FolloweeProfilePicture *string `json:"followee_profile_picture,omitempty" db:"followee_profile_picture"`
    FolloweeIsVerified     *bool   `json:"followee_is_verified,omitempty" db:"followee_is_verified"`
}

// FollowStats represents follow statistics for a user
type FollowStats struct {
    UserID         uuid.UUID `json:"user_id" db:"user_id"`
    FollowerCount  int       `json:"follower_count" db:"follower_count"`
    FollowingCount int       `json:"following_count" db:"following_count"`
}

// MutualFollowCheck represents mutual follow status between two users
type MutualFollowCheck struct {
    UserID1        uuid.UUID `json:"user_id_1"`
    UserID2        uuid.UUID `json:"user_id_2"`
    User1Follows2  bool      `json:"user1_follows_user2"`
    User2Follows1  bool      `json:"user2_follows_user1"`
    AreMutual      bool      `json:"are_mutual"`
}

// FollowSuggestion represents a user suggestion for following
type FollowSuggestion struct {
    UserID           uuid.UUID `json:"user_id" db:"user_id"`
    Username         string    `json:"username" db:"username"`
    FullName         *string   `json:"full_name" db:"full_name"`
    ProfilePicture   *string   `json:"profile_picture" db:"profile_picture"`
    IsVerified       bool      `json:"is_verified" db:"is_verified"`
    MutualFollowers  int       `json:"mutual_followers" db:"mutual_followers"`
    SuggestionReason string    `json:"suggestion_reason" db:"suggestion_reason"`
}

// NewFollow creates a new follow relationship
func NewFollow(followerID, followeeID uuid.UUID) *Follow {
    return &Follow{
        ID:         uuid.New(),
        FollowerID: followerID,
        FolloweeID: followeeID,
        CreatedAt:  time.Now(),
    }
}

// GetFollowerInfo extracts follower information
func (fw *FollowWithUser) GetFollowerInfo() *UserResponse {
    if fw.FollowerUsername == nil {
        return nil
    }
    
    return &UserResponse{
        ID:             fw.FollowerID,
        Username:       *fw.FollowerUsername,
        FullName:       fw.FollowerFullName,
        ProfilePicture: fw.FollowerProfilePicture,
        IsVerified:     fw.FollowerIsVerified != nil && *fw.FollowerIsVerified,
    }
}

// GetFolloweeInfo extracts followee information
func (fw *FollowWithUser) GetFolloweeInfo() *UserResponse {
    if fw.FolloweeUsername == nil {
        return nil
    }
    
    return &UserResponse{
        ID:             fw.FolloweeID,
        Username:       *fw.FolloweeUsername,
        FullName:       fw.FolloweeFullName,
        ProfilePicture: fw.FolloweeProfilePicture,
        IsVerified:     fw.FolloweeIsVerified != nil && *fw.FolloweeIsVerified,
    }
}

// CheckMutual checks if the follow relationship is mutual
func (mfc *MutualFollowCheck) CheckMutual() {
    mfc.AreMutual = mfc.User1Follows2 && mfc.User2Follows1
}
