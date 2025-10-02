package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "runtime"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/auth"
    "github.com/Abhiro0p/stories-backend/internal/handlers"
    "github.com/Abhiro0p/stories-backend/internal/media"
    "github.com/Abhiro0p/stories-backend/internal/middleware"
    "github.com/Abhiro0p/stories-backend/internal/realtime"
    "github.com/Abhiro0p/stories-backend/internal/storage"
    "github.com/Abhiro0p/stories-backend/pkg/config"
    "github.com/Abhiro0p/stories-backend/pkg/logger"
    "github.com/Abhiro0p/stories-backend/pkg/metrics"
)

var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
    GoVersion = runtime.Version()
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

    // ✅ FIXED: Use proper zap field constructors
    zapLogger.Info("Starting Stories Backend API",
        zap.String("version", Version),
        zap.String("commit", Commit),
        zap.String("build_time", BuildTime),
        zap.String("go_version", GoVersion),
        zap.String("environment", cfg.Environment),
        zap.String("port", cfg.Port),
        zap.String("api_prefix", cfg.APIPrefix),
    )

    // Initialize metrics
    metricsCollector := metrics.NewCollector()

    // Initialize database
    db, err := storage.NewPostgresDB(cfg, zapLogger)
    if err != nil {
        
        zapLogger.Fatal("Failed to initialize database", zap.Error(err))
    }
    defer db.Close()

    // Initialize Redis
    redisClient, err := storage.NewRedisClient(cfg, zapLogger)
    if err != nil {
        zapLogger.Fatal("Failed to initialize Redis", zap.Error(err))
    }
    defer redisClient.Close()

    // Initialize storage layer
    userStore := storage.NewUserStore(db.DB(), redisClient, zapLogger)
    storyStore := storage.NewStoryStore(db.DB(), redisClient, zapLogger)
    followStore := storage.NewFollowStore(db.DB(), redisClient, zapLogger)
    viewStore := storage.NewViewStore(db.DB(), redisClient, zapLogger)
    reactionStore := storage.NewReactionStore(db.DB(), redisClient, zapLogger)

    zapLogger.Info("Storage layer initialized successfully")

    // Initialize auth service
    authService := auth.NewService(cfg, userStore, zapLogger)

    // Initialize media service
    mediaService, err := media.NewService(cfg, zapLogger)
    if err != nil {
        zapLogger.Fatal("Failed to initialize media service", zap.Error(err))
    }

    // Initialize WebSocket hub
    wsHub := realtime.NewHub(zapLogger)
    go wsHub.Run()

    zapLogger.Info("WebSocket hub started")

    // Setup Gin router
    if cfg.Environment == "production" {
        gin.SetMode(gin.ReleaseMode)
        zapLogger.Info("Gin set to release mode")
    }

    router := gin.New()

    // Global middleware
    router.Use(middleware.Logger(zapLogger))
    router.Use(middleware.Recovery(zapLogger))
    router.Use(middleware.CORS(cfg))
    router.Use(middleware.Metrics(metricsCollector))

    // Rate limiting middleware
    router.Use(middleware.RateLimit(redisClient, cfg.RateLimit))

    zapLogger.Info("Middleware configured successfully")

    // Health check endpoint
    healthHandler := handlers.NewHealthHandler(db, redisClient, zapLogger)
    router.GET("/health", healthHandler.Health)
    router.GET("/health/ready", healthHandler.Ready)
    router.GET("/health/live", healthHandler.Live)

    // Metrics endpoint
    router.GET("/metrics", gin.WrapH(promhttp.Handler()))

    // WebSocket endpoint
    wsHandler := handlers.NewWebSocketHandler(wsHub, authService, zapLogger)
    router.GET("/ws", wsHandler.HandleWebSocket)

    // API routes
    apiGroup := router.Group(cfg.APIPrefix)
    
    // Auth routes
    authHandler := handlers.NewAuthHandler(authService, zapLogger)
    authGroup := apiGroup.Group("/auth")
    {
        authGroup.POST("/signup", authHandler.Signup)
        authGroup.POST("/login", authHandler.Login)
        authGroup.POST("/refresh", authHandler.Refresh)
        authGroup.POST("/logout", auth.RequireAuth(authService), authHandler.Logout)
        authGroup.POST("/verify-email", authHandler.VerifyEmail)
        authGroup.POST("/forgot-password", authHandler.ForgotPassword)
        authGroup.POST("/reset-password", authHandler.ResetPassword)
        authGroup.PUT("/change-password", auth.RequireAuth(authService), authHandler.ChangePassword)
    }

    // Protected routes
    protected := apiGroup.Group("")
    protected.Use(auth.RequireAuth(authService))

    // User routes
    userHandler := handlers.NewUserHandler(userStore, followStore, zapLogger)
    userGroup := protected.Group("/users")
    {
        userGroup.GET("/me", userHandler.GetCurrentUser)
        userGroup.PUT("/me", userHandler.UpdateCurrentUser)
        userGroup.GET("/search", userHandler.SearchUsers)
        userGroup.GET("/:id", userHandler.GetUser)
        userGroup.POST("/:id/follow", userHandler.FollowUser)
        userGroup.DELETE("/:id/follow", userHandler.UnfollowUser)
        userGroup.GET("/:id/followers", userHandler.GetFollowers)
        userGroup.GET("/:id/following", userHandler.GetFollowing)
    }

    // Story routes
    storyHandler := handlers.NewStoryHandler(storyStore, viewStore, reactionStore, wsHub, zapLogger)
    storyGroup := protected.Group("/stories")
    {
        storyGroup.GET("", storyHandler.GetStories)
        storyGroup.POST("", storyHandler.CreateStory)
        storyGroup.GET("/:id", storyHandler.GetStory)
        storyGroup.PUT("/:id", storyHandler.UpdateStory)
        storyGroup.DELETE("/:id", storyHandler.DeleteStory)
        storyGroup.POST("/:id/view", storyHandler.ViewStory)
        storyGroup.GET("/:id/views", storyHandler.GetStoryViews)
        storyGroup.GET("/:id/reactions", storyHandler.GetStoryReactions)
        storyGroup.POST("/:id/reactions", storyHandler.AddReaction)
        storyGroup.PUT("/:id/reactions/:reaction_id", storyHandler.UpdateReaction)
        storyGroup.DELETE("/:id/reactions/:reaction_id", storyHandler.RemoveReaction)
    }

    // Media routes
    mediaHandler := handlers.NewMediaHandler(mediaService, zapLogger)
    mediaGroup := protected.Group("/media")
    {
        mediaGroup.POST("/upload-url", mediaHandler.GetUploadURL)
        mediaGroup.GET("/:key", mediaHandler.GetMedia)
        mediaGroup.DELETE("/:key", mediaHandler.DeleteMedia)
    }

    zapLogger.Info("Routes configured successfully",
        zap.Int("total_routes", len(router.Routes())),
    )

    // Create HTTP server
    server := &http.Server{
        Addr:    fmt.Sprintf(":%s", cfg.Port),
        Handler: router,
        
        // Timeouts
        ReadTimeout:       30 * time.Second,
        WriteTimeout:      30 * time.Second,
        IdleTimeout:       60 * time.Second,
        ReadHeaderTimeout: 10 * time.Second,
    }

    // Start server in goroutine
    serverErrCh := make(chan error, 1)
    go func() {
        zapLogger.Info("Starting HTTP server", 
            zap.String("port", cfg.Port),
            zap.String("address", server.Addr),
        )
        
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            zapLogger.Error("HTTP server failed", zap.Error(err))
            serverErrCh <- err
        }
    }()

    // Start metrics server
    var metricsServer *http.Server
    if cfg.PrometheusEnabled {
        metricsServer = &http.Server{
            Addr:    fmt.Sprintf(":%s", cfg.PrometheusPort),
            Handler: promhttp.Handler(),
        }

        go func() {
            zapLogger.Info("Starting metrics server", 
                zap.String("port", cfg.PrometheusPort),
                zap.String("endpoint", "/metrics"),
            )
            
            if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                zapLogger.Error("Metrics server error", zap.Error(err))
            }
        }()
    }

    zapLogger.Info("All services started successfully",
        zap.Bool("metrics_enabled", cfg.PrometheusEnabled),
        zap.Bool("websocket_enabled", true),
        zap.String("environment", cfg.Environment),
    )

    // Wait for interrupt signal or server error
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
    
    select {
    case sig := <-quit:
        zapLogger.Info("Shutdown signal received", 
            zap.String("signal", sig.String()),
        )
    case err := <-serverErrCh:
        zapLogger.Error("Server error, shutting down", zap.Error(err))
    }

    zapLogger.Info("Initiating graceful shutdown")
    shutdownStart := time.Now()

    // Create shutdown context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Shutdown HTTP server
    if err := server.Shutdown(ctx); err != nil {
        zapLogger.Error("Server forced to shutdown", zap.Error(err))
    }

    // Shutdown metrics server
    if metricsServer != nil {
        if err := metricsServer.Shutdown(ctx); err != nil {
            zapLogger.Error("Metrics server forced to shutdown", zap.Error(err))
        }
    }

    // Close WebSocket hub
    wsHub.Shutdown()

    zapLogger.Info("Server exited gracefully",
        zap.Duration("shutdown_duration", time.Since(shutdownStart)),
        zap.String("status", "success"),
    )
}
