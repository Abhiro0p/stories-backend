package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector holds all Prometheus metrics
type Collector struct {
    HTTPRequestsTotal    *prometheus.CounterVec
    HTTPRequestDuration  *prometheus.HistogramVec
    HTTPRequestSize      *prometheus.HistogramVec
    HTTPResponseSize     *prometheus.HistogramVec
    
    DatabaseConnections  prometheus.Gauge
    DatabaseQueries      *prometheus.CounterVec
    DatabaseQueryDuration *prometheus.HistogramVec
    
    RedisOperations      *prometheus.CounterVec
    RedisConnectionPool  prometheus.Gauge
    
    StoriesCreated       prometheus.Counter
    StoriesViewed        prometheus.Counter
    ReactionsCreated     prometheus.Counter
    UsersRegistered      prometheus.Counter
    
    ActiveWebSocketConnections prometheus.Gauge
    WebSocketMessages          *prometheus.CounterVec
    
    WorkerJobsProcessed *prometheus.CounterVec
    WorkerJobDuration   *prometheus.HistogramVec
    WorkerQueueSize     *prometheus.GaugeVec
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
    return &Collector{
        HTTPRequestsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "http_requests_total",
                Help: "Total number of HTTP requests",
            },
            []string{"method", "endpoint", "status_code"},
        ),
        
        HTTPRequestDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "http_request_duration_seconds",
                Help: "HTTP request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "endpoint"},
        ),
        
        HTTPRequestSize: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "http_request_size_bytes",
                Help: "HTTP request size in bytes",
                Buckets: prometheus.ExponentialBuckets(1024, 2, 10),
            },
            []string{"method", "endpoint"},
        ),
        
        HTTPResponseSize: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "http_response_size_bytes",
                Help: "HTTP response size in bytes",
                Buckets: prometheus.ExponentialBuckets(1024, 2, 10),
            },
            []string{"method", "endpoint"},
        ),
        
        DatabaseConnections: promauto.NewGauge(
            prometheus.GaugeOpts{
                Name: "database_connections_active",
                Help: "Number of active database connections",
            },
        ),
        
        DatabaseQueries: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "database_queries_total",
                Help: "Total number of database queries",
            },
            []string{"operation", "table"},
        ),
        
        DatabaseQueryDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "database_query_duration_seconds",
                Help: "Database query duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"operation", "table"},
        ),
        
        RedisOperations: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "redis_operations_total",
                Help: "Total number of Redis operations",
            },
            []string{"operation", "status"},
        ),
        
        RedisConnectionPool: promauto.NewGauge(
            prometheus.GaugeOpts{
                Name: "redis_connection_pool_size",
                Help: "Redis connection pool size",
            },
        ),
        
        StoriesCreated: promauto.NewCounter(
            prometheus.CounterOpts{
                Name: "stories_created_total",
                Help: "Total number of stories created",
            },
        ),
        
        StoriesViewed: promauto.NewCounter(
            prometheus.CounterOpts{
                Name: "stories_viewed_total",
                Help: "Total number of story views",
            },
        ),
        
        ReactionsCreated: promauto.NewCounter(
            prometheus.CounterOpts{
                Name: "reactions_created_total",
                Help: "Total number of reactions created",
            },
        ),
        
        UsersRegistered: promauto.NewCounter(
            prometheus.CounterOpts{
                Name: "users_registered_total",
                Help: "Total number of users registered",
            },
        ),
        
        ActiveWebSocketConnections: promauto.NewGauge(
            prometheus.GaugeOpts{
                Name: "websocket_connections_active",
                Help: "Number of active WebSocket connections",
            },
        ),
        
        WebSocketMessages: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "websocket_messages_total",
                Help: "Total number of WebSocket messages",
            },
            []string{"type", "direction"},
        ),
        
        WorkerJobsProcessed: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "worker_jobs_processed_total",
                Help: "Total number of worker jobs processed",
            },
            []string{"job_type", "status"},
        ),
        
        WorkerJobDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "worker_job_duration_seconds",
                Help: "Worker job processing duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"job_type"},
        ),
        
        WorkerQueueSize: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "worker_queue_size",
                Help: "Number of jobs in worker queues",
            },
            []string{"queue_name"},
        ),
    }
}
