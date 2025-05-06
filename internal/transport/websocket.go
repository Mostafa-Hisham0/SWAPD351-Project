package transport

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"rtcs/internal/model"
	"rtcs/internal/service"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

const (
	maxMessageSize    = 4096 // 4KB
	writeWait         = 10 * time.Second
	pongWait          = 60 * time.Second
	pingPeriod        = (pongWait * 9) / 10
	maxConnections    = 10000
	messagesPerSecond = 5               // Rate limit: messages per second per client
	heartbeatInterval = 5 * time.Second // Reduced interval for more frequent updates
	statusInterval    = 5 * time.Second // How often to broadcast status updates
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		log.Printf("Accepting WebSocket connection from origin: %s", origin)
		return true
	},
}

type Client struct {
	conn     *websocket.Conn
	userID   string
	send     chan []byte
	handler  *WebSocketHandler
	limiter  *rate.Limiter
	closed   bool
	closeMux sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

type WebSocketHandler struct {
	clients        map[*Client]bool
	clientsMux     sync.RWMutex
	broadcast      chan []byte
	register       chan *Client
	unregister     chan *Client
	stats          *WebSocketStats
	shutdown       chan struct{}
	userIDs        map[string]*Client
	knownUsers     map[string]bool // Track all users who have ever connected
	statusService  *service.StatusService
	profileService *service.ProfileService
}

type WebSocketStats struct {
	ActiveConnections int64
	MessagesSent      int64
	MessagesReceived  int64
	Errors            int64
}

type WebSocketMessage struct {
	Type     string                        `json:"type"`
	UserID   string                        `json:"userId,omitempty"`
	Text     string                        `json:"text,omitempty"`
	Sender   string                        `json:"sender,omitempty"`
	Users    []string                      `json:"users,omitempty"`
	Status   string                        `json:"status,omitempty"`
	Statuses map[string]string             `json:"statuses,omitempty"`
	Profiles map[string]*model.UserProfile `json:"profiles,omitempty"`
}

func NewWebSocketHandler(statusService *service.StatusService, profileService *service.ProfileService) *WebSocketHandler {
	h := &WebSocketHandler{
		clients:        make(map[*Client]bool),
		broadcast:      make(chan []byte, 256),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		stats:          &WebSocketStats{},
		shutdown:       make(chan struct{}),
		userIDs:        make(map[string]*Client),
		knownUsers:     make(map[string]bool), // Initialize knownUsers map
		statusService:  statusService,
		profileService: profileService,
	}

	go h.run()
	go h.periodicStatusBroadcast()
	return h
}

func (h *WebSocketHandler) run() {
	for {
		select {
		case client := <-h.register:
			h.clientsMux.Lock()
			h.clients[client] = true
			h.clientsMux.Unlock()
			atomic.AddInt64(&h.stats.ActiveConnections, 1)

		case client := <-h.unregister:
			h.clientsMux.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				if client.userID != "" {
					// Remove from active clients but keep in knownUsers
					delete(h.userIDs, client.userID)

					if h.statusService != nil {
						ctx := context.Background()
						if err := h.statusService.SetUserOffline(ctx, client.userID); err != nil {
							log.Printf("[ERROR] Failed to set user %s offline: %v", client.userID, err)
						} else {
							log.Printf("[INFO] Set user %s offline during unregister", client.userID)
							// Broadcast the status change
							h.broadcastUserStatus(client.userID, "offline")
							// Update all clients with the latest user list
							h.broadcastUserList()
						}
					}
				}
			}
			h.clientsMux.Unlock()
			atomic.AddInt64(&h.stats.ActiveConnections, -1)

		case message := <-h.broadcast:
			h.clientsMux.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
					atomic.AddInt64(&h.stats.MessagesSent, 1)
				default:
					client.close()
					h.unregister <- client
				}
			}
			h.clientsMux.RUnlock()

		case <-h.shutdown:
			h.clientsMux.Lock()
			for client := range h.clients {
				client.close()
			}
			h.clientsMux.Unlock()
			return
		}
	}
}

// Periodically broadcast status updates to all clients
func (h *WebSocketHandler) periodicStatusBroadcast() {
	ticker := time.NewTicker(statusInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if h.statusService != nil {
				h.broadcastUserList()
			}
		case <-h.shutdown:
			return
		}
	}
}

func (c *Client) close() {
	c.closeMux.Lock()
	defer c.closeMux.Unlock()

	if !c.closed {
		c.closed = true
		c.cancel()
		c.conn.Close()
	}
}

func (c *Client) readPump() {
	defer func() {
		if c.userID != "" {
			// Ensure user is marked as offline when connection closes
			if c.handler.statusService != nil {
				ctx := context.Background()
				if err := c.handler.statusService.SetUserOffline(ctx, c.userID); err != nil {
					log.Printf("[ERROR] Failed to set user %s offline on disconnect: %v", c.userID, err)
				} else {
					log.Printf("[INFO] Set user %s offline on disconnect", c.userID)
				}
			}

			c.handler.broadcastMessage(WebSocketMessage{
				Type:   "user_leave",
				UserID: c.userID,
				Status: "offline",
			})

			// Update all clients with the latest user list
			c.handler.broadcastUserList()
		}

		c.handler.unregister <- c
		c.close()
		log.Printf("[INFO] WebSocket connection closed: %s", c.userID)
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
				log.Printf("[ERROR] WebSocket error: %v", err)
				atomic.AddInt64(&c.handler.stats.Errors, 1)
			}
			break
		}

		// Update user status on any activity
		if c.userID != "" && c.handler.statusService != nil {
			ctx := context.Background()
			if err := c.handler.statusService.SetUserOnline(ctx, c.userID); err != nil {
				log.Printf("[ERROR] Failed to refresh user status: %v", err)
			}
		}

		log.Printf("[DEBUG] Received message: %s", string(message))

		if !c.limiter.Allow() {
			log.Printf("[WARN] Rate limit exceeded for client %s", c.userID)
			continue
		}

		var wsMsg WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("[ERROR] Failed to parse WebSocket message: %v", err)
			c.handler.broadcastMessage(WebSocketMessage{
				Type: "message",
				Text: string(message),
			})
			continue
		}

		switch wsMsg.Type {
		case "user_join":
			log.Printf("[INFO] User joined: %s", wsMsg.UserID)
			c.handler.clientsMux.Lock()
			c.userID = wsMsg.UserID
			c.handler.userIDs[wsMsg.UserID] = c
			c.handler.knownUsers[wsMsg.UserID] = true // Add to known users
			c.handler.clientsMux.Unlock()

			// Ensure status is set in Redis
			if c.handler.statusService != nil {
				ctx := context.Background()
				if err := c.handler.statusService.SetUserOnline(ctx, wsMsg.UserID); err != nil {
					log.Printf("[ERROR] Failed to set user %s online: %v", wsMsg.UserID, err)
				} else {
					log.Printf("[INFO] Set user %s online in Redis", wsMsg.UserID)
				}
			}

			go c.startHeartbeat()

			// Make sure status is included in the broadcast
			wsMsg.Status = "online"
			c.handler.broadcastMessage(wsMsg)

			// Send updated user list to all clients
			c.handler.broadcastUserList()

		case "user_leave":
			log.Printf("[INFO] User left: %s", wsMsg.UserID)
			if c.userID != "" {
				c.handler.clientsMux.Lock()
				delete(c.handler.userIDs, c.userID)
				// Keep in knownUsers
				c.handler.clientsMux.Unlock()

				if c.handler.statusService != nil {
					ctx := context.Background()
					if err := c.handler.statusService.SetUserOffline(ctx, c.userID); err != nil {
						log.Printf("[ERROR] Failed to set user %s offline: %v", c.userID, err)
					} else {
						log.Printf("[INFO] Set user %s offline", c.userID)
					}
				}

				wsMsg.Status = "offline"
				c.handler.broadcastMessage(wsMsg)

				// Broadcast updated user list
				c.handler.broadcastUserList()
			}

		case "message":
			log.Printf("[INFO] Message from %s: %s", c.userID, wsMsg.Text)
			wsMsg.Sender = c.userID

			// Update status when sending message
			if c.handler.statusService != nil {
				ctx := context.Background()
				if err := c.handler.statusService.SetUserOnline(ctx, c.userID); err != nil {
					log.Printf("[ERROR] Failed to refresh user status: %v", err)
				}
			}

			c.handler.broadcastMessage(wsMsg)

		case "status_request":
			log.Printf("[INFO] Status request from %s", c.userID)
			c.handler.sendUserListWithStatus(c)

		case "profile_update":
			log.Printf("[INFO] Profile update from %s", c.userID)
			// Broadcast the profile update to all clients
			c.handler.broadcastMessage(wsMsg)

		case "heartbeat":
			log.Printf("[DEBUG] Heartbeat from %s", c.userID)
			if c.userID != "" && c.handler.statusService != nil {
				ctx := context.Background()
				if err := c.handler.statusService.SetUserOnline(ctx, c.userID); err != nil {
					log.Printf("[ERROR] Failed to refresh user status: %v", err)
				}
			}

		default:
			log.Printf("[WARN] Unknown message type: %s", wsMsg.Type)
			c.handler.broadcastMessage(wsMsg)
		}

		atomic.AddInt64(&c.handler.stats.MessagesReceived, 1)
	}
}

func (c *Client) startHeartbeat() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.userID != "" && c.handler.statusService != nil {
				log.Printf("[DEBUG] Sending server-side heartbeat for user %s", c.userID)
				ctx := context.Background()
				if err := c.handler.statusService.SetUserOnline(ctx, c.userID); err != nil {
					log.Printf("[ERROR] Failed to refresh user status: %v", err)
				}
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[ERROR] Failed to write message: %v", err)
				return
			}

			n := len(c.send)
			for i := 0; i < n; i++ {
				if err := c.conn.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
					log.Printf("[ERROR] Failed to write queued message: %v", err)
					return
				}
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[ERROR] Failed to write ping: %v", err)
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt64(&h.stats.ActiveConnections) >= maxConnections {
		log.Printf("[WARN] Connection rejected: maximum connections reached")
		http.Error(w, "Too many connections", http.StatusServiceUnavailable)
		return
	}

	log.Printf("[INFO] New WebSocket connection request from %s", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERROR] Failed to upgrade connection: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		conn:    conn,
		send:    make(chan []byte, 256),
		handler: h,
		limiter: rate.NewLimiter(rate.Limit(messagesPerSecond), 1),
		ctx:     ctx,
		cancel:  cancel,
	}

	log.Printf("[INFO] WebSocket connection established from %s", r.RemoteAddr)
	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (h *WebSocketHandler) broadcastMessage(msg WebSocketMessage) {
	if msg.Sender != "" && h.statusService != nil {
		ctx := context.Background()
		if err := h.statusService.SetUserOnline(ctx, msg.Sender); err != nil {
			log.Printf("[ERROR] Failed to refresh user status: %v", err)
		}
	}

	if msg.Type == "heartbeat" {
		return // Don't broadcast heartbeat messages
	}

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal message: %v", err)
		return
	}

	log.Printf("[DEBUG] Broadcasting message: %s", string(messageBytes))
	h.broadcast <- messageBytes

}

func (h *WebSocketHandler) broadcastUserStatus(userID, status string) {
	log.Printf("[INFO] Broadcasting status change: user %s is now %s", userID, status)
	msg := WebSocketMessage{
		Type:   "status_change",
		UserID: userID,
		Status: status,
	}
	h.broadcastMessage(msg)
}

// Broadcast user list to all clients
func (h *WebSocketHandler) broadcastUserList() {
	if h.statusService == nil {
		log.Printf("[WARN] Status service is nil, cannot broadcast user list")
		return
	}

	ctx := context.Background()

	// Get all user statuses from Redis
	statuses, err := h.statusService.GetAllUserStatuses(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to get user statuses: %v", err)
		return
	}

	// Create a list of all known users
	h.clientsMux.RLock()
	allUsers := make([]string, 0, len(h.knownUsers))
	userIDs := make([]uuid.UUID, 0, len(h.knownUsers))

	for userIDStr := range h.knownUsers {
		allUsers = append(allUsers, userIDStr)

		// Convert string to UUID for profile lookup
		userID, err := uuid.Parse(userIDStr)
		if err == nil {
			userIDs = append(userIDs, userID)
		}
	}
	h.clientsMux.RUnlock()

	// Get profiles for all users
	profiles := make(map[string]*model.UserProfile)
	if h.profileService != nil && len(userIDs) > 0 {
		userProfiles, err := h.profileService.GetProfiles(ctx, userIDs)
		if err != nil {
			log.Printf("[ERROR] Failed to get user profiles: %v", err)
		} else {
			// Convert UUID keys to string keys for JSON
			for id, profile := range userProfiles {
				profiles[id.String()] = profile
			}
		}
	}

	log.Printf("[INFO] Broadcasting user list with %d users, %d statuses, and %d profiles",
		len(allUsers), len(statuses), len(profiles))

	msg := WebSocketMessage{
		Type:     "user_list",
		Users:    allUsers,
		Statuses: statuses,
		Profiles: profiles, // Add profiles to the message
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal user list: %v", err)
		return
	}

	h.broadcast <- msgBytes
}

func (h *WebSocketHandler) sendUserListWithStatus(client *Client) {
	if h.statusService == nil {
		log.Printf("[WARN] Status service is nil, cannot send user list")
		return
	}

	ctx := context.Background()

	// Get all user statuses from Redis
	statuses, err := h.statusService.GetAllUserStatuses(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to get user statuses: %v", err)
		return
	}

	// Create a list of all known users
	h.clientsMux.RLock()
	allUsers := make([]string, 0, len(h.knownUsers))
	for userID := range h.knownUsers {
		allUsers = append(allUsers, userID)

		// If user is connected, ensure they're marked as online
		if _, isConnected := h.userIDs[userID]; isConnected {
			statuses[userID] = "online"
		} else if _, hasStatus := statuses[userID]; !hasStatus {
			// If user is not connected and has no status, mark as offline
			statuses[userID] = "offline"
		}
	}
	h.clientsMux.RUnlock()

	log.Printf("[INFO] Sending user list to client %s: %d users, %d statuses",
		client.userID, len(allUsers), len(statuses))

	msg := WebSocketMessage{
		Type:     "user_list",
		Users:    allUsers,
		Statuses: statuses,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal user list: %v", err)
		return
	}

	client.send <- msgBytes
}
