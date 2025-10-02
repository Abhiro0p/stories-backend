package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/auth"
    "github.com/Abhiro0p/stories-backend/internal/media"
    "github.com/Abhiro0p/stories-backend/pkg/validator"
)

// MediaHandler handles media upload endpoints
type MediaHandler struct {
    mediaService *media.Service
    logger       *zap.Logger
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(mediaService *media.Service, logger *zap.Logger) *MediaHandler {
    return &MediaHandler{
        mediaService: mediaService,
        logger:       logger.With(zap.String("handler", "media")),
    }
}

// UploadURLRequest represents request for presigned upload URL
type UploadURLRequest struct {
    Filename    string `json:"filename" validate:"required,max=255"`
    ContentType string `json:"content_type" validate:"required"`
    Size        int64  `json:"size" validate:"required,min=1,max=10485760"` // Max 10MB
}

// GetUploadURL generates a presigned URL for media upload
func (h *MediaHandler) GetUploadURL(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    var req UploadURLRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.logger.Warn("Invalid upload URL request", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": "Invalid request body",
            "details": err.Error(),
        })
        return
    }

    // Validate request
    if err := validator.ValidateStruct(&req); err != nil {
        h.logger.Warn("Upload URL validation failed", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "validation_failed",
            "message": "Request validation failed",
            "details": err.Error(),
        })
        return
    }

    // Validate content type
    if !h.isValidContentType(req.ContentType) {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_content_type",
            "message": "Content type not supported",
        })
        return
    }

    // Generate presigned URL
    response, err := h.mediaService.GenerateUploadURL(c.Request.Context(), user.ID, req.Filename, req.ContentType)
    if err != nil {
        h.logger.Error("Failed to generate upload URL", 
            zap.String("user_id", user.ID.String()),
            zap.String("filename", req.Filename),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "upload_url_failed",
            "message": "Failed to generate upload URL",
        })
        return
    }

    h.logger.Info("Upload URL generated successfully", 
        zap.String("user_id", user.ID.String()),
        zap.String("key", response.Key),
        zap.String("filename", req.Filename),
    )

    c.JSON(http.StatusOK, response)
}

// GetMedia retrieves a media file
func (h *MediaHandler) GetMedia(c *gin.Context) {
    key := c.Param("key")
    if key == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "missing_key",
            "message": "Media key is required",
        })
        return
    }

    // Generate presigned download URL
    url, err := h.mediaService.GenerateDownloadURL(c.Request.Context(), key)
    if err != nil {
        h.logger.Error("Failed to generate download URL", 
            zap.String("key", key),
            zap.Error(err),
        )
        c.JSON(http.StatusNotFound, gin.H{
            "error":   "media_not_found",
            "message": "Media file not found",
        })
        return
    }

    // Redirect to presigned URL
    c.Redirect(http.StatusTemporaryRedirect, url)
}

// DeleteMedia deletes a media file
func (h *MediaHandler) DeleteMedia(c *gin.Context) {
    user, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "User not found in context",
        })
        return
    }

    key := c.Param("key")
    if key == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "missing_key",
            "message": "Media key is required",
        })
        return
    }

    // Delete media file
    if err := h.mediaService.DeleteMedia(c.Request.Context(), user.ID, key); err != nil {
        h.logger.Error("Failed to delete media", 
            zap.String("user_id", user.ID.String()),
            zap.String("key", key),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "delete_failed",
            "message": "Failed to delete media file",
        })
        return
    }

    h.logger.Info("Media deleted successfully", 
        zap.String("user_id", user.ID.String()),
        zap.String("key", key),
    )

    c.JSON(http.StatusOK, gin.H{
        "message": "Media deleted successfully",
    })
}

// isValidContentType checks if the content type is supported
func (h *MediaHandler) isValidContentType(contentType string) bool {
    allowedTypes := []string{
        "image/jpeg",
        "image/png",
        "image/gif",
        "image/webp",
        "video/mp4",
        "video/quicktime",
        "video/x-msvideo", // AVI
    }

    for _, allowed := range allowedTypes {
        if contentType == allowed {
            return true
        }
    }

    return false
}
