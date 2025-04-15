package transport

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

const (
	maxMessageSize = 4096 // 4KB
	writeWait     = 10 * time.Second
	pongWait      = 60 * time.Second
	pingPeriod    = (pongWait * 9) / 10
	maxConnections = 10000
	messagesPerSecond = 5 // Rate limit: messages per second per client
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for debugging purposes
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
}

type WebSocketHandler struct {
	clients    map[*Client]bool
	clientsMux sync.RWMutex
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stats      *WebSocketStats
	shutdown   chan struct{}
	userIDs    map[string]*Client // Map to track user IDs to clients
}

type WebSocketStats struct {
	ActiveConnections int64
	MessagesSent     int64
	MessagesReceived int64
	Errors           int64
}

func NewWebSocketHandler() *WebSocketHandler {
	h := &WebSocketHandler{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stats:      &WebSocketStats{},
		shutdown:   make(chan struct{}),
		userIDs:    make(map[string]*Client),
	}

	// Start the broadcast handler
	go h.run()

	return h
}

type WebSocketMessage struct {
	Type   string   `json:"type"`
	UserID string   `json:"userId,omitempty"`
	Text   string   `json:"text,omitempty"`
	Sender string   `json:"sender,omitempty"`
	Users  []string `json:"users,omitempty"` // Add users field for user list
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

func (c *Client) close() {
	c.closeMux.Lock()
	defer c.closeMux.Unlock()

	if !c.closed {
		c.closed = true
		c.conn.Close()
	}
}

func (c *Client) readPump() {
	defer func() {
		// If user was identified, notify others about leave
		if c.userID != "" {
			c.handler.broadcastMessage(WebSocketMessage{
				Type:   "user_leave",
				UserID: c.userID,
			})
			c.handler.clientsMux.Lock()
			delete(c.handler.userIDs, c.userID)
			c.handler.clientsMux.Unlock()
		}

		c.handler.unregister <- c
		c.close()
		log.Printf("WebSocket connection closed: %s", c.userID)
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
				log.Printf("WebSocket error: %v", err)
				atomic.AddInt64(&c.handler.stats.Errors, 1)
			}
			break
		}

		log.Printf("Received message: %s", string(message))

		// Apply rate limiting
		if !c.limiter.Allow() {
			log.Printf("Rate limit exceeded for client %s", c.userID)
			continue
		}

		// Parse message
		var wsMsg WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Error parsing WebSocket message: %v", err)
			// If not JSON, treat as regular message
			c.handler.broadcastMessage(WebSocketMessage{
				Type: "message",
				Text: string(message),
			})
			continue
		}

		// Handle different message types
		switch wsMsg.Type {
		case "user_join":
			log.Printf("User joined: %s", wsMsg.UserID)
			c.handler.clientsMux.Lock()
			c.userID = wsMsg.UserID
			c.handler.userIDs[wsMsg.UserID] = c
			c.handler.clientsMux.Unlock()
			// Broadcast the user join to all clients
			c.handler.broadcastMessage(wsMsg)
			// Send user list to the new client
			c.handler.sendUserList(c)
		case "user_leave":
			log.Printf("User left: %s", wsMsg.UserID)
			if c.userID != "" {
				c.handler.clientsMux.Lock()
				delete(c.handler.userIDs, c.userID)
				c.handler.clientsMux.Unlock()
				c.handler.broadcastMessage(wsMsg)
			}
		case "message":
			log.Printf("Message from %s: %s", c.userID, wsMsg.Text)
			wsMsg.Sender = c.userID
			c.handler.broadcastMessage(wsMsg)
		default:
			log.Printf("Unknown message type: %s", wsMsg.Type)
			c.handler.broadcastMessage(wsMsg)
		}

		atomic.AddInt64(&c.handler.stats.MessagesReceived, 1)
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
				// Channel was closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write message directly
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}

			// Process any pending messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				if err := c.conn.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
					log.Printf("Error writing queued message: %v", err)
					return
				}
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error writing ping: %v", err)
				return
			}
		}
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt64(&h.stats.ActiveConnections) >= maxConnections {
		log.Printf("Connection rejected: maximum connections reached")
		http.Error(w, "Too many connections", http.StatusServiceUnavailable)
		return
	}

	log.Printf("New WebSocket connection request from %s", r.RemoteAddr)
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		conn:    conn,
		send:    make(chan []byte, 256),
		handler: h,
		limiter: rate.NewLimiter(rate.Limit(messagesPerSecond), 1),
	}

	log.Printf("WebSocket connection established from %s", r.RemoteAddr)
	h.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

func (h *WebSocketHandler) broadcastMessage(msg WebSocketMessage) {
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	log.Printf("Broadcasting message: %s", string(messageBytes))
	h.broadcast <- messageBytes
}

func (h *WebSocketHandler) sendUserList(client *Client) {
	h.clientsMux.RLock()
	users := make([]string, 0, len(h.clients))
	for c := range h.clients {
		if c.userID != "" {
			users = append(users, c.userID)
		}
	}
	h.clientsMux.RUnlock()

	log.Printf("Sending user list: %v", users)
	msg := WebSocketMessage{
		Type:  "user_list",
		Users: users,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling user list: %v", err)
		return
	}

	client.send <- msgBytes
}
