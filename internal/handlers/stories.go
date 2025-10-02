package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/auth"
    "github.com/Abhiro0p/stories-backend/internal/models"
    "github.com/Abhiro0p/stories-backend/internal/realtime"
    "github.com/Abhiro0p/stories-backend/internal/storage"
    "github.com/Abhiro0p/stories-backend/pkg/validator"
)

// StoryHandler handles story-related endpoints
type StoryHandler struct {
    storyStore    storage.StoryStore
    viewStore     storage.ViewStore
    reactionStore storage.ReactionStore
    wsHub         *realtime.Hub
    logger        *zap.Logger
}

// NewStoryHandler creates a new story handler
func NewStoryHandler(
    storyStore storage.StoryStore,
    viewStore storage.ViewStore,
    reactionStore storage.ReactionStore,
    wsHub *realtime.Hub,
    logger *zap.Logger,
) *StoryHandler {
    return &StoryHandler{
        storyStore:    storyStore,
        viewStore:     viewStore,
        reactionStore: reactionStore,
        wsHub:         wsHub,
        logger:        logger.With(zap.String("handler", "story")),
    }
}

// CreateStory creates a new story
func (h *StoryHandler) CreateStory(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    var req models.StoryCreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid create story request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
            "details": err.Error(),
        })
        return
    }

    // Validate request
    if err := validator.ValidateStruct(&req); err != nil {
        h.logger.Warn("Create story validation failed", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "validation_failed",
            "message": "Request validation failed",
            "details": err.Error(),
        })
        return
    }

    // Create story
    story := models.NewStory(user.ID, req)

    // Save to database
    if err := h.storyStore.Create(c.Request.Context(), story); err != nil {
        h.logger.Error("Failed to create story", 
            zap.String("user_id", user.ID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "create_failed",
            "message": "Failed to create story",
        })
        return
    }

    h.logger.Info("Story created successfully", 
        zap.String("story_id", story.ID.String()),
        zap.String("author_id", user.ID.String()),
        zap.String("type", string(story.Type)),
    )

    // Send real-time notification
    if h.wsHub != nil {
        event := &realtime.Event{
            Type: realtime.EventStoryCreated,
            Payload: gin.H{
                "story":  story,
                "author": user.ToResponse(),
            },
        }
        h.wsHub.BroadcastToFollowers(user.ID, event)
    }

    c.JSON(http.StatusCreated, story)
}

// GetStories gets stories feed
func (h *StoryHandler) GetStories(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse query parameters
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

    // Get stories feed
    stories, err := h.storyStore.GetFeed(c.Request.Context(), user.ID, limit, offset)
    if err != nil {
        h.logger.Error("Failed to get stories feed", 
            zap.String("user_id", user.ID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get stories",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "stories": stories,
        "count":   len(stories),
    })
}

// GetStory gets a specific story
func (h *StoryHandler) GetStory(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse story ID
    storyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid story ID",
        })
        return
    }

    // Get story
    story, err := h.storyStore.GetByID(c.Request.Context(), storyID)
    if err != nil {
        if err == storage.ErrNotFound {
            c.JSON(http.StatusNotFound, gin.H{
                "error":   "not_found",
                "message": "Story not found",
            })
            return
        }
        
        h.logger.Error("Failed to get story", 
            zap.String("story_id", storyID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get story",
        })
        return
    }

    // Check if user can view this story
    if !story.CanView(&user.ID) {
        c.JSON(http.StatusForbidden, gin.H{
            "error":   "forbidden",
            "message": "You don't have permission to view this story",
        })
        return
    }

    c.JSON(http.StatusOK, story)
}

// UpdateStory updates a story
func (h *StoryHandler) UpdateStory(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse story ID
    storyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid story ID",
        })
        return
    }

    var req models.StoryUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid update story request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
        })
        return
    }

    // Get existing story
    story, err := h.storyStore.GetByID(c.Request.Context(), storyID)
    if err != nil {
        if err == storage.ErrNotFound {
            c.JSON(http.StatusNotFound, gin.H{
                "error":   "not_found",
                "message": "Story not found",
            })
            return
        }
        
        h.logger.Error("Failed to get story for update", 
            zap.String("story_id", storyID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get story",
        })
        return
    }

    // Check if user can edit this story
    if !story.CanEdit(user.ID) {
        c.JSON(http.StatusForbidden, gin.H{
            "error":   "forbidden",
            "message": "You don't have permission to edit this story",
        })
        return
    }

    // Update story
    story.Update(req)

    // Save to database
    if err := h.storyStore.Update(c.Request.Context(), story); err != nil {
        h.logger.Error("Failed to update story", 
            zap.String("story_id", storyID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "update_failed",
            "message": "Failed to update story",
        })
        return
    }

    h.logger.Info("Story updated successfully", 
        zap.String("story_id", storyID.String()),
        zap.String("user_id", user.ID.String()),
    )

    c.JSON(http.StatusOK, story)
}

// DeleteStory deletes a story
func (h *StoryHandler) DeleteStory(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse story ID
    storyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid story ID",
        })
        return
    }

    // Get story to check permissions
    story, err := h.storyStore.GetByID(c.Request.Context(), storyID)
    if err != nil {
        if err == storage.ErrNotFound {
            c.JSON(http.StatusNotFound, gin.H{
                "error":   "not_found",
                "message": "Story not found",
            })
            return
        }
        
        h.logger.Error("Failed to get story for deletion", 
            zap.String("story_id", storyID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get story",
        })
        return
    }

    // Check if user can delete this story
    if !story.CanDelete(user.ID) {
        c.JSON(http.StatusForbidden, gin.H{
            "error":   "forbidden",
            "message": "You don't have permission to delete this story",
        })
        return
    }

    // Delete story (soft delete)
    if err := h.storyStore.Delete(c.Request.Context(), storyID); err != nil {
        h.logger.Error("Failed to delete story", 
            zap.String("story_id", storyID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "delete_failed",
            "message": "Failed to delete story",
        })
        return
    }

    h.logger.Info("Story deleted successfully", 
        zap.String("story_id", storyID.String()),
        zap.String("user_id", user.ID.String()),
    )

    c.JSON(http.StatusOK, gin.H{
        "message": "Story deleted successfully",
    })
}

// ViewStory marks a story as viewed
func (h *StoryHandler) ViewStory(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse story ID
    storyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid story ID",
        })
        return
    }

    // Create story view
    ipAddress := c.ClientIP()
    userAgent := c.GetHeader("User-Agent")
    
    view := models.NewStoryView(storyID, user.ID, &ipAddress, &userAgent)

    // Save view
    if err := h.viewStore.Create(c.Request.Context(), view); err != nil {
        h.logger.Error("Failed to create story view", 
            zap.String("story_id", storyID.String()),
            zap.String("user_id", user.ID.String()),
            zap.Error(err),
        )
        // Don't return error for views, as it's not critical
    } else {
        h.logger.Info("Story view recorded", 
            zap.String("story_id", storyID.String()),
            zap.String("viewer_id", user.ID.String()),
        )

        // Send real-time notification to story author
        if h.wsHub != nil {
            // Get story to get author ID
            story, err := h.storyStore.GetByID(c.Request.Context(), storyID)
            if err == nil && story.AuthorID != user.ID {
                event := &realtime.Event{
                    Type: realtime.EventStoryViewed,
                    Payload: gin.H{
                        "story_id": storyID,
                        "viewer":   user.ToResponse(),
                    },
                }
                h.wsHub.SendToUser(story.AuthorID, event)
            }
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Story viewed",
    })
}

// GetStoryViews gets views for a story
func (h *StoryHandler) GetStoryViews(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse story ID
    storyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid story ID",
        })
        return
    }

    // Check if user owns this story
    story, err := h.storyStore.GetByID(c.Request.Context(), storyID)
    if err != nil {
        if err == storage.ErrNotFound {
            c.JSON(http.StatusNotFound, gin.H{
                "error":   "not_found",
                "message": "Story not found",
            })
            return
        }
        
        h.logger.Error("Failed to get story for views", 
            zap.String("story_id", storyID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get story",
        })
        return
    }

    // Only story author can see views
    if story.AuthorID != user.ID {
        c.JSON(http.StatusForbidden, gin.H{
            "error":   "forbidden",
            "message": "You can only view your own story views",
        })
        return
    }

    // Get views
    views, err := h.viewStore.GetByStoryID(c.Request.Context(), storyID, 50, 0)
    if err != nil {
        h.logger.Error("Failed to get story views", 
            zap.String("story_id", storyID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get story views",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "views": views,
        "count": len(views),
    })
}

// GetStoryReactions gets reactions for a story
func (h *StoryHandler) GetStoryReactions(c *gin.Context) {
    // Parse story ID
    storyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid story ID",
        })
        return
    }

    // Get reactions
    reactions, err := h.reactionStore.GetByStoryID(c.Request.Context(), storyID, 100, 0)
    if err != nil {
        h.logger.Error("Failed to get story reactions", 
            zap.String("story_id", storyID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get reactions",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "reactions": reactions,
        "count":     len(reactions),
    })
}

// AddReaction adds a reaction to a story
func (h *StoryHandler) AddReaction(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse story ID
    storyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid story ID",
        })
        return
    }

    var req models.ReactionCreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid add reaction request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
        })
        return
    }

    // Validate reaction type
    if !models.ValidateReactionType(string(req.Type)) {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_reaction",
            "message": "Invalid reaction type",
        })
        return
    }

    // Create reaction
    reaction := models.NewReaction(storyID, user.ID, req.Type)

    // Save reaction
    if err := h.reactionStore.Create(c.Request.Context(), reaction); err != nil {
        h.logger.Error("Failed to add reaction", 
            zap.String("story_id", storyID.String()),
            zap.String("user_id", user.ID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "create_failed",
            "message": "Failed to add reaction",
        })
        return
    }

    h.logger.Info("Reaction added successfully", 
        zap.String("reaction_id", reaction.ID.String()),
        zap.String("story_id", storyID.String()),
        zap.String("user_id", user.ID.String()),
        zap.String("type", string(req.Type)),
    )

    // Send real-time notification
    if h.wsHub != nil {
        // Get story to get author ID
        story, err := h.storyStore.GetByID(c.Request.Context(), storyID)
        if err == nil && story.AuthorID != user.ID {
            event := &realtime.Event{
                Type: realtime.EventStoryReaction,
                Payload: gin.H{
                    "story_id": storyID,
                    "reaction": reaction,
                    "user":     user.ToResponse(),
                },
            }
            h.wsHub.SendToUser(story.AuthorID, event)
        }
    }

    c.JSON(http.StatusCreated, reaction)
}

// UpdateReaction updates a reaction
func (h *StoryHandler) UpdateReaction(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse reaction ID
    reactionID, err := uuid.Parse(c.Param("reaction_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid reaction ID",
        })
        return
    }

    var req models.ReactionUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid update reaction request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
        })
        return
    }

    // Get existing reaction
    reaction, err := h.reactionStore.GetByID(c.Request.Context(), reactionID)
    if err != nil {
        if err == storage.ErrNotFound {
            c.JSON(http.StatusNotFound, gin.H{
                "error":   "not_found",
                "message": "Reaction not found",
            })
            return
        }
        
        h.logger.Error("Failed to get reaction for update", 
            zap.String("reaction_id", reactionID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get reaction",
        })
        return
    }

    // Check if user owns this reaction
    if reaction.UserID != user.ID {
        c.JSON(http.StatusForbidden, gin.H{
            "error":   "forbidden",
            "message": "You can only update your own reactions",
        })
        return
    }

    // Validate reaction type
    if !models.ValidateReactionType(string(req.Type)) {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_reaction",
            "message": "Invalid reaction type",
        })
        return
    }

    // Update reaction
    reaction.Update(req.Type)

    // Save reaction
    if err := h.reactionStore.Update(c.Request.Context(), reaction); err != nil {
        h.logger.Error("Failed to update reaction", 
            zap.String("reaction_id", reactionID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "update_failed",
            "message": "Failed to update reaction",
        })
        return
    }

    h.logger.Info("Reaction updated successfully", 
        zap.String("reaction_id", reactionID.String()),
        zap.String("user_id", user.ID.String()),
    )

    c.JSON(http.StatusOK, reaction)
}

// RemoveReaction removes a reaction
func (h *StoryHandler) RemoveReaction(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    // Parse reaction ID
    reactionID, err := uuid.Parse(c.Param("reaction_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_id",
            "message": "Invalid reaction ID",
        })
        return
    }

    // Get reaction to check ownership
    reaction, err := h.reactionStore.GetByID(c.Request.Context(), reactionID)
    if err != nil {
        if err == storage.ErrNotFound {
            c.JSON(http.StatusNotFound, gin.H{
                "error":   "not_found",
                "message": "Reaction not found",
            })
            return
        }
        
        h.logger.Error("Failed to get reaction for deletion", 
            zap.String("reaction_id", reactionID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "fetch_failed",
            "message": "Failed to get reaction",
        })
        return
    }

    // Check if user owns this reaction
    if reaction.UserID != user.ID {
        c.JSON(http.StatusForbidden, gin.H{
            "error":   "forbidden",
            "message": "You can only remove your own reactions",
        })
        return
    }

    // Delete reaction
    if err := h.reactionStore.Delete(c.Request.Context(), reactionID); err != nil {
        h.logger.Error("Failed to delete reaction", 
            zap.String("reaction_id", reactionID.String()),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "delete_failed",
            "message": "Failed to remove reaction",
        })
        return
    }

    h.logger.Info("Reaction removed successfully", 
        zap.String("reaction_id", reactionID.String()),
        zap.String("user_id", user.ID.String()),
    )

    c.JSON(http.StatusOK, gin.H{
        "message": "Reaction removed successfully",
    })
}
