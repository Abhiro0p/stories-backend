package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/auth"
    "github.com/Abhiro0p/stories-backend/internal/realtime"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
    hub         *realtime.Hub
    authService *auth.Service
    logger      *zap.Logger
    upgrader    websocket.Upgrader
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *realtime.Hub, authService *auth.Service, logger *zap.Logger) *WebSocketHandler {
    return &WebSocketHandler{
        hub:         hub,
        authService: authService,
        logger:      logger.With(zap.String("handler", "websocket")),
        upgrader: websocket.Upgrader{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
            CheckOrigin: func(r *http.Request) bool {
                // TODO: Implement proper origin checking for production
                return true
            },
        },
    }
}

// HandleWebSocket handles WebSocket connection upgrade and management
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
    // Get token from query parameter
    token := c.Query("token")
    if token == "" {
        h.logger.Warn("WebSocket connection attempted without token")
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "Token is required for WebSocket connection",
        })
        return
    }

    // Validate token and get user
    user, err := h.authService.ValidateToken(token)
    if err != nil {
        h.logger.Warn("WebSocket connection with invalid token", zap.Error(err))
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "Invalid or expired token",
        })
        return
    }

    // Upgrade HTTP connection to WebSocket
    conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        h.logger.Error("Failed to upgrade WebSocket connection", zap.Error(err))
        return
    }

    h.logger.Info("WebSocket connection established", 
        zap.String("user_id", user.ID.String()),
        zap.String("username", user.Username),
        zap.String("remote_addr", c.Request.RemoteAddr),
    )

    // Create and register client
    client := realtime.NewClient(h.hub, conn, user, h.logger)
    h.hub.Register(client)

    // Start client goroutines
    go client.WritePump()
    go client.ReadPump()
}
