package ws

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	infraWS "Hospital-Referral-System/internal/infrastructure/ws"
	"Hospital-Referral-System/internal/pkg/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type JWTAuth interface {
	ValidateToken(tokenStr string) (*auth.TokenPayload, error)
}

type Handler struct {
	hub     *infraWS.Hub
	jwtAuth JWTAuth
	chatUC  iusecase.ChatUseCase
}

func NewHandler(hub *infraWS.Hub, jwtAuth JWTAuth, chatUC iusecase.ChatUseCase) *Handler {
	return &Handler{hub: hub, jwtAuth: jwtAuth, chatUC: chatUC}
}

// ServeWS handles the WebSocket upgrade request.
func (h *Handler) ServeWS(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token is required"})
		return
	}

	claims, err := h.jwtAuth.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	hospIDStr := ""
	if claims.HospID != nil {
		hospIDStr = claims.HospID.String()
	}

	deptIDStr := ""
	if claims.DeptID != nil {
		deptIDStr = claims.DeptID.String()
	}

	client := &infraWS.Client{
		UserID:       claims.UserID.String(),
		Role:         string(claims.Role),
		HospitalID:   hospIDStr,
		DepartmentID: deptIDStr,
		Conn:         conn,
		Send:         make(chan []byte, 256),
		Hub:          h.hub,
		OnChatMessage: func(ctx context.Context, senderID, receiverID uuid.UUID, referralID *uuid.UUID, content string) error {
			_, err := h.chatUC.SendMessage(ctx, senderID, receiverID, referralID, content)
			return err
		},
	}

	h.hub.Register(claims.UserID.String(), client)

	go client.WritePump()
	go client.ReadPump()
}

type RealJWTAuth struct{}

func (RealJWTAuth) ValidateToken(tokenStr string) (*auth.TokenPayload, error) {
	return auth.ValidateToken(tokenStr)
}

