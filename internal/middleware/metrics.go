package middleware

import (
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"

    "github.com/Abhiro0p/stories-backend/pkg/metrics"
)

// Metrics middleware collects HTTP metrics for Prometheus
func Metrics(collector *metrics.Collector) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        // Process request
        c.Next()

        // Collect metrics
        duration := time.Since(start).Seconds()
        status := strconv.Itoa(c.Writer.Status())
        
        // Update metrics
        collector.HTTPRequestsTotal.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            status,
        ).Inc()

        collector.HTTPRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(duration)

        collector.HTTPRequestSize.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(float64(c.Request.ContentLength))

        collector.HTTPResponseSize.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(float64(c.Writer.Size()))
    }
}

// DatabaseMetrics middleware for database operation metrics
func DatabaseMetrics(collector *metrics.Collector) gin.HandlerFunc {
    return func(c *gin.Context) {
        // This would be called from database operations
        // For now, it's a placeholder
        c.Next()
    }
}

// ActiveConnections tracks active WebSocket connections
var ActiveConnections = prometheus.NewGauge(prometheus.GaugeOpts{
    Name: "websocket_connections_active",
    Help: "Number of active WebSocket connections",
})

func init() {
    prometheus.MustRegister(ActiveConnections)
}
