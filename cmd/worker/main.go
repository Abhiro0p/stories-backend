package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/Abhiro0p/stories-backend/internal/worker"
    "github.com/Abhiro0p/stories-backend/pkg/config"
    "github.com/Abhiro0p/stories-backend/pkg/logger"
)

var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
    GoVersion = "unknown"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Initialize logger
    zapLogger, err := logger.New(cfg.LogLevel, cfg.LogFormat)
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer zapLogger.Sync()

    zapLogger.Info("Starting Stories Backend Worker", 
        "version", Version,
        "commit", Commit,
        "build_time", BuildTime,
        "go_version", GoVersion,
        "environment", cfg.Environment,
    )

    // Create worker manager
    workerManager, err := worker.NewManager(cfg, zapLogger)
    if err != nil {
        zapLogger.Fatal("Failed to create worker manager", "error", err)
    }

    // Create context for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start worker manager
    if err := workerManager.Start(ctx); err != nil {
        zapLogger.Fatal("Failed to start worker manager", "error", err)
    }

    zapLogger.Info("Worker manager started successfully")

    // Wait for interrupt signal to gracefully shutdown the worker
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    zapLogger.Info("Shutting down worker...")

    // Create shutdown context with timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    // Shutdown worker manager
    if err := workerManager.Shutdown(shutdownCtx); err != nil {
        zapLogger.Error("Worker manager shutdown error", "error", err)
    } else {
        zapLogger.Info("Worker manager stopped gracefully")
    }
}
