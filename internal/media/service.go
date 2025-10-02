package media

import (
    "context"
    "fmt"
    "path/filepath"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// Service handles media storage operations
type Service struct {
    client *minio.Client
    bucket string
    config *config.Config
    logger *zap.Logger
}

// UploadURLResponse represents the response for upload URL generation
type UploadURLResponse struct {
    UploadURL string `json:"upload_url"`
    Key       string `json:"key"`
    ExpiresAt int64  `json:"expires_at"`
}

// NewService creates a new media service
func NewService(cfg *config.Config, logger *zap.Logger) (*Service, error) {
    // Initialize MinIO client
    client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
        Secure: cfg.MinIOUseSSL,
        Region: cfg.MinIORegion,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
    }

    service := &Service{
        client: client,
        bucket: cfg.MinIOBucket,
        config: cfg,
        logger: logger.With(zap.String("component", "media_service")),
    }

    // Ensure bucket exists
    if err := service.ensureBucket(context.Background()); err != nil {
        return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
    }

    logger.Info("Media service initialized successfully",
        zap.String("endpoint", cfg.MinIOEndpoint),
        zap.String("bucket", cfg.MinIOBucket),
        zap.Bool("ssl", cfg.MinIOUseSSL),
    )

    return service, nil
}

// GenerateUploadURL generates a presigned URL for uploading media
func (s *Service) GenerateUploadURL(ctx context.Context, userID uuid.UUID, filename, contentType string) (*UploadURLResponse, error) {
    // Generate unique key for the file
    key := s.generateKey(userID, filename)

    // Set expiry time
    expiry := time.Duration(s.config.PresignedURLExpiryMinutes) * time.Minute

    // Generate presigned POST URL
    uploadURL, err := s.client.PresignedPutObject(ctx, s.bucket, key, expiry)
    if err != nil {
        return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
    }

    s.logger.Info("Generated upload URL",
        zap.String("user_id", userID.String()),
        zap.String("key", key),
        zap.String("filename", filename),
        zap.Duration("expiry", expiry),
    )

    return &UploadURLResponse{
        UploadURL: uploadURL.String(),
        Key:       key,
        ExpiresAt: time.Now().Add(expiry).Unix(),
    }, nil
}

// GenerateDownloadURL generates a presigned URL for downloading media
func (s *Service) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
    // Check if object exists
    _, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
    if err != nil {
        return "", fmt.Errorf("object not found: %w", err)
    }

    // Generate presigned GET URL
    expiry := time.Duration(s.config.PresignedURLExpiryMinutes) * time.Minute
    downloadURL, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
    if err != nil {
        return "", fmt.Errorf("failed to generate download URL: %w", err)
    }

    s.logger.Debug("Generated download URL",
        zap.String("key", key),
        zap.Duration("expiry", expiry),
    )

    return downloadURL.String(), nil
}

// DeleteMedia deletes a media file
func (s *Service) DeleteMedia(ctx context.Context, userID uuid.UUID, key string) error {
    // Verify that the user owns this media file
    if !s.userOwnsMedia(userID, key) {
        return fmt.Errorf("user does not own this media file")
    }

    // Delete the object
    err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
    if err != nil {
        return fmt.Errorf("failed to delete media: %w", err)
    }

    s.logger.Info("Media deleted successfully",
        zap.String("user_id", userID.String()),
        zap.String("key", key),
    )

    return nil
}

// GetMediaInfo gets information about a media file
func (s *Service) GetMediaInfo(ctx context.Context, key string) (*minio.ObjectInfo, error) {
    info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
    if err != nil {
        return nil, fmt.Errorf("failed to get media info: %w", err)
    }

    return &info, nil
}

// ListUserMedia lists all media files for a user
func (s *Service) ListUserMedia(ctx context.Context, userID uuid.UUID) ([]minio.ObjectInfo, error) {
    prefix := fmt.Sprintf("users/%s/", userID.String())
    
    objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
        Prefix:    prefix,
        Recursive: true,
    })

    var objects []minio.ObjectInfo
    for object := range objectCh {
        if object.Err != nil {
            return nil, fmt.Errorf("error listing objects: %w", object.Err)
        }
        objects = append(objects, object)
    }

    return objects, nil
}

// generateKey generates a unique key for storing media files
func (s *Service) generateKey(userID uuid.UUID, filename string) string {
    // Extract file extension
    ext := filepath.Ext(filename)
    
    // Generate unique filename
    uniqueID := uuid.New().String()
    
    // Create key with user folder structure
    key := fmt.Sprintf("users/%s/%s%s", userID.String(), uniqueID, ext)
    
    return key
}

// userOwnsMedia checks if a user owns a media file based on the key
func (s *Service) userOwnsMedia(userID uuid.UUID, key string) bool {
    expectedPrefix := fmt.Sprintf("users/%s/", userID.String())
    return strings.HasPrefix(key, expectedPrefix)
}

// ensureBucket ensures that the bucket exists
func (s *Service) ensureBucket(ctx context.Context) error {
    // Check if bucket exists
    exists, err := s.client.BucketExists(ctx, s.bucket)
    if err != nil {
        return fmt.Errorf("failed to check bucket existence: %w", err)
    }

    if !exists {
        // Create bucket
        err = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{
            Region: s.config.MinIORegion,
        })
        if err != nil {
            return fmt.Errorf("failed to create bucket: %w", err)
        }

        s.logger.Info("Created new bucket", zap.String("bucket", s.bucket))

        // Set bucket policy to allow public read for media files
        policy := fmt.Sprintf(`{
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Principal": "*",
                    "Action": ["s3:GetObject"],
                    "Resource": ["arn:aws:s3:::%s/*"]
                }
            ]
        }`, s.bucket)

        err = s.client.SetBucketPolicy(ctx, s.bucket, policy)
        if err != nil {
            s.logger.Warn("Failed to set bucket policy", zap.Error(err))
            // Don't fail if policy setting fails
        }
    }

    return nil
}

// Health checks the health of the media service
func (s *Service) Health(ctx context.Context) error {
    // Try to list objects in the bucket (with limit to avoid overhead)
    objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
        MaxKeys: 1,
    })

    // Check if we can list objects without error
    for object := range objectCh {
        if object.Err != nil {
            return fmt.Errorf("media service health check failed: %w", object.Err)
        }
        break // Only check the first object
    }

    return nil
}

// GetStorageStats returns storage statistics
func (s *Service) GetStorageStats(ctx context.Context) (map[string]interface{}, error) {
    stats := make(map[string]interface{})

    // List all objects to calculate stats
    objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
        Recursive: true,
    })

    var totalSize int64
    var objectCount int64

    for object := range objectCh {
        if object.Err != nil {
            return nil, fmt.Errorf("error calculating stats: %w", object.Err)
        }
        totalSize += object.Size
        objectCount++
    }

    stats["total_objects"] = objectCount
    stats["total_size_bytes"] = totalSize
    stats["bucket"] = s.bucket

    return stats, nil
}
