package worker

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/storage"
    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// ExpirationWorker handles cleanup of expired stories
type ExpirationWorker struct {
    storyStore  storage.StoryStore
    redisClient *storage.RedisClient
    logger      *zap.Logger
    config      config.WorkerConfig
    
    isRunning bool
    stopCh    chan struct{}
}

// NewExpirationWorker creates a new story expiration worker
func NewExpirationWorker(
    storyStore storage.StoryStore,
    redisClient *storage.RedisClient,
    logger *zap.Logger,
    config config.WorkerConfig,
) *ExpirationWorker {
    return &ExpirationWorker{
        storyStore:  storyStore,
        redisClient: redisClient,
        logger:      logger.With(zap.String("worker", "expiration")),
        config:      config,
        stopCh:      make(chan struct{}),
    }
}

// IsRunning returns whether the worker is running
func (w *ExpirationWorker) IsRunning() bool {
    return w.isRunning
}

// Start starts the expiration worker
func (w *ExpirationWorker) Start(ctx context.Context) error {
    if w.isRunning {
        return fmt.Errorf("expiration worker is already running")
    }

    w.isRunning = true
    w.logger.Info("Starting expiration worker", 
        zap.Duration("interval", w.config.Interval),
        zap.Int("batch_size", w.config.BatchSize),
    )

    ticker := time.NewTicker(w.config.Interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            w.logger.Info("Expiration worker stopping due to context cancellation")
            w.isRunning = false
            return nil
            
        case <-w.stopCh:
            w.logger.Info("Expiration worker stopping")
            w.isRunning = false
            return nil
            
        case <-ticker.C:
            if err := w.processExpiredStories(ctx); err != nil {
                w.logger.Error("Failed to process expired stories", zap.Error(err))
            }
        }
    }
}

// Stop stops the expiration worker
func (w *ExpirationWorker) Stop(ctx context.Context) error {
    if !w.isRunning {
        return nil
    }

    w.logger.Info("Stopping expiration worker")
    close(w.stopCh)
    
    // Wait a bit for graceful shutdown
    select {
    case <-time.After(5 * time.Second):
        w.logger.Warn("Expiration worker stop timeout")
    case <-ctx.Done():
    }
    
    return nil
}

// processExpiredStories processes and deletes expired stories
func (w *ExpirationWorker) processExpiredStories(ctx context.Context) error {
    startTime := time.Now()
    
    // Get expired stories in batches
    offset := 0
    totalProcessed := 0
    
    for {
        stories, err := w.storyStore.GetExpired(ctx, w.config.BatchSize, offset)
        if err != nil {
            return fmt.Errorf("failed to get expired stories: %w", err)
        }
        
        if len(stories) == 0 {
            break
        }
        
        // Process each expired story
        for _, story := range stories {
            if err := w.processExpiredStory(ctx, story.ID); err != nil {
                w.logger.Error("Failed to process expired story", 
                    zap.String("story_id", story.ID.String()),
                    zap.Error(err),
                )
                continue
            }
            totalProcessed++
        }
        
        // If we got fewer stories than batch size, we're done
        if len(stories) < w.config.BatchSize {
            break
        }
        
        offset += w.config.BatchSize
        
        // Prevent infinite loops
        if offset > 10000 {
            w.logger.Warn("Too many expired stories, stopping batch processing", 
                zap.Int("processed", totalProcessed),
            )
            break
        }
        
        // Add small delay between batches to avoid overwhelming the database
        select {
        case <-ctx.Done():
            return nil
        case <-time.After(100 * time.Millisecond):
        }
    }
    
    duration := time.Since(startTime)
    
    if totalProcessed > 0 {
        w.logger.Info("Processed expired stories", 
            zap.Int("count", totalProcessed),
            zap.Duration("duration", duration),
        )
        
        // Update metrics
        w.recordMetrics(totalProcessed, duration)
    }
    
    return nil
}

// processExpiredStory processes a single expired story
func (w *ExpirationWorker) processExpiredStory(ctx context.Context, storyID uuid.UUID) error {
    // Soft delete the story
    if err := w.storyStore.Delete(ctx, storyID); err != nil {
        return fmt.Errorf("failed to delete expired story: %w", err)
    }
    
    // Clear related cache entries
    cacheKeys := []string{
        fmt.Sprintf("story:%s", storyID.String()),
        fmt.Sprintf("story_views:%s", storyID.String()),
        fmt.Sprintf("story_reactions:%s", storyID.String()),
        fmt.Sprintf("story_analytics:%s", storyID.String()),
    }
    
    if err := w.redisClient.DeleteMany(ctx, cacheKeys); err != nil {
        w.logger.Warn("Failed to clear cache for expired story", 
            zap.String("story_id", storyID.String()),
            zap.Error(err),
        )
    }
    
    w.logger.Debug("Processed expired story", 
        zap.String("story_id", storyID.String()),
    )
    
    return nil
}

// recordMetrics records expiration worker metrics
func (w *ExpirationWorker) recordMetrics(processed int, duration time.Duration) {
    // Store metrics in Redis for monitoring
    ctx := context.Background()
    timestamp := time.Now().Unix()
    
    metrics := map[string]interface{}{
        "processed_count": processed,
        "duration_ms":     duration.Milliseconds(),
        "timestamp":       timestamp,
        "worker":          "expiration",
    }
    
    key := fmt.Sprintf("metrics:expiration:%d", timestamp)
    if err := w.redisClient.Set(ctx, key, metrics, 86400); err != nil { // Keep for 24 hours
        w.logger.Warn("Failed to record metrics", zap.Error(err))
    }
}

// GetStats returns worker statistics
func (w *ExpirationWorker) GetStats() map[string]interface{} {
    return map[string]interface{}{
        "name":         "expiration",
        "running":      w.isRunning,
        "interval":     w.config.Interval.String(),
        "batch_size":   w.config.BatchSize,
        "last_run":     time.Now().Format(time.RFC3339),
    }
}

// ForceRun forces an immediate run of the expiration process
func (w *ExpirationWorker) ForceRun(ctx context.Context) error {
    if !w.isRunning {
        return fmt.Errorf("expiration worker is not running")
    }
    
    w.logger.Info("Force running expiration process")
    return w.processExpiredStories(ctx)
}
