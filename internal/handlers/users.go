package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/auth"
    "github.com/Abhiro0p/stories-backend/internal/models"
    "github.com/Abhiro0p/stories-backend/internal/storage"
    "github.com/Abhiro0p/stories-backend/pkg/validator"
)

// UserHandler handles user-related endpoints
type UserHandler struct {
    userStore   storage.UserStore
    followStore storage.FollowStore
    logger      *zap.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userStore storage.UserStore, followStore storage.FollowStore, logger *zap.Logger) *UserHandler {
    return &UserHandler{
        userStore:   userStore,
        followStore: followStore,
        logger:      logger.With(zap.String("handler", "user")),
    }
}

// GetCurrentUser gets the current authenticated user
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    c.JSON(http.StatusOK, user.ToResponse())
}

// UpdateCurrentUser updates the current user's profile
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    var req models.UserUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid update user request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
            "details": err.Error(),
        })
        return
    }

    // Validate request
    if err := validator.ValidateStruct(&req); err != nil {
        h.logger.Warn("Update user validation failed", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "validation_failed",
            "message": "Request validation failed",
            "details": err.Error(),
        })
        return
    }

    // Check username availability if changing username
    if req.Username != nil && *req.Username != user.Username {
        existingUser, err := h.userStore.GetByUsername(c.Request.Context(), *req.Username)
        if err == nil && existingUser != nil {
            c.JSON(http.StatusConflict, gin.H{
                "error":   "username_taken",
                "message": "Username is already taken",
            })
            return
        }
    }

    // Update user
    user.Update(req)

    // Save to database
    if err := h.userStore.Update(c.Request.Context(), user); err != nil {
        h.logger.Error("Failed to update user", 
            zap.String("user_id", user.ID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "update_failed",
            "message": "Failed to update user profile",
        })
        return
    }

    h.logger.Info("User profile updated successfully", 
        zap.String("user_id", user.ID.String()),
    )

    c.JSON(http.StatusOK, user.ToResponse())
}

// GetUser gets a user by ID
func (h *UserHandler) GetUser(c *gin.Context) {
    // Parse user ID
    userID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid user ID",
        })
        return
    }

    // Get user
    user, err := h.userStore.GetByID(c.Request.Context(), userID)
    if err != nil {
        if err == storage.ErrNotFound {
            c.JSON(http.StatusNotFound, gin.H{
                "error":   "not_found",
                "message": "User not found",
            })
            return
        }
        
        h.logger.Error("Failed to get user", 
            zap.String("user_id", userID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get user",
        })
        return
    }

    // Check if current user is following this user
    currentUser, ok := auth.GetCurrentUser(c)
    if ok {
        isFollowing, err := h.followStore.IsFollowing(c.Request.Context(), currentUser.ID, userID)
        if err == nil {
            user.IsFollowing = isFollowing
        }
    }

    c.JSON(http.StatusOK, user.ToResponse())
}

// SearchUsers searches for users
func (h *UserHandler) SearchUsers(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "missing_query",
            "message": "Search query is required",
        })
        return
    }

    // Parse pagination parameters
    limit := 20
    if l := c.Query("limit"); l != "" {
        if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
            limit = parsed
        }
    }

    offset := 0
    if o := c.Query("offset"); o != "" {
        if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
            offset = parsed
        }
    }

    // Search users
    users, err := h.userStore.Search(c.Request.Context(), query, limit, offset)
    if err != nil {
        h.logger.Error("Failed to search users", 
            zap.String("query", query),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "search_failed",
            "message": "Failed to search users",
        })
        return
    }

    // Convert to response format
    responses := make([]*models.UserResponse, len(users))
    for i, user := range users {
        responses[i] = user.ToResponse()
    }

    c.JSON(http.StatusOK, gin.H{
        "users": responses,
        "count": len(responses),
        "query": query,
    })
}

// FollowUser follows a user
func (h *UserHandler) FollowUser(c *gin.Context) {
    currentUser, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse target user ID
    targetUserID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid user ID",
        })
        return
    }

    // Can't follow yourself
    if currentUser.ID == targetUserID {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_action",
            "message": "Cannot follow yourself",
        })
        return
    }

    // Check if target user exists
    targetUser, err := h.userStore.GetByID(c.Request.Context(), targetUserID)
    if err != nil {
        if err == storage.ErrNotFound {
            c.JSON(http.StatusNotFound, gin.H{
                "error":   "not_found",
                "message": "User not found",
            })
            return
        }
        
        h.logger.Error("Failed to get target user for follow", 
            zap.String("target_user_id", targetUserID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get user",
        })
        return
    }

    // Check if already following
    isFollowing, err := h.followStore.IsFollowing(c.Request.Context(), currentUser.ID, targetUserID)
    if err != nil {
        h.logger.Error("Failed to check follow status", 
            zap.String("follower_id", currentUser.ID.String()),
            zap.String("followee_id", targetUserID.String()),
            zap.Error(err),
        )
    } else if isFollowing {
        c.JSON(http.StatusConflict, gin.H{
            "error":   "already_following",
            "message": "Already following this user",
        })
        return
    }

    // Create follow relationship
    follow := models.NewFollow(currentUser.ID, targetUserID)

    // Save follow
    if err := h.followStore.Create(c.Request.Context(), follow); err != nil {
        h.logger.Error("Failed to create follow relationship", 
            zap.String("follower_id", currentUser.ID.String()),
            zap.String("followee_id", targetUserID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "follow_failed",
            "message": "Failed to follow user",
        })
        return
    }

    h.logger.Info("User followed successfully", 
        zap.String("follower_id", currentUser.ID.String()),
        zap.String("followee_id", targetUserID.String()),
    )

    c.JSON(http.StatusCreated, gin.H{
        "message":     "User followed successfully",
        "followed_user": targetUser.ToResponse(),
    })
}

// UnfollowUser unfollows a user
func (h *UserHandler) UnfollowUser(c *gin.Context) {
    currentUser, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse target user ID
    targetUserID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid user ID",
        })
        return
    }

    // Can't unfollow yourself
    if currentUser.ID == targetUserID {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_action",
            "message": "Cannot unfollow yourself",
        })
        return
    }

    // Check if following
    isFollowing, err := h.followStore.IsFollowing(c.Request.Context(), currentUser.ID, targetUserID)
    if err != nil {
        h.logger.Error("Failed to check follow status", 
            zap.String("follower_id", currentUser.ID.String()),
            zap.String("followee_id", targetUserID.String()),
            zap.Error(err),
        )
    } else if !isFollowing {
        c.JSON(http.StatusNotFound, gin.H{
            "error":   "not_following",
            "message": "Not following this user",
        })
        return
    }

    // Unfollow user
    if err := h.followStore.DeleteByUsers(c.Request.Context(), currentUser.ID, targetUserID); err != nil {
        h.logger.Error("Failed to unfollow user", 
            zap.String("follower_id", currentUser.ID.String()),
            zap.String("followee_id", targetUserID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "unfollow_failed",
            "message": "Failed to unfollow user",
        })
        return
    }

    h.logger.Info("User unfollowed successfully", 
        zap.String("follower_id", currentUser.ID.String()),
        zap.String("followee_id", targetUserID.String()),
    )

    c.JSON(http.StatusOK, gin.H{
        "message": "User unfollowed successfully",
    })
}

// GetFollowers gets user's followers
func (h *UserHandler) GetFollowers(c *gin.Context) {
    // Parse user ID
    userID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid user ID",
        })
        return
    }

    // Parse pagination parameters
    limit := 20
    if l := c.Query("limit"); l != "" {
        if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
            limit = parsed
        }
    }

    offset := 0
    if o := c.Query("offset"); o != "" {
        if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
            offset = parsed
        }
    }

    // Get followers
    follows, err := h.followStore.GetFollowers(c.Request.Context(), userID, limit, offset)
    if err != nil {
        h.logger.Error("Failed to get followers", 
            zap.String("user_id", userID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get followers",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "followers": follows,
        "count":     len(follows),
    })
}

// GetFollowing gets users that the user is following
func (h *UserHandler) GetFollowing(c *gin.Context) {
    // Parse user ID
    userID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid user ID",
        })
        return
    }

    // Parse pagination parameters
    limit := 20
    if l := c.Query("limit"); l != "" {
        if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
            limit = parsed
        }
    }

    offset := 0
    if o := c.Query("offset"); o != "" {
        if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
            offset = parsed
        }
    }

    // Get following
    follows, err := h.followStore.GetFollowing(c.Request.Context(), userID, limit, offset)
    if err != nil {
        h.logger.Error("Failed to get following", 
            zap.String("user_id", userID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get following",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "following": follows,
        "count":     len(follows),
    })
}
