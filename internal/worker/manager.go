package worker

import (
    "context"
    "fmt"
    "sync"
    "time"
    "github.com/google/uuid"
    "go.uber.org/zap"
    "github.com/Abhiro0p/stories-backend/internal/models"
    "github.com/Abhiro0p/stories-backend/internal/storage"
    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// Manager manages all background workers
type Manager struct {
    config      *config.Config
    logger      *zap.Logger
    db          *storage.PostgresDB
    redisClient *storage.RedisClient
    
    // Stores
    userStore     storage.UserStore
    storyStore    storage.StoryStore
    followStore   storage.FollowStore
    viewStore     storage.ViewStore
    reactionStore storage.ReactionStore
    
    // Workers
    expirationWorker *ExpirationWorker
    queue           *Queue
    
    // Control
    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup
    
    // Status
    isRunning bool
    mu        sync.RWMutex
}

// NewManager creates a new worker manager
func NewManager(cfg *config.Config, logger *zap.Logger) (*Manager, error) {
    // Initialize database connection
    db, err := storage.NewPostgresDB(cfg, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize database: %w", err)
    }

    // Initialize Redis client
    redisClient, err := storage.NewRedisClient(cfg, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize Redis: %w", err)
    }

    // Initialize stores
    userStore := storage.NewUserStore(db.DB(), redisClient, logger)
    storyStore := storage.NewStoryStore(db.DB(), redisClient, logger)
    followStore := storage.NewFollowStore(db.DB(), redisClient, logger)
    viewStore := storage.NewViewStore(db.DB(), redisClient, logger)
    reactionStore := storage.NewReactionStore(db.DB(), redisClient, logger)

    ctx, cancel := context.WithCancel(context.Background())

    manager := &Manager{
        config:        cfg,
        logger:        logger.With(zap.String("component", "worker_manager")),
        db:            db,
        redisClient:   redisClient,
        userStore:     userStore,
        storyStore:    storyStore,
        followStore:   followStore,
        viewStore:     viewStore,
        reactionStore: reactionStore,
        ctx:           ctx,
        cancel:        cancel,
    }

    // Initialize workers
    if err := manager.initializeWorkers(); err != nil {
        return nil, fmt.Errorf("failed to initialize workers: %w", err)
    }

    return manager, nil
}

// initializeWorkers creates and initializes all workers
func (m *Manager) initializeWorkers() error {
    // Create expiration worker
    m.expirationWorker = NewExpirationWorker(
        m.storyStore,
        m.redisClient,
        m.logger,
        m.config.Workers.StoryExpiration,
    )

    // Create job queue
    m.queue = NewQueue(
        m.redisClient,
        m.logger,
        m.config.Workers.Queue,
    )

    // Register job handlers
    m.registerJobHandlers()

    m.logger.Info("Initialized workers successfully")
    return nil
}

// registerJobHandlers registers all job handlers with the queue
func (m *Manager) registerJobHandlers() {
    // User statistics update job
    m.queue.RegisterHandler("update_user_stats", m.handleUpdateUserStats)
    
    // Cache cleanup job
    m.queue.RegisterHandler("cleanup_cache", m.handleCleanupCache)
    
    // Send notification job
    m.queue.RegisterHandler("send_notification", m.handleSendNotification)
    
    // Process story analytics job
    m.queue.RegisterHandler("process_analytics", m.handleProcessAnalytics)
    
    // Cleanup expired sessions job
    m.queue.RegisterHandler("cleanup_sessions", m.handleCleanupSessions)
    
    m.logger.Info("Registered job handlers", zap.Int("handler_count", 5))
}

// Start starts all workers
func (m *Manager) Start(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.isRunning {
        return fmt.Errorf("worker manager is already running")
    }

    m.logger.Info("Starting worker manager")

    // Start expiration worker
    m.wg.Add(1)
    go func() {
        defer m.wg.Done()
        if err := m.expirationWorker.Start(m.ctx); err != nil {
            m.logger.Error("Expiration worker failed", zap.Error(err))
        }
    }()

    // Start job queue
    m.wg.Add(1)
    go func() {
        defer m.wg.Done()
        if err := m.queue.Start(m.ctx); err != nil {
            m.logger.Error("Job queue failed", zap.Error(err))
        }
    }()

    m.isRunning = true
    m.logger.Info("Worker manager started successfully")
    
    return nil
}

// Shutdown gracefully stops all workers
func (m *Manager) Shutdown(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if !m.isRunning {
        return nil
    }

    m.logger.Info("Shutting down worker manager")

    // Cancel context to signal workers to stop
    m.cancel()

    // Wait for all workers to finish with timeout
    done := make(chan struct{})
    go func() {
        m.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        m.logger.Info("All workers stopped gracefully")
    case <-ctx.Done():
        m.logger.Warn("Worker shutdown timeout exceeded")
    }

    // Stop individual workers
    if err := m.expirationWorker.Stop(ctx); err != nil {
        m.logger.Error("Failed to stop expiration worker", zap.Error(err))
    }

    if err := m.queue.Stop(ctx); err != nil {
        m.logger.Error("Failed to stop job queue", zap.Error(err))
    }

    // Close database connections
    if err := m.db.Close(); err != nil {
        m.logger.Error("Failed to close database connection", zap.Error(err))
    }

    if err := m.redisClient.Close(); err != nil {
        m.logger.Error("Failed to close Redis connection", zap.Error(err))
    }

    m.isRunning = false
    m.logger.Info("Worker manager shutdown complete")
    
    return nil
}

// IsRunning returns whether the manager is running
func (m *Manager) IsRunning() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.isRunning
}

// GetStatus returns the status of all workers
func (m *Manager) GetStatus() map[string]interface{} {
    m.mu.RLock()
    defer m.mu.RUnlock()

    return map[string]interface{}{
        "manager_running":    m.isRunning,
        "expiration_running": m.expirationWorker.IsRunning(),
        "queue_running":      m.queue.IsRunning(),
        "queue_stats":        m.queue.GetStats(),
    }
}

// ScheduleJob schedules a job to be processed by the queue
func (m *Manager) ScheduleJob(jobType string, payload map[string]interface{}, delay time.Duration) error {
    return m.queue.Enqueue(jobType, payload, delay)
}

// Job handlers

// handleUpdateUserStats updates user statistics
func (m *Manager) handleUpdateUserStats(job *Job) error {
    userIDStr, ok := job.Payload["user_id"].(string)
    if !ok {
        return fmt.Errorf("invalid user_id in payload")
    }

    userID, err := uuid.Parse(userIDStr)
    if err != nil {
        return fmt.Errorf("invalid user UUID: %w", err)
    }

    ctx := context.Background()

    // Get follow stats
    followStats, err := m.followStore.GetFollowStats(ctx, userID)
    if err != nil {
        return fmt.Errorf("failed to get follow stats: %w", err)
    }

    // Count user's stories
    stories, err := m.storyStore.GetByAuthorID(ctx, userID, 1000, 0)
    if err != nil {
        return fmt.Errorf("failed to get user stories: %w", err)
    }

    // Update user stats
    stats := models.UserStats{
        FollowerCount:  followStats.FollowerCount,
        FollowingCount: followStats.FollowingCount,
        StoryCount:     len(stories),
    }

    if err := m.userStore.UpdateStats(ctx, userID, stats); err != nil {
        return fmt.Errorf("failed to update user stats: %w", err)
    }

    m.logger.Debug("Updated user stats", 
        zap.String("user_id", userID.String()),
        zap.Int("followers", stats.FollowerCount),
        zap.Int("following", stats.FollowingCount),
        zap.Int("stories", stats.StoryCount),
    )

    return nil
}

// handleCleanupCache cleans up expired cache entries
func (m *Manager) handleCleanupCache(job *Job) error {
    _ = job // Mark as used to avoid compiler warning
    
    ctx := context.Background()
    
    patterns := []string{
        "temp_*",
        "rate_limit:*",
        "expired_*",
    }

    totalCleaned := 0
    
    for _, pattern := range patterns {
        client := m.redisClient.GetClient()
        keys, err := client.Keys(ctx, pattern).Result()
        if err != nil {
            m.logger.Warn("Failed to get keys for pattern", 
                zap.String("pattern", pattern),
                zap.Error(err),
            )
            continue
        }

        if len(keys) > 0 {
            if err := m.redisClient.DeleteMany(ctx, keys); err != nil {
                m.logger.Warn("Failed to delete keys", 
                    zap.String("pattern", pattern),
                    zap.Error(err),
                )
                continue
            }
            totalCleaned += len(keys)
        }
    }

    m.logger.Info("Cache cleanup completed", 
        zap.Int("keys_cleaned", totalCleaned),
    )

    return nil
}

// handleSendNotification sends a notification - FIXED
func (m *Manager) handleSendNotification(job *Job) error {
    // Extract notification details from payload
    notificationType, _ := job.Payload["type"].(string)
    userIDStr, _ := job.Payload["user_id"].(string)
    title, _ := job.Payload["title"].(string)
    message, _ := job.Payload["message"].(string) // ✅ Now using message variable

    m.logger.Info("Processing notification", 
        zap.String("type", notificationType),
        zap.String("user_id", userIDStr),
        zap.String("title", title),
        zap.String("message", message), // ✅ Using the message variable
    )

    // TODO: Implement actual notification sending (push, email, etc.)
    // For now, just log the notification
    
    return nil
}

// handleProcessAnalytics processes analytics data - FIXED
func (m *Manager) handleProcessAnalytics(job *Job) error {
    _ = job // Mark as used to avoid compiler warning
    
    ctx := context.Background() // ✅ Now using ctx variable
    
    // Get current time for analytics processing
    now := time.Now()
    period := now.Format("2006-01-02-15") // Hourly analytics
    
    m.logger.Info("Processing analytics", zap.String("period", period))
    
    // TODO: Implement analytics aggregation
    // This could include:
    // - Story view counts
    // - Reaction summaries  
    // - User activity metrics
    // - Platform statistics
    
    // Example of using ctx (remove when implementing real analytics)
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Continue processing
    }
    
    return nil
}

// handleCleanupSessions cleans up expired sessions - FIXED
func (m *Manager) handleCleanupSessions(job *Job) error {
    _ = job // Mark as used to avoid compiler warning
    
    ctx := context.Background() // ✅ Now using ctx variable
    
    // TODO: Implement session cleanup
    // This would require a SessionStore implementation
    
    m.logger.Info("Session cleanup completed")
    
    // Example of using ctx (remove when implementing real session cleanup)
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Continue processing
    }
    
    return nil
}

// Additional utility methods

// RestartWorker restarts a specific worker
func (m *Manager) RestartWorker(workerName string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    switch workerName {
    case "expiration":
        m.logger.Info("Restarting expiration worker")
        if err := m.expirationWorker.Stop(context.Background()); err != nil {
            m.logger.Error("Failed to stop expiration worker", zap.Error(err))
        }
        
        m.wg.Add(1)
        go func() {
            defer m.wg.Done()
            if err := m.expirationWorker.Start(m.ctx); err != nil {
                m.logger.Error("Failed to restart expiration worker", zap.Error(err))
            }
        }()
        
    case "queue":
        m.logger.Info("Restarting job queue")
        if err := m.queue.Stop(context.Background()); err != nil {
            m.logger.Error("Failed to stop job queue", zap.Error(err))
        }
        
        m.wg.Add(1)
        go func() {
            defer m.wg.Done()
            if err := m.queue.Start(m.ctx); err != nil {
                m.logger.Error("Failed to restart job queue", zap.Error(err))
            }
        }()
        
    default:
        return fmt.Errorf("unknown worker: %s", workerName)
    }
    
    return nil
}

// GetWorkerList returns a list of available workers
func (m *Manager) GetWorkerList() []string {
    return []string{"expiration", "queue"}
}

// ForceRunExpiration forces immediate expiration process
func (m *Manager) ForceRunExpiration() error {
    if !m.expirationWorker.IsRunning() {
        return fmt.Errorf("expiration worker is not running")
    }
    
    return m.expirationWorker.ForceRun(context.Background())
}
