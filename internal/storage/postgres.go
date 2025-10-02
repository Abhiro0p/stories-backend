package storage

import (
    "context"
    "fmt"
    "time"

    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/pkg/config"
)

// PostgresDB wraps the database connection
type PostgresDB struct {
    db     *sqlx.DB
    logger *zap.Logger
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.Config, logger *zap.Logger) (*PostgresDB, error) {
    db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
    db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.Database.MaxLifetime)

    // Test the connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    logger.Info("Connected to PostgreSQL successfully",
        zap.String("max_open_conns", fmt.Sprintf("%d", cfg.Database.MaxOpenConns)),
        zap.String("max_idle_conns", fmt.Sprintf("%d", cfg.Database.MaxIdleConns)),
        zap.Duration("max_lifetime", cfg.Database.MaxLifetime),
    )

    return &PostgresDB{
        db:     db,
        logger: logger.With(zap.String("component", "postgres")),
    }, nil
}

// DB returns the underlying database connection
func (p *PostgresDB) DB() *sqlx.DB {
    return p.db
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
    return p.db.Close()
}

// Health checks the database connection health
func (p *PostgresDB) Health(ctx context.Context) error {
    return p.db.PingContext(ctx)
}

// GetStats returns database connection statistics
func (p *PostgresDB) GetStats() SqlDbStats {
    stats := p.db.Stats()
    return SqlDbStats{
        OpenConnections: stats.OpenConnections,
        InUse:          stats.InUse,
        Idle:           stats.Idle,
        WaitCount:      stats.WaitCount,
        WaitDuration:   stats.WaitDuration,
        MaxIdleClosed:  stats.MaxIdleClosed,
        MaxIdleTimeClosed: stats.MaxIdleTimeClosed,
        MaxLifetimeClosed: stats.MaxLifetimeClosed,
    }
}

// SqlDbStats represents database connection statistics
type SqlDbStats struct {
    OpenConnections   int
    InUse            int
    Idle             int
    WaitCount        int64
    WaitDuration     time.Duration
    MaxIdleClosed    int64
    MaxIdleTimeClosed int64
    MaxLifetimeClosed int64
}

// WithTransaction executes a function within a database transaction
func (p *PostgresDB) WithTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
    tx, err := p.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }

    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            panic(r)
        }
    }()

    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            p.logger.Error("Failed to rollback transaction", zap.Error(rbErr))
        }
        return err
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}

// Migrate runs database migrations
func (p *PostgresDB) Migrate(ctx context.Context, migrationDir string) error {
    // This would typically use a migration library like golang-migrate
    // For now, we'll just log that migrations should be run separately
    p.logger.Info("Database migrations should be run using migration scripts",
        zap.String("migration_dir", migrationDir),
    )
    return nil
}
