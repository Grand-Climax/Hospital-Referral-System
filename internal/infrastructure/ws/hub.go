package ws

import (
	"encoding/json"
	"sync"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]bool // userID → connections
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]bool),
	}
}

// Register adds a client connection for a user.
func (h *Hub) Register(userID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]bool)
	}
	h.clients[userID][client] = true
}

// Unregister removes a client connection.
func (h *Hub) Unregister(userID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[userID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.Send)
			if len(clients) == 0 {
				delete(h.clients, userID)
			}
		}
	}
}

// SendToUser pushes a JSON message to all active connections of a user.
// Non‑blocking: slow clients are skipped.
func (h *Hub) SendToUser(userID string, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients[userID] {
		client.SafeSend(data)
	}
}

// BroadcastToRole pushes a message to all users with a given role in a hospital.
func (h *Hub) BroadcastToRole(role, hospitalID string, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, clients := range h.clients {
		for client := range clients {
			if client.Role == role && client.HospitalID == hospitalID {
				client.SafeSend(data)
			}
		}
	}
}
