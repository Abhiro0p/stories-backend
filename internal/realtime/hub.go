package realtime

import (
    "sync"

    "github.com/google/uuid"
    "go.uber.org/zap"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
    // Registered clients
    clients map[*Client]bool

    // User ID to clients mapping for targeted messaging
    userClients map[uuid.UUID][]*Client

    // Inbound messages from the clients
    broadcast chan []byte

    // Register requests from the clients
    register chan *Client

    // Unregister requests from clients
    unregister chan *Client

    // Event broadcasting
    eventBroadcast chan *Event

    // Targeted events to specific users
    userEvents chan *UserEvent

    // Logger
    logger *zap.Logger

    // Mutex for thread-safe operations
    mutex sync.RWMutex
}

// UserEvent represents an event targeted to a specific user
type UserEvent struct {
    UserID uuid.UUID
    Event  *Event
}

// NewHub creates a new WebSocket hub
func NewHub(logger *zap.Logger) *Hub {
    return &Hub{
        clients:        make(map[*Client]bool),
        userClients:    make(map[uuid.UUID][]*Client),
        broadcast:      make(chan []byte),
        register:       make(chan *Client),
        unregister:     make(chan *Client),
        eventBroadcast: make(chan *Event),
        userEvents:     make(chan *UserEvent),
        logger:         logger.With(zap.String("component", "websocket_hub")),
    }
}

// Run starts the hub
func (h *Hub) Run() {
    h.logger.Info("Starting WebSocket hub")
    
    for {
        select {
        case client := <-h.register:
            h.registerClient(client)

        case client := <-h.unregister:
            h.unregisterClient(client)

        case message := <-h.broadcast:
            h.broadcastMessage(message)

        case event := <-h.eventBroadcast:
            h.broadcastEvent(event)

        case userEvent := <-h.userEvents:
            h.sendToUser(userEvent.UserID, userEvent.Event)
        }
    }
}

// Register registers a new client
func (h *Hub) Register(client *Client) {
    h.register <- client
}

// Unregister unregisters a client
func (h *Hub) Unregister(client *Client) {
    h.unregister <- client
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
    h.broadcast <- message
}

// BroadcastEvent sends an event to all connected clients
func (h *Hub) BroadcastEvent(event *Event) {
    h.eventBroadcast <- event
}

// SendToUser sends an event to a specific user
func (h *Hub) SendToUser(userID uuid.UUID, event *Event) {
    h.userEvents <- &UserEvent{
        UserID: userID,
        Event:  event,
    }
}

// BroadcastToFollowers sends an event to all followers of a user
func (h *Hub) BroadcastToFollowers(userID uuid.UUID, event *Event) {
    // This would typically require a database lookup to get followers
    // For now, we'll broadcast to all clients
    // TODO: Implement proper follower lookup and targeted broadcasting
    h.BroadcastEvent(event)
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
    h.mutex.Lock()
    defer h.mutex.Unlock()

    h.clients[client] = true

    // Add to user clients mapping
    userID := client.User.ID
    h.userClients[userID] = append(h.userClients[userID], client)

    h.logger.Info("Client registered",
        zap.String("user_id", userID.String()),
        zap.String("username", client.User.Username),
        zap.Int("total_clients", len(h.clients)),
        zap.Int("user_clients", len(h.userClients[userID])),
    )

    // Send welcome message
    welcomeEvent := &Event{
        Type: EventWelcome,
        Payload: map[string]interface{}{
            "message": "Connected to Stories Backend",
            "user":    client.User.ToResponse(),
        },
    }
    client.Send(welcomeEvent)
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
    h.mutex.Lock()
    defer h.mutex.Unlock()

    if _, ok := h.clients[client]; ok {
        delete(h.clients, client)
        close(client.send)

        // Remove from user clients mapping
        userID := client.User.ID
        userClientList := h.userClients[userID]
        for i, c := range userClientList {
            if c == client {
                h.userClients[userID] = append(userClientList[:i], userClientList[i+1:]...)
                break
            }
        }

        // Clean up empty user client list
        if len(h.userClients[userID]) == 0 {
            delete(h.userClients, userID)
        }

        h.logger.Info("Client unregistered",
            zap.String("user_id", userID.String()),
            zap.String("username", client.User.Username),
            zap.Int("total_clients", len(h.clients)),
            zap.Int("user_clients", len(h.userClients[userID])),
        )
    }
}

// broadcastMessage sends a raw message to all clients
func (h *Hub) broadcastMessage(message []byte) {
    h.mutex.RLock()
    defer h.mutex.RUnlock()

    for client := range h.clients {
        select {
        case client.send <- message:
        default:
            close(client.send)
            delete(h.clients, client)
        }
    }

    h.logger.Debug("Broadcasted message to all clients",
        zap.Int("client_count", len(h.clients)),
        zap.Int("message_size", len(message)),
    )
}

// broadcastEvent sends an event to all clients
func (h *Hub) broadcastEvent(event *Event) {
    h.mutex.RLock()
    defer h.mutex.RUnlock()

    for client := range h.clients {
        client.Send(event)
    }

    h.logger.Debug("Broadcasted event to all clients",
        zap.String("event_type", string(event.Type)),
        zap.Int("client_count", len(h.clients)),
    )
}

// sendToUser sends an event to a specific user
func (h *Hub) sendToUser(userID uuid.UUID, event *Event) {
    h.mutex.RLock()
    defer h.mutex.RUnlock()

    clients, exists := h.userClients[userID]
    if !exists {
        h.logger.Debug("No clients found for user",
            zap.String("user_id", userID.String()),
            zap.String("event_type", string(event.Type)),
        )
        return
    }

    for _, client := range clients {
        client.Send(event)
    }

    h.logger.Debug("Sent event to user",
        zap.String("user_id", userID.String()),
        zap.String("event_type", string(event.Type)),
        zap.Int("client_count", len(clients)),
    )
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
    h.mutex.RLock()
    defer h.mutex.RUnlock()

    return map[string]interface{}{
        "total_clients":     len(h.clients),
        "connected_users":   len(h.userClients),
        "clients_per_user":  h.getClientsPerUser(),
    }
}

// getClientsPerUser returns the distribution of clients per user
func (h *Hub) getClientsPerUser() map[string]int {
    distribution := make(map[string]int)
    
    for userID, clients := range h.userClients {
        key := userID.String()
        distribution[key] = len(clients)
    }
    
    return distribution
}

// GetConnectedUsers returns a list of connected user IDs
func (h *Hub) GetConnectedUsers() []uuid.UUID {
    h.mutex.RLock()
    defer h.mutex.RUnlock()

    users := make([]uuid.UUID, 0, len(h.userClients))
    for userID := range h.userClients {
        users = append(users, userID)
    }

    return users
}

// IsUserConnected checks if a user is currently connected
func (h *Hub) IsUserConnected(userID uuid.UUID) bool {
    h.mutex.RLock()
    defer h.mutex.RUnlock()

    clients, exists := h.userClients[userID]
    return exists && len(clients) > 0
}

// GetUserClientCount returns the number of clients for a user
func (h *Hub) GetUserClientCount(userID uuid.UUID) int {
    h.mutex.RLock()
    defer h.mutex.RUnlock()

    clients, exists := h.userClients[userID]
    if !exists {
        return 0
    }
    return len(clients)
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
    h.logger.Info("Shutting down WebSocket hub")

    h.mutex.Lock()
    defer h.mutex.Unlock()

    // Close all client connections
    for client := range h.clients {
        close(client.send)
    }

    // Clear all mappings
    h.clients = make(map[*Client]bool)
    h.userClients = make(map[uuid.UUID][]*Client)

    h.logger.Info("WebSocket hub shutdown complete")
}
