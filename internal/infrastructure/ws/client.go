package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	UserID        string
	Role          string
	HospitalID    string
	DepartmentID  string
	Conn          *websocket.Conn
	Send          chan []byte
	Hub           *Hub
	OnChatMessage func(ctx context.Context, senderID, receiverID uuid.UUID, referralID *uuid.UUID, content string) error
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 20 * time.Second
	maxMessageSize = 8192 // Increased to 8KB to support messages up to 5000 characters
)

// ReadPump reads messages from the WebSocket connection.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c.UserID, c)
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// Unmarshal WebSocket inbound frames
		var msg struct {
			Type       string  `json:"type"`
			ReceiverID string  `json:"receiver_id"`
			ReferralID *string `json:"referral_id"`
			Content    string  `json:"content"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Handle inbound chat frames
		if msg.Type == "chat" && c.OnChatMessage != nil {
			receiverUUID, err1 := uuid.Parse(msg.ReceiverID)
			var refUUID *uuid.UUID
			if msg.ReferralID != nil && *msg.ReferralID != "" {
				parsed, err2 := uuid.Parse(*msg.ReferralID)
				if err2 == nil {
					refUUID = &parsed
				}
			}
			senderUUID, err3 := uuid.Parse(c.UserID)

			if err1 != nil || err3 != nil {
				c.SafeSend([]byte(`{"type":"error","data":{"message":"Invalid user ID formatting"}}`))
				continue
			}

			if err := c.OnChatMessage(context.Background(), senderUUID, receiverUUID, refUUID, msg.Content); err != nil {
				c.SafeSend([]byte(`{"type":"error","data":{"message":"` + err.Error() + `"}}`))
				continue
			}
		}
	}
}

// WritePump writes messages from the Send channel to the WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SafeSend sends a message to the client's Send channel, recovering from closed channel panics.
func (c *Client) SafeSend(data []byte) {
	defer func() {
		if r := recover(); r != nil {
			// Recovered from write to a closed channel
		}
	}()
	select {
	case c.Send <- data:
	default:
		// slow client – skip
	}
}
