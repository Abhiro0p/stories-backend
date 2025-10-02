package realtime

import (
    "encoding/json"
    "time"
)

// EventType represents different types of real-time events
type EventType string

const (
    // Connection events
    EventWelcome      EventType = "welcome"
    EventError        EventType = "error"
    EventPing         EventType = "ping"
    EventPong         EventType = "pong"
    
    // Story events
    EventStoryCreated EventType = "story_created"
    EventStoryUpdated EventType = "story_updated"
    EventStoryDeleted EventType = "story_deleted"
    EventStoryViewed  EventType = "story_viewed"
    EventStoryExpired EventType = "story_expired"
    
    // Reaction events
    EventStoryReaction        EventType = "story_reaction"
    EventStoryReactionUpdated EventType = "story_reaction_updated"
    EventStoryReactionRemoved EventType = "story_reaction_removed"
    
    // User events
    EventUserFollowed   EventType = "user_followed"
    EventUserUnfollowed EventType = "user_unfollowed"
    EventUserUpdated    EventType = "user_updated"
    EventUserOnline     EventType = "user_online"
    EventUserOffline    EventType = "user_offline"
    
    // Typing events
    EventTyping EventType = "typing"
    
    // Notification events
    EventNotification EventType = "notification"
    
    // Activity events
    EventActivityUpdate EventType = "activity_update"
    
    // System events
    EventSystemMaintenance EventType = "system_maintenance"
    EventSystemAnnouncement EventType = "system_announcement"
)

// Event represents a real-time event
type Event struct {
    Type      EventType              `json:"type"`
    Payload   map[string]interface{} `json:"payload"`
    Timestamp int64                  `json:"timestamp"`
    ID        string                 `json:"id,omitempty"`
}

// NewEvent creates a new event with timestamp
func NewEvent(eventType EventType, payload map[string]interface{}) *Event {
    return &Event{
        Type:      eventType,
        Payload:   payload,
        Timestamp: time.Now().Unix(),
    }
}

// NewEventWithID creates a new event with ID and timestamp
func NewEventWithID(id string, eventType EventType, payload map[string]interface{}) *Event {
    return &Event{
        ID:        id,
        Type:      eventType,
        Payload:   payload,
        Timestamp: time.Now().Unix(),
    }
}

// ToJSON converts the event to JSON
func (e *Event) ToJSON() ([]byte, error) {
    return json.Marshal(e)
}

// FromJSON creates an event from JSON data
func FromJSON(data []byte) (*Event, error) {
    var event Event
    err := json.Unmarshal(data, &event)
    return &event, err
}

// StoryCreatedEvent creates a story created event
func StoryCreatedEvent(story interface{}, author interface{}) *Event {
    return NewEvent(EventStoryCreated, map[string]interface{}{
        "story":  story,
        "author": author,
    })
}

// StoryViewedEvent creates a story viewed event
func StoryViewedEvent(storyID string, viewer interface{}) *Event {
    return NewEvent(EventStoryViewed, map[string]interface{}{
        "story_id": storyID,
        "viewer":   viewer,
    })
}

// StoryReactionEvent creates a story reaction event
func StoryReactionEvent(storyID string, reaction interface{}, user interface{}) *Event {
    return NewEvent(EventStoryReaction, map[string]interface{}{
        "story_id": storyID,
        "reaction": reaction,
        "user":     user,
    })
}

// UserFollowedEvent creates a user followed event
func UserFollowedEvent(follower interface{}, followee interface{}) *Event {
    return NewEvent(EventUserFollowed, map[string]interface{}{
        "follower": follower,
        "followee": followee,
    })
}

// NotificationEvent creates a notification event
func NotificationEvent(notificationType string, title string, message string, data map[string]interface{}) *Event {
    payload := map[string]interface{}{
        "type":    notificationType,
        "title":   title,
        "message": message,
    }
    
    // Add additional data if provided
    for key, value := range data {
        payload[key] = value
    }
    
    return NewEvent(EventNotification, payload)
}

// ErrorEvent creates an error event
func ErrorEvent(errorCode string, errorMessage string) *Event {
    return NewEvent(EventError, map[string]interface{}{
        "error":   errorCode,
        "message": errorMessage,
    })
}

// WelcomeEvent creates a welcome event
func WelcomeEvent(user interface{}) *Event {
    return NewEvent(EventWelcome, map[string]interface{}{
        "message": "Connected to Stories Backend",
        "user":    user,
    })
}

// TypingEvent creates a typing indicator event
func TypingEvent(user interface{}, isTyping bool) *Event {
    return NewEvent(EventTyping, map[string]interface{}{
        "user":      user,
        "is_typing": isTyping,
    })
}

// SystemMaintenanceEvent creates a system maintenance event
func SystemMaintenanceEvent(message string, scheduledAt int64, duration int) *Event {
    return NewEvent(EventSystemMaintenance, map[string]interface{}{
        "message":      message,
        "scheduled_at": scheduledAt,
        "duration":     duration,
    })
}

// SystemAnnouncementEvent creates a system announcement event
func SystemAnnouncementEvent(title string, message string, priority string) *Event {
    return NewEvent(EventSystemAnnouncement, map[string]interface{}{
        "title":    title,
        "message":  message,
        "priority": priority,
    })
}

// UserOnlineEvent creates a user online event
func UserOnlineEvent(user interface{}) *Event {
    return NewEvent(EventUserOnline, map[string]interface{}{
        "user": user,
    })
}

// UserOfflineEvent creates a user offline event
func UserOfflineEvent(user interface{}) *Event {
    return NewEvent(EventUserOffline, map[string]interface{}{
        "user": user,
    })
}

// ActivityUpdateEvent creates an activity update event
func ActivityUpdateEvent(activityType string, data map[string]interface{}) *Event {
    payload := map[string]interface{}{
        "activity_type": activityType,
    }
    
    // Add activity data
    for key, value := range data {
        payload[key] = value
    }
    
    return NewEvent(EventActivityUpdate, payload)
}

// IsStoryEvent checks if the event is story-related
func (e *Event) IsStoryEvent() bool {
    storyEvents := []EventType{
        EventStoryCreated,
        EventStoryUpdated,
        EventStoryDeleted,
        EventStoryViewed,
        EventStoryExpired,
        EventStoryReaction,
        EventStoryReactionUpdated,
        EventStoryReactionRemoved,
    }
    
    for _, eventType := range storyEvents {
        if e.Type == eventType {
            return true
        }
    }
    
    return false
}

// IsUserEvent checks if the event is user-related
func (e *Event) IsUserEvent() bool {
    userEvents := []EventType{
        EventUserFollowed,
        EventUserUnfollowed,
        EventUserUpdated,
        EventUserOnline,
        EventUserOffline,
    }
    
    for _, eventType := range userEvents {
        if e.Type == eventType {
            return true
        }
    }
    
    return false
}

// IsSystemEvent checks if the event is system-related
func (e *Event) IsSystemEvent() bool {
    systemEvents := []EventType{
        EventSystemMaintenance,
        EventSystemAnnouncement,
    }
    
    for _, eventType := range systemEvents {
        if e.Type == eventType {
            return true
        }
    }
    
    return false
}

// GetPayloadValue safely gets a value from the event payload
func (e *Event) GetPayloadValue(key string) (interface{}, bool) {
    if e.Payload == nil {
        return nil, false
    }
    
    value, exists := e.Payload[key]
    return value, exists
}

// GetPayloadString safely gets a string value from the event payload
func (e *Event) GetPayloadString(key string) (string, bool) {
    if value, exists := e.GetPayloadValue(key); exists {
        if str, ok := value.(string); ok {
            return str, true
        }
    }
    return "", false
}

// GetPayloadInt safely gets an int value from the event payload
func (e *Event) GetPayloadInt(key string) (int, bool) {
    if value, exists := e.GetPayloadValue(key); exists {
        if i, ok := value.(int); ok {
            return i, true
        }
        if f, ok := value.(float64); ok {
            return int(f), true
        }
    }
    return 0, false
}

// GetPayloadBool safely gets a bool value from the event payload
func (e *Event) GetPayloadBool(key string) (bool, bool) {
    if value, exists := e.GetPayloadValue(key); exists {
        if b, ok := value.(bool); ok {
            return b, true
        }
    }
    return false, false
}

// Clone creates a copy of the event
func (e *Event) Clone() *Event {
    // Deep copy payload
    payloadCopy := make(map[string]interface{})
    for key, value := range e.Payload {
        payloadCopy[key] = value
    }
    
    return &Event{
        ID:        e.ID,
        Type:      e.Type,
        Payload:   payloadCopy,
        Timestamp: e.Timestamp,
    }
}

// String returns string representation of the event
func (e *Event) String() string {
    data, _ := e.ToJSON()
    return string(data)
}
