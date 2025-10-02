package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"

    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/worker"
    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// Build information (set via ldflags during build)
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
    GoVersion = "unknown"
)

func main() {
    // Initialize logger
    logger, err := zap.NewProduction()
    if err != nil {
        panic(err)
    }
    defer logger.Sync()

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        // ✅ FIXED: Use zap.Error() for error values
        logger.Fatal("Failed to load configuration", zap.Error(err))
    }

    // ✅ FIXED: Use proper zap field constructors
    logger.Info("Starting Stories Backend Worker",
        zap.String("version", Version),      // Use zap.String()
        zap.String("commit", Commit),        // Use zap.String()
        zap.String("build_time", BuildTime), // Use zap.String()
        zap.String("go_version", GoVersion), // Use zap.String()
        zap.String("environment", cfg.Environment), // Use zap.String()
    )

    // Create worker manager
    manager, err := worker.NewManager(cfg, logger)
    if err != nil {
        // ✅ FIXED: Use zap.Error() for error values
        logger.Fatal("Failed to create worker manager", zap.Error(err))
    }

    // Create context for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start workers
    go func() {
        if err := manager.Start(ctx); err != nil {
            // ✅ FIXED: Use zap.Error() for error values
            logger.Error("Worker manager failed", zap.Error(err))
        }
    }()

    logger.Info("Worker manager started successfully")

    // Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    <-quit
    logger.Info("Shutdown signal received")

    // Graceful shutdown with timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    if err := manager.Shutdown(shutdownCtx); err != nil {
        // ✅ FIXED: Use zap.Error() for error values
        logger.Error("Failed to shutdown worker manager gracefully", zap.Error(err))
    }

    logger.Info("Worker shutdown complete")
}
