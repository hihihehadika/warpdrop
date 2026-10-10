package ws

import (
	"sync"
	"time"
)

// Message represents a structured event transmitted over WebSocket connection.
type Message struct {
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	SenderID  string    `json:"sender_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Hub maintains the set of active connected clients and broadcasts messages.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	stopChan   chan struct{}
}

// NewHub creates and initializes a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stopChan:   make(chan struct{}),
	}
}

// Run starts the event loop handling client registration and broadcasting.
func (h *Hub) Run() {
	for {
		select {
		case <-h.stopChan:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// Don't echo clipboard message back to original sender
				if message.SenderID != "" && client.id == message.SenderID {
					continue
				}

				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Stop gracefully shuts down the hub and disconnects all clients.
func (h *Hub) Stop() {
	close(h.stopChan)
}

// Broadcast sends a message to all connected clients (except sender).
func (h *Hub) Broadcast(msg Message) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	h.broadcast <- msg
}

// ClientCount returns the number of currently active WebSocket connections.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
