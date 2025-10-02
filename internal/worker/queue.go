package worker

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"  // ✅ Added missing import
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/storage"
    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// Job represents a background job
type Job struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Payload   map[string]interface{} `json:"payload"`
    CreatedAt time.Time              `json:"created_at"`
    RunAt     time.Time              `json:"run_at"`
    Attempts  int                    `json:"attempts"`
    MaxRetries int                   `json:"max_retries"`
}

// JobHandler is a function that processes a job
type JobHandler func(job *Job) error

// Queue manages background job processing
type Queue struct {
    redisClient *storage.RedisClient
    logger      *zap.Logger
    config      config.WorkerConfig
    
    handlers  map[string]JobHandler
    handlerMu sync.RWMutex
    
    isRunning bool
    stopCh    chan struct{}
    workers   int
    
    // Stats
    stats     *QueueStats
    statsMu   sync.RWMutex
}

// QueueStats holds queue statistics
type QueueStats struct {
    JobsProcessed   int64     `json:"jobs_processed"`
    JobsFailed      int64     `json:"jobs_failed"`
    JobsRetried     int64     `json:"jobs_retried"`
    LastProcessed   time.Time `json:"last_processed"`
    WorkersActive   int       `json:"workers_active"`
    QueueLength     int64     `json:"queue_length"`
}

// NewQueue creates a new job queue - UPDATED
func NewQueue(
    redisClient *storage.RedisClient,
    logger *zap.Logger,
    cfg config.WorkerConfig,
) *Queue {
    // Set defaults and handle concurrency vs workers
    workers := cfg.Workers
    if workers == 0 && cfg.Concurrency > 0 {
        workers = cfg.Concurrency // Use concurrency if workers not set
    }
    if workers == 0 {
        workers = 2 // Default fallback
    }
    
    if cfg.Interval == 0 {
        cfg.Interval = 5 * time.Second
    }

    return &Queue{
        redisClient: redisClient,
        logger:      logger.With(zap.String("component", "job_queue")),
        config:      cfg,
        handlers:    make(map[string]JobHandler),
        stopCh:      make(chan struct{}),
        workers:     workers,
        stats: &QueueStats{
            LastProcessed: time.Now(),
        },
    }
}

// RegisterHandler registers a job handler
func (q *Queue) RegisterHandler(jobType string, handler JobHandler) {
    q.handlerMu.Lock()
    defer q.handlerMu.Unlock()
    
    q.handlers[jobType] = handler
    q.logger.Debug("Registered job handler", zap.String("job_type", jobType))
}

// IsRunning returns whether the queue is running
func (q *Queue) IsRunning() bool {
    return q.isRunning
}

// Start starts the job queue processing
func (q *Queue) Start(ctx context.Context) error {
    if q.isRunning {
        return fmt.Errorf("job queue is already running")
    }

    if !q.config.Enabled {
        q.logger.Info("Job queue is disabled")
        return nil
    }

    q.isRunning = true
    q.logger.Info("Starting job queue", 
        zap.Int("workers", q.workers),
        zap.Duration("poll_interval", q.config.Interval),
    )

    // Start worker goroutines
    var wg sync.WaitGroup
    for i := 0; i < q.workers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            q.worker(ctx, workerID)
        }(i)
    }

    // Wait for all workers to finish
    wg.Wait()
    
    q.isRunning = false
    q.logger.Info("Job queue stopped")
    
    return nil
}

// Stop stops the job queue
func (q *Queue) Stop(ctx context.Context) error {
    if !q.isRunning {
        return nil
    }

    q.logger.Info("Stopping job queue")
    close(q.stopCh)
    
    return nil
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(jobType string, payload map[string]interface{}, delay time.Duration) error {
    job := &Job{
        ID:         uuid.New().String(),
        Type:       jobType,
        Payload:    payload,
        CreatedAt:  time.Now(),
        RunAt:      time.Now().Add(delay),
        Attempts:   0,
        MaxRetries: q.config.MaxRetries,
    }

    return q.enqueueJob(job)
}

// EnqueueJob adds a pre-created job to the queue
func (q *Queue) EnqueueJob(job *Job) error {
    return q.enqueueJob(job)
}

// enqueueJob adds a job to Redis - FIXED
func (q *Queue) enqueueJob(job *Job) error {
    ctx := context.Background()
    
    jobData, err := json.Marshal(job)
    if err != nil {
        return fmt.Errorf("failed to marshal job: %w", err)
    }

    // Add job to sorted set with run time as score - FIXED ZAdd syntax
    score := float64(job.RunAt.Unix())
    client := q.redisClient.GetClient()
    
    if err := client.ZAdd(ctx, "job_queue", redis.Z{  // ✅ Fixed: Use redis.Z struct
        Score:  score,
        Member: string(jobData),
    }).Err(); err != nil {
        return fmt.Errorf("failed to enqueue job: %w", err)
    }

    q.logger.Debug("Enqueued job", 
        zap.String("job_id", job.ID),
        zap.String("job_type", job.Type),
        zap.Time("run_at", job.RunAt),
    )

    return nil
}

// worker processes jobs from the queue
func (q *Queue) worker(ctx context.Context, workerID int) {
    logger := q.logger.With(zap.Int("worker_id", workerID))
    logger.Info("Starting job worker")

    ticker := time.NewTicker(q.config.Interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            logger.Info("Job worker stopping due to context cancellation")
            return
            
        case <-q.stopCh:
            logger.Info("Job worker stopping")
            return
            
        case <-ticker.C:
            if err := q.processNextJob(ctx, workerID); err != nil {
                logger.Error("Failed to process job", zap.Error(err))
            }
        }
    }
}

// processNextJob processes the next available job - FIXED
func (q *Queue) processNextJob(ctx context.Context, workerID int) error {
    // Get jobs that are ready to run
    now := time.Now().Unix()
    client := q.redisClient.GetClient()
    
    // Get the next job from the sorted set - FIXED ZRangeBy syntax
    result, err := client.ZRangeByScoreWithScores(ctx, "job_queue", &redis.ZRangeBy{
        Min:    "0",
        Max:    fmt.Sprintf("%d", now),
        Offset: 0,
        Count:  1,
    }).Result()
    
    if err != nil {
        return fmt.Errorf("failed to get jobs: %w", err)
    }
    
    if len(result) == 0 {
        return nil // No jobs to process
    }
    
    jobData := result[0].Member.(string)
    
    // Remove job from queue atomically
    removed, err := client.ZRem(ctx, "job_queue", jobData).Result()
    if err != nil {
        return fmt.Errorf("failed to remove job from queue: %w", err)
    }
    
    if removed == 0 {
        return nil // Job was already processed by another worker
    }
    
    // Parse job
    var job Job
    if err := json.Unmarshal([]byte(jobData), &job); err != nil {
        q.logger.Error("Failed to unmarshal job", zap.Error(err))
        return nil
    }
    
    // Process job
    return q.processJob(ctx, &job, workerID)
}

// processJob processes a single job - UPDATED
func (q *Queue) processJob(ctx context.Context, job *Job, workerID int) error {
    logger := q.logger.With(
        zap.String("job_id", job.ID),
        zap.String("job_type", job.Type),
        zap.Int("worker_id", workerID),
        zap.Int("attempt", job.Attempts+1),
    )
    
    logger.Info("Processing job")
    
    // Get handler
    q.handlerMu.RLock()
    handler, exists := q.handlers[job.Type]
    q.handlerMu.RUnlock()
    
    if !exists {
        logger.Error("No handler registered for job type")
        q.updateStats(false, false)
        return nil
    }
    
    job.Attempts++
    startTime := time.Now()
    
    // Process job with timeout
    timeout := q.config.Timeout
    if timeout == 0 {
        timeout = 5 * time.Minute
    }
    
    jobCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    done := make(chan error, 1)
    go func() {
        done <- handler(job)
    }()
    
    var err error
    select {
    case err = <-done:
    case <-jobCtx.Done():
        err = fmt.Errorf("job timeout after %s", timeout)
    }
    
    duration := time.Since(startTime)
    
    if err != nil {
        logger.Error("Job failed", 
            zap.Error(err),
            zap.Duration("duration", duration),
        )
        
        // Retry job if not exceeded max retries
        maxRetries := job.MaxRetries
        if maxRetries == 0 {
            maxRetries = q.config.MaxRetries
        }
        if maxRetries == 0 {
            maxRetries = 3
        }
        
        if job.Attempts < maxRetries {
            // Exponential backoff for retry delay
            retryDelay := time.Duration(job.Attempts) * time.Minute
            job.RunAt = time.Now().Add(retryDelay)
            
            if retryErr := q.enqueueJob(job); retryErr != nil {
                logger.Error("Failed to retry job", zap.Error(retryErr))
            } else {
                logger.Info("Job scheduled for retry", 
                    zap.Time("run_at", job.RunAt),
                    zap.Duration("delay", retryDelay),
                )
                q.updateStats(false, true)
                return nil
            }
        }
        
        q.updateStats(false, false)
        return err
    }
    
    logger.Info("Job completed successfully", zap.Duration("duration", duration))
    q.updateStats(true, false)
    
    return nil
}

// updateStats updates queue statistics
func (q *Queue) updateStats(success bool, retry bool) {
    q.statsMu.Lock()
    defer q.statsMu.Unlock()
    
    if success {
        q.stats.JobsProcessed++
    } else {
        q.stats.JobsFailed++
    }
    
    if retry {
        q.stats.JobsRetried++
    }
    
    q.stats.LastProcessed = time.Now()
}

// GetStats returns queue statistics
func (q *Queue) GetStats() *QueueStats {
    q.statsMu.RLock()
    defer q.statsMu.RUnlock()
    
    // Get current queue length
    ctx := context.Background()
    client := q.redisClient.GetClient()
    queueLength, _ := client.ZCard(ctx, "job_queue").Result()
    
    stats := *q.stats
    stats.QueueLength = queueLength
    stats.WorkersActive = q.workers
    
    return &stats
}

// GetQueueLength returns the current queue length
func (q *Queue) GetQueueLength() (int64, error) {
    ctx := context.Background()
    client := q.redisClient.GetClient()
    return client.ZCard(ctx, "job_queue").Result()
}

// ClearQueue removes all jobs from the queue
func (q *Queue) ClearQueue() error {
    ctx := context.Background()
    client := q.redisClient.GetClient()
    return client.Del(ctx, "job_queue").Err()
}

// ListJobs returns pending jobs (for debugging)
func (q *Queue) ListJobs(limit int) ([]*Job, error) {
    ctx := context.Background()
    client := q.redisClient.GetClient()
    
    result, err := client.ZRange(ctx, "job_queue", 0, int64(limit-1)).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to list jobs: %w", err)
    }
    
    jobs := make([]*Job, 0, len(result))
    for _, jobData := range result {
        var job Job
        if err := json.Unmarshal([]byte(jobData), &job); err != nil {
            q.logger.Warn("Failed to unmarshal job in list", zap.Error(err))
            continue
        }
        jobs = append(jobs, &job)
    }
    
    return jobs, nil
}

// PauseQueue pauses job processing
func (q *Queue) PauseQueue() {
    // TODO: Implement pause functionality
    q.logger.Info("Queue pause requested (not implemented)")
}

// ResumeQueue resumes job processing  
func (q *Queue) ResumeQueue() {
    // TODO: Implement resume functionality
    q.logger.Info("Queue resume requested (not implemented)")
}

// GetQueueInfo returns detailed queue information
func (q *Queue) GetQueueInfo() map[string]interface{} {
    stats := q.GetStats()
    
    return map[string]interface{}{
        "running":         q.isRunning,
        "enabled":         q.config.Enabled,
        "workers":         q.workers,
        "interval":        q.config.Interval.String(),
        "timeout":         q.config.Timeout.String(),
        "max_retries":     q.config.MaxRetries,
        "stats":           stats,
        "registered_handlers": len(q.handlers),
    }
}
