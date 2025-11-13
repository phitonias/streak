package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/phitonias/streak/internal/config"
	"github.com/phitonias/streak/internal/models"
	"github.com/phitonias/streak/internal/service"
	"github.com/phitonias/streak/internal/utils"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type Client struct {
	conn        *websocket.Conn
	send        chan []byte
	streamID    string
	user        *utils.Claims
	hub         *Hub
	mu          sync.Mutex
}

type Hub struct {
	clients      map[string]map[*Client]bool // streamID -> clients
	broadcast    chan *BroadcastMessage
	register     chan *Client
	unregister   chan *Client
	mu           sync.RWMutex
	chatService  *service.ChatService
	streamService *service.StreamService
}

type BroadcastMessage struct {
	StreamID string
	Message  []byte
}

type SocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type ChatMessagePayload struct {
	StreamID string `json:"streamId"`
	Message  string `json:"message"`
}

func NewHub(chatService *service.ChatService, streamService *service.StreamService) *Hub {
	return &Hub{
		clients:       make(map[string]map[*Client]bool),
		broadcast:     make(chan *BroadcastMessage, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		chatService:   chatService,
		streamService: streamService,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.streamID] == nil {
				h.clients[client.streamID] = make(map[*Client]bool)
			}
			h.clients[client.streamID][client] = true
			h.mu.Unlock()

			// Add viewer to stream
			streamID, _ := uuid.Parse(client.streamID)
			count, _ := h.streamService.AddViewer(streamID, client.user.UserID)

			// Broadcast viewer count update
			h.broadcastToStream(client.streamID, map[string]interface{}{
				"type": "viewer_count_update",
				"payload": map[string]interface{}{
					"streamId":    client.streamID,
					"viewerCount": count,
				},
			})

			log.Printf("Client registered: user=%s stream=%s", client.user.Username, client.streamID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.streamID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)

					// Remove viewer from stream
					streamID, _ := uuid.Parse(client.streamID)
					count, _ := h.streamService.RemoveViewer(streamID, client.user.UserID)

					// Broadcast viewer count update
					h.broadcastToStream(client.streamID, map[string]interface{}{
						"type": "viewer_count_update",
						"payload": map[string]interface{}{
							"streamId":    client.streamID,
							"viewerCount": count,
						},
					})

					if len(clients) == 0 {
						delete(h.clients, client.streamID)
					}
				}
			}
			h.mu.Unlock()

			log.Printf("Client unregistered: user=%s stream=%s", client.user.Username, client.streamID)

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[message.StreamID]
			h.mu.RUnlock()

			for client := range clients {
				select {
				case client.send <- message.Message:
				default:
					close(client.send)
					h.mu.Lock()
					delete(h.clients[message.StreamID], client)
					h.mu.Unlock()
				}
			}
		}
	}
}

func (h *Hub) broadcastToStream(streamID string, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.broadcast <- &BroadcastMessage{
		StreamID: streamID,
		Message:  data,
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(message []byte) {
	var msg SocketMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error parsing message: %v", err)
		return
	}

	switch msg.Type {
	case "send_message":
		c.handleChatMessage(msg.Payload)
	}
}

func (c *Client) handleChatMessage(payload interface{}) {
	data, _ := json.Marshal(payload)
	var chatPayload ChatMessagePayload
	if err := json.Unmarshal(data, &chatPayload); err != nil {
		log.Printf("Error parsing chat payload: %v", err)
		return
	}

	if chatPayload.Message == "" || len(chatPayload.Message) > 500 {
		return
	}

	// Create chat message
	chatMessage := &models.ChatMessage{
		StreamID:  c.streamID,
		UserID:    c.user.UserID.String(),
		Username:  c.user.Username,
		Message:   chatPayload.Message,
		Timestamp: time.Now(),
		IsDeleted: false,
	}

	// Save to database
	if err := c.hub.chatService.SaveMessage(chatMessage); err != nil {
		log.Printf("Error saving message: %v", err)
		return
	}

	// Broadcast to all clients in the stream
	c.hub.broadcastToStream(c.streamID, map[string]interface{}{
		"type":    "message",
		"payload": chatMessage,
	})
}

func HandleWebSocket(cfg *config.Config, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from query param
		token := c.Query("token")
		if token == "" {
			c.JSON(401, gin.H{"error": "No token provided"})
			return
		}

		// Validate token
		claims, err := utils.ValidateToken(token, cfg.JWT.Secret)
		if err != nil {
			c.JSON(401, gin.H{"error": "Invalid token"})
			return
		}

		// Get streamID from query
		streamID := c.Query("streamId")
		if streamID == "" {
			c.JSON(400, gin.H{"error": "Stream ID required"})
			return
		}

		// Upgrade connection
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// Create client
		client := &Client{
			conn:     conn,
			send:     make(chan []byte, 256),
			streamID: streamID,
			user:     claims,
			hub:      hub,
		}

		// Register client
		hub.register <- client

		// Send chat history
		messages, err := hub.chatService.GetRecentMessages(streamID, 50)
		if err == nil {
			historyData, _ := json.Marshal(map[string]interface{}{
				"type": "chat_history",
				"payload": map[string]interface{}{
					"streamId": streamID,
					"messages": messages,
				},
			})
			client.send <- historyData
		}

		// Start pumps
		go client.writePump()
		go client.readPump()
	}
}
