package realtime

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/gorilla/websocket"
    "go.uber.org/zap"

    "github.com/Abhiro0p/stories-backend/internal/models"
)

const (
    // Time allowed to write a message to the peer
    writeWait = 10 * time.Second

    // Time allowed to read the next pong message from the peer
    pongWait = 60 * time.Second

    // Send pings to peer with this period. Must be less than pongWait
    pingPeriod = (pongWait * 9) / 10

    // Maximum message size allowed from peer
    maxMessageSize = 512
)

// Client is a middleman between the websocket connection and the hub
type Client struct {
    hub    *Hub
    conn   *websocket.Conn
    send   chan []byte
    User   *models.User
    logger *zap.Logger
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, user *models.User, logger *zap.Logger) *Client {
    return &Client{
        hub:    hub,
        conn:   conn,
        send:   make(chan []byte, 256),
        User:   user,
        logger: logger.With(zap.String("component", "websocket_client")),
    }
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
    defer func() {
        c.hub.Unregister(c)
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                c.logger.Error("WebSocket error",
                    zap.String("user_id", c.User.ID.String()),
                    zap.Error(err),
                )
            }
            break
        }

        // Handle incoming message
        c.handleMessage(message)
    }
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                // The hub closed the channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)

            // Add queued messages to the current websocket message
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.send)
            }

            if err := w.Close(); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

// Send sends an event to the client
func (c *Client) Send(event *Event) {
    data, err := json.Marshal(event)
    if err != nil {
        c.logger.Error("Failed to marshal event",
            zap.String("user_id", c.User.ID.String()),
            zap.String("event_type", string(event.Type)),
            zap.Error(err),
        )
        return
    }

    select {
    case c.send <- data:
    default:
        close(c.send)
        delete(c.hub.clients, c)
    }
}

// handleMessage handles incoming messages from the client
func (c *Client) handleMessage(message []byte) {
    var incomingEvent Event
    if err := json.Unmarshal(message, &incomingEvent); err != nil {
        c.logger.Warn("Failed to unmarshal incoming message",
            zap.String("user_id", c.User.ID.String()),
            zap.Error(err),
        )
        
        // Send error response
        errorEvent := &Event{
            Type: EventError,
            Payload: map[string]interface{}{
                "error":   "invalid_message",
                "message": "Invalid message format",
            },
        }
        c.Send(errorEvent)
        return
    }

    c.logger.Debug("Received message from client",
        zap.String("user_id", c.User.ID.String()),
        zap.String("event_type", string(incomingEvent.Type)),
    )

    // Handle different event types
    switch incomingEvent.Type {
    case EventPing:
        c.handlePing()
    case EventStoryView:
        c.handleStoryView(incomingEvent.Payload)
    case EventTyping:
        c.handleTyping(incomingEvent.Payload)
    default:
        c.logger.Warn("Unknown event type received",
            zap.String("user_id", c.User.ID.String()),
            zap.String("event_type", string(incomingEvent.Type)),
        )
        
        // Send error response
        errorEvent := &Event{
            Type: EventError,
            Payload: map[string]interface{}{
                "error":   "unknown_event_type",
                "message": "Unknown event type: " + string(incomingEvent.Type),
            },
        }
        c.Send(errorEvent)
    }
}

// handlePing handles ping messages
func (c *Client) handlePing() {
    pongEvent := &Event{
        Type: EventPong,
        Payload: map[string]interface{}{
            "timestamp": time.Now().Unix(),
        },
    }
    c.Send(pongEvent)
}

// handleStoryView handles story view events
func (c *Client) handleStoryView(payload map[string]interface{}) {
    storyID, ok := payload["story_id"].(string)
    if !ok {
        c.logger.Warn("Invalid story_id in story view event",
            zap.String("user_id", c.User.ID.String()),
        )
        return
    }

    c.logger.Info("Story view event received",
        zap.String("user_id", c.User.ID.String()),
        zap.String("story_id", storyID),
    )

    // TODO: Process story view (update database, send notifications, etc.)
}

// handleTyping handles typing indicator events
func (c *Client) handleTyping(payload map[string]interface{}) {
    isTyping, ok := payload["is_typing"].(bool)
    if !ok {
        c.logger.Warn("Invalid is_typing in typing event",
            zap.String("user_id", c.User.ID.String()),
        )
        return
    }

    c.logger.Debug("Typing event received",
        zap.String("user_id", c.User.ID.String()),
        zap.Bool("is_typing", isTyping),
    )

    // Broadcast typing indicator to relevant users
    typingEvent := &Event{
        Type: EventTyping,
        Payload: map[string]interface{}{
            "user":      c.User.ToResponse(),
            "is_typing": isTyping,
        },
    }
    
    // TODO: Broadcast to specific users (followers, friends, etc.)
    c.hub.BroadcastEvent(typingEvent)
}

// Close closes the client connection
func (c *Client) Close() {
    c.conn.Close()
}

// IsConnected checks if the client is still connected
func (c *Client) IsConnected() bool {
    return c.conn != nil
}

// GetRemoteAddr returns the remote address of the client
func (c *Client) GetRemoteAddr() string {
    if c.conn != nil {
        return c.conn.RemoteAddr().String()
    }
    return ""
}

// GetUserAgent returns the user agent of the client
func (c *Client) GetUserAgent() string {
    if c.conn != nil {
        return c.conn.Subprotocol()
    }
    return ""
}
